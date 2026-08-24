import { useCallback, useEffect, useRef } from 'react';
import { resolveApiUrl } from '@/utils/urls';
import {
  findCurrentWord,
  validateTimingWords,
} from '@/utils/timing';
import { getLocalBookById } from '@/db/repositories/bookRepository';
import { usePlayback } from '@/hooks/usePlayback';
import { useReadingProgress } from '@/hooks/useReadingProgress';
import type { PlaybackStatus } from '@/services/audio/types';
import {
  getSavedProgress,
  loadPlaylist,
  markBookCompletedInDb,
  resolveResumeLocation,
  syncBookFromBackend,
  type ChunkRef,
} from '@/services/reader/readerService';
import { timingService } from '@/services/reader/timingService';
import { useBookStore } from '@/store/bookStore';
import { useReaderStore } from '@/store/readerStore';

function errorMessage(error: unknown): string {
  return error instanceof Error ? error.message : String(error);
}

/**
 * Coordinates the reading session: chapter/chunk loading, timing,
 * playback, word synchronization, chunk transitions and progress
 * persistence. UI components stay thin; this hook is the only place
 * where these concerns meet.
 */
export function useReader(bookId: string | null) {
  const playlistRef = useRef<ChunkRef[]>([]);
  const currentIndexRef = useRef(0);
  const wordIndexRef = useRef(-1);
  const lastValidWordIndexRef = useRef(0);
  const transitioningRef = useRef(false);
  const { saveThrottled, saveNow } = useReadingProgress(bookId);
  const saveThrottledRef = useRef(saveThrottled);
  saveThrottledRef.current = saveThrottled;
  const saveNowRef = useRef(saveNow);
  saveNowRef.current = saveNow;

  const currentRef = useCallback(
    (): ChunkRef | undefined => playlistRef.current[currentIndexRef.current],
    [],
  );

  const persistCurrent = useCallback(
    (force: boolean): void => {
      const ref = currentRef();
      const time = audioCommandsRef.current.getCurrentTime();
      if (!ref || time <= 0) return;
      const input = {
        ref,
        positionSeconds: time,
        wordIndex: lastValidWordIndexRef.current,
      };
      if (force) {
        saveNowRef.current(input);
      } else {
        saveThrottledRef.current(input);
      }
    },
    [currentRef],
  );

  const applyChunk = useCallback(
    async (ref: ChunkRef, wasPlaying: boolean): Promise<void> => {
      const store = useReaderStore.getState();

      const timing = await timingService.load(ref.timingUrl);
      validateTimingWords(timing.words);

      store.setChapter({
        chapterId: ref.chapterId,
        chapterIndex: ref.chapterIndex,
        chapterTitle: ref.chapterTitle,
      });
      store.setChunk({
        chunkId: ref.chunkId,
        chunkIndex: ref.chunkIndex,
        text: timing.text,
        words: timing.words,
      });
      store.setDuration(ref.duration ?? 0);

      await audioCommandsRef.current.load(resolveApiUrl(ref.audioUrl));

      wordIndexRef.current = -1;
      lastValidWordIndexRef.current = 0;
      store.setCurrentWordIndex(-1);

      if (wasPlaying) {
        await audioCommandsRef.current.play();
      }

      const next = playlistRef.current[currentIndexRef.current + 1];
      if (next) {
        timingService.preload(next.timingUrl);
      }
    },
    [],
  );

  /**
   * Chunk transition: persist progress for the next chunk, swap timing
   * + audio, reset the word cursor and continue seamlessly.
   */
  const advanceChunk = useCallback(async (): Promise<void> => {
    if (transitioningRef.current || !bookId) {
      return;
    }
    transitioningRef.current = true;

    try {
      const playlist = playlistRef.current;
      const nextIndex = currentIndexRef.current + 1;
      const finishedRef = currentRef();
      if (!finishedRef) {
        return;
      }

      const store = useReaderStore.getState();
      const wasPlaying = store.isPlaying;

      if (nextIndex >= playlist.length) {
        markBookCompletedInDb(bookId, finishedRef);
        await audioCommandsRef.current.stop();
        store.pause();
        store.setCurrentWordIndex(-1);
        store.setCurrentTime(0);
        store.setReaderStatus('completed');
        const bookStore = useBookStore.getState();
        bookStore.markCompleted();
        bookStore.setHasProgress(true);
        return;
      }

      currentIndexRef.current = nextIndex;
      const next = playlist[nextIndex];

      // Progress points at the new chunk from the moment it starts.
      saveNowRef.current({
        ref: next,
        positionSeconds: 0,
        wordIndex: 0,
      });

      store.setReaderStatus('loading');
      try {
        await applyChunk(next, wasPlaying);
        useReaderStore.getState().setReaderStatus(wasPlaying ? 'playing' : 'paused');
      } catch (error) {
        useReaderStore.getState().setError(
          `Unable to load the next section: ${errorMessage(error)}`,
        );
      }
    } finally {
      transitioningRef.current = false;
    }
  }, [applyChunk, bookId, currentRef]);

  const advanceChunkRef = useRef(advanceChunk);
  advanceChunkRef.current = advanceChunk;

  // --- audio status → word synchronization -------------------------------
  const handleStatusUpdate = useCallback((update: PlaybackStatus): void => {
    const store = useReaderStore.getState();

    if (update.didJustFinish) {
      void advanceChunkRef.current();
      return;
    }

    const rate = store.playbackRate > 0 ? store.playbackRate : 1;
    // Timing files correspond to 1x speed: scale audio time into timing space.
    const scaledTime = update.currentTime / rate;

    store.setCurrentTime(scaledTime);
    if (update.duration > 0) {
      store.setDuration(update.duration);
    }

    const words = store.words;
    if (words.length === 0) {
      return;
    }

    const index = findCurrentWord(words, scaledTime, wordIndexRef.current);
    wordIndexRef.current = index;
    if (index >= 0) {
      lastValidWordIndexRef.current = index;
    }
    if (index !== store.currentWordIndex) {
      store.setCurrentWordIndex(index);
    }

    persistCurrent(false);
  }, [persistCurrent]);

  const audioCommands = usePlayback(handleStatusUpdate);
  const audioCommandsRef = useRef(audioCommands);
  audioCommandsRef.current = audioCommands;

  // --- session restore ----------------------------------------------------
  useEffect(() => {
    let cancelled = false;

    const initialize = async (): Promise<void> => {
      const store = useReaderStore.getState();
      store.reset();
      store.setBookId(bookId);

      if (!bookId) {
        store.setError('No book selected.');
        return;
      }

      store.setReaderStatus('loading');

      try {
        let playlist = loadPlaylist(bookId);
        if (playlist.length === 0) {
          const localBook = getLocalBookById(bookId);
          if (!localBook) {
            throw new Error('Unable to load this book.');
          }
          await syncBookFromBackend(bookId);
          playlist = loadPlaylist(bookId);
        }
        if (playlist.length === 0) {
          throw new Error('This audiobook has no playable audio yet.');
        }
        if (cancelled) {
          return;
        }

        playlistRef.current = playlist;

        const progress = getSavedProgress(bookId);
        const location = resolveResumeLocation(playlist, progress);

        if (location.completed) {
          const finalRef = playlist[playlist.length - 1];
          currentIndexRef.current = playlist.length - 1;
          const timing = await timingService.load(finalRef.timingUrl);
          if (cancelled) return;
          store.setChapter({
            chapterId: finalRef.chapterId,
            chapterIndex: finalRef.chapterIndex,
            chapterTitle: finalRef.chapterTitle,
          });
          store.setChunk({
            chunkId: finalRef.chunkId,
            chunkIndex: finalRef.chunkIndex,
            text: timing.text,
            words: timing.words,
          });
          store.setDuration(finalRef.duration ?? 0);
          store.setReaderStatus('completed');
          return;
        }

        const ref = playlist[location.index];
        currentIndexRef.current = location.index;

        const timing = await timingService.load(ref.timingUrl);
        validateTimingWords(timing.words);
        if (cancelled) {
          return;
        }

        store.setChapter({
          chapterId: ref.chapterId,
          chapterIndex: ref.chapterIndex,
          chapterTitle: ref.chapterTitle,
        });
        store.setChunk({
          chunkId: ref.chunkId,
          chunkIndex: ref.chunkIndex,
          text: timing.text,
          words: timing.words,
        });
        store.setDuration(ref.duration ?? 0);

        await audioCommandsRef.current.load(resolveApiUrl(ref.audioUrl));
        if (cancelled) {
          return;
        }

        if (location.positionSeconds > 0) {
          await audioCommandsRef.current.seek(location.positionSeconds);
          store.setCurrentTime(location.positionSeconds / (store.playbackRate || 1));
        }

        const restoredWordIndex = findCurrentWord(
          timing.words,
          location.positionSeconds,
          location.wordIndex,
        );
        wordIndexRef.current = restoredWordIndex;
        if (restoredWordIndex >= 0) {
          lastValidWordIndexRef.current = restoredWordIndex;
        } else {
          lastValidWordIndexRef.current =
            location.wordIndex < timing.words.length ? location.wordIndex : 0;
        }
        store.setCurrentWordIndex(restoredWordIndex);

        store.setReaderStatus('paused');

        const next = playlist[location.index + 1];
        if (next) {
          timingService.preload(next.timingUrl);
        }
      } catch (error) {
        if (!cancelled) {
          useReaderStore
            .getState()
            .setError(errorMessage(error));
        }
      }
    };

    void initialize();

    return () => {
      cancelled = true;
    };
  }, [bookId]);

  // --- actions --------------------------------------------------------------
  const play = useCallback(async (): Promise<void> => {
    await audioCommandsRef.current.play();
    useReaderStore.getState().play();
  }, []);

  const pause = useCallback(async (): Promise<void> => {
    await audioCommandsRef.current.pause();
    useReaderStore.getState().pause();
    persistCurrent(true);
  }, [persistCurrent]);

  const togglePlayback = useCallback((): void => {
    if (useReaderStore.getState().isPlaying) {
      void pause();
    } else {
      void play();
    }
  }, [pause, play]);

  const seekBy = useCallback(
    async (deltaSeconds: number): Promise<void> => {
      const commands = audioCommandsRef.current;
      const target = Math.max(0, commands.getCurrentTime() + deltaSeconds);
      await commands.seek(target);

      const store = useReaderStore.getState();
      const rate = store.playbackRate > 0 ? store.playbackRate : 1;
      const scaledTime = target / rate;
      const index = findCurrentWord(store.words, scaledTime, -1);
      wordIndexRef.current = index;
      if (index >= 0) {
        lastValidWordIndexRef.current = index;
      }
      store.setCurrentTime(scaledTime);
      store.setCurrentWordIndex(index);
      persistCurrent(true);
    },
    [persistCurrent],
  );

  const changeRate = useCallback(async (rate: number): Promise<void> => {
    await audioCommandsRef.current.setRate(rate);
    useReaderStore.getState().setPlaybackRate(rate);
  }, []);

  /** Stop behaviour: persist, stop audio, release resources, reset state. */
  const exit = useCallback(async (): Promise<void> => {
    persistCurrent(true);
    try {
      await audioCommandsRef.current.stop();
    } finally {
      audioCommandsRef.current.release();
      timingService.clearCache();
      useReaderStore.getState().reset();
    }
  }, [persistCurrent]);

  return { play, pause, togglePlayback, seekBy, changeRate, exit };
}
