import type { BookDetail } from '@/schemas/book';
import { getBook } from '@/services/api/books';
import {
  updateBookStatus,
} from '@/db/repositories/bookRepository';
import { getChapters, saveChapters } from '@/db/repositories/chapterRepository';
import { getChunks, saveChunks } from '@/db/repositories/chunkRepository';
import {
  deleteReadingProgress,
  getReadingProgress,
  saveReadingProgress,
} from '@/db/repositories/progressRepository';
import type { ReadingProgressRow } from '@/db/schema';

/** One playable unit: a chunk within its chapter context. */
export interface ChunkRef {
  chapterId: string;
  chapterIndex: number;
  chapterTitle: string;
  chapterCount: number;
  chunkId: string;
  chunkIndex: number;
  audioUrl: string;
  timingUrl: string;
  duration: number | null;
}

/**
 * Loads the ordered playlist (chapter → chunks) for a book from SQLite.
 * Returns an empty list when the book detail has not been persisted yet.
 */
export function loadPlaylist(bookId: string): ChunkRef[] {
  const chapters = getChapters(bookId);
  if (chapters.length === 0) {
    return [];
  }

  const rows = getChunks(chapters.map((chapter) => chapter.id));
  const byChapter = new Map<string, typeof rows>();
  for (const row of rows) {
    const list = byChapter.get(row.chapterId) ?? [];
    list.push(row);
    byChapter.set(row.chapterId, list);
  }

  const playlist: ChunkRef[] = [];
  for (const chapter of [...chapters].sort(
    (a, b) => a.chapterIndex - b.chapterIndex,
  )) {
    const chapterChunks = byChapter.get(chapter.id) ?? [];
    for (const chunk of chapterChunks.sort(
      (a, b) => a.chunkIndex - b.chunkIndex,
    )) {
      if (!chunk.audioUrl || !chunk.timingUrl) {
        continue;
      }
      playlist.push({
        chapterId: chapter.id,
        chapterIndex: chapter.chapterIndex,
        chapterTitle: chapter.title,
        chapterCount: chapters.length,
        chunkId: chunk.id,
        chunkIndex: chunk.chunkIndex,
        audioUrl: chunk.audioUrl,
        timingUrl: chunk.timingUrl,
        duration: chunk.duration ?? null,
      });
    }
  }
  return playlist;
}

export interface ResumeLocation {
  index: number;
  positionSeconds: number;
  wordIndex: number;
  completed: boolean;
}

const START_LOCATION: ResumeLocation = {
  index: 0,
  positionSeconds: 0,
  wordIndex: 0,
  completed: false,
};

/**
 * Maps a persisted reading progress onto the playlist. Falls back to the
 * beginning when no progress exists or the saved chunk is gone.
 */
export function resolveResumeLocation(
  playlist: readonly ChunkRef[],
  progress: ReadingProgressRow | null,
): ResumeLocation {
  if (!progress || playlist.length === 0) {
    return START_LOCATION;
  }

  const index = playlist.findIndex((ref) => ref.chunkId === progress.chunkId);
  if (index === -1) {
    return START_LOCATION;
  }

  const maxPosition =
    playlist[index]?.duration != null
      ? Math.max(0, playlist[index].duration - 0.25)
      : progress.positionSeconds;

  return {
    index,
    positionSeconds:
      progress.completed > 0 ? 0 : Math.min(progress.positionSeconds, maxPosition),
    wordIndex: progress.completed > 0 ? 0 : progress.wordIndex,
    completed: progress.completed > 0 && index === playlist.length - 1,
  };
}

export function getSavedProgress(bookId: string): ReadingProgressRow | null {
  return getReadingProgress(bookId);
}

export function persistProgress(input: {
  bookId: string;
  ref: ChunkRef;
  positionSeconds: number;
  wordIndex: number;
}): void {
  saveReadingProgress({
    bookId: input.bookId,
    chapterId: input.ref.chapterId,
    chunkId: input.ref.chunkId,
    positionSeconds: input.positionSeconds,
    wordIndex: input.wordIndex,
    completed: false,
  });
}

export function markBookCompletedInDb(bookId: string, ref: ChunkRef): void {
  saveReadingProgress({
    bookId,
    chapterId: ref.chapterId,
    chunkId: ref.chunkId,
    positionSeconds: 0,
    wordIndex: 0,
    completed: true,
  });
}

export function clearProgress(bookId: string): void {
  deleteReadingProgress(bookId);
}

/**
 * Fetches full book detail from the backend and persists chapters +
 * chunks locally so the reader can work offline of the API afterwards.
 */
export async function syncBookFromBackend(bookId: string): Promise<BookDetail> {
  const detail = await getBook(bookId);
  saveChapters(
    detail.chapters.map((chapter) => ({
      id: chapter.id,
      bookId: detail.id,
      chapterIndex: chapter.chapter_index,
      title: chapter.title,
    })),
  );

  saveChunks(
    detail.chapters.flatMap((chapter) =>
      chapter.chunks.map((chunk) => ({
        id: chunk.id,
        chapterId: chapter.id,
        chunkIndex: chunk.chunk_index,
        duration: chunk.duration ?? null,
        audioUrl: chunk.audio_url,
        timingUrl: chunk.timing_url,
      })),
    ),
  );

  updateBookStatus(detail.id, detail.status);
  return detail;
}
