import { create } from 'zustand';
import type { WordTiming } from '@/schemas/timing';

export type ReaderStatus =
  | 'idle'
  | 'loading'
  | 'ready'
  | 'playing'
  | 'paused'
  | 'completed'
  | 'error';

/**
 * Session state for the reader. Only the *current* chunk's words are held
 * here (the working set of one screen); full book contents and complete
 * timing files stay in services/SQLite.
 */
interface ReaderState {
  bookId: string | null;
  chapterId: string | null;
  chapterIndex: number;
  chapterTitle: string | null;
  chunkId: string | null;
  /** Exact text supplied by the backend timing file. */
  chunkText: string | null;
  chunkIndex: number;
  words: WordTiming[];
  currentWordIndex: number;
  currentTime: number;
  duration: number;
  isPlaying: boolean;
  playbackRate: number;
  readerStatus: ReaderStatus;
  error: string | null;
  setBookId: (bookId: string | null) => void;
  setChapter: (chapter: {
    chapterId: string;
    chapterIndex: number;
    chapterTitle: string;
  }) => void;
  setChunk: (chunk: {
    chunkId: string;
    chunkIndex: number;
    text: string;
    words: WordTiming[];
  }) => void;
  setCurrentWordIndex: (index: number) => void;
  setCurrentTime: (time: number) => void;
  setDuration: (duration: number) => void;
  play: () => void;
  pause: () => void;
  setPlaybackRate: (rate: number) => void;
  setReaderStatus: (status: ReaderStatus) => void;
  setError: (error: string | null) => void;
  reset: () => void;
}

const initialState = {
  bookId: null,
  chapterId: null,
  chapterIndex: 0,
  chapterTitle: null as string | null,
  chunkId: null,
  chunkText: null as string | null,
  chunkIndex: 0,
  words: [] as WordTiming[],
  currentWordIndex: -1,
  currentTime: 0,
  duration: 0,
  isPlaying: false,
  playbackRate: 1,
  readerStatus: 'idle' as ReaderStatus,
  error: null as string | null,
};

export const useReaderStore = create<ReaderState>((set) => ({
  ...initialState,
  setBookId: (bookId) => set({ bookId }),
  setChapter: ({ chapterId, chapterIndex, chapterTitle }) =>
    set({ chapterId, chapterIndex, chapterTitle }),
  setChunk: ({ chunkId, chunkIndex, text, words }) =>
    set({
      chunkId,
      chunkIndex,
      chunkText: text,
      words,
      currentWordIndex: -1,
      currentTime: 0,
    }),
  setCurrentWordIndex: (currentWordIndex) => set({ currentWordIndex }),
  setCurrentTime: (currentTime) => set({ currentTime }),
  setDuration: (duration) => set({ duration }),
  play: () => set({ isPlaying: true, readerStatus: 'playing' }),
  pause: () => set({ isPlaying: false, readerStatus: 'paused' }),
  setPlaybackRate: (playbackRate) => set({ playbackRate }),
  setReaderStatus: (readerStatus) => set({ readerStatus }),
  setError: (error) =>
    set(error ? { error, readerStatus: 'error' } : { error }),
  reset: () => set({ ...initialState }),
}));
