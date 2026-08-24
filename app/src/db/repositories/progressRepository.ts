import { eq } from 'drizzle-orm';
import { getDb } from '@/db/client';
import {
  readingProgress,
  type ReadingProgressRow,
} from '@/db/schema';

export interface ReadingProgressInput {
  bookId: string;
  chapterId: string;
  chunkId: string;
  positionSeconds: number;
  wordIndex: number;
  completed?: boolean;
}

function toValues(input: ReadingProgressInput) {
  return {
    bookId: input.bookId,
    chapterId: input.chapterId,
    chunkId: input.chunkId,
    positionSeconds: Math.max(0, input.positionSeconds),
    wordIndex: Math.max(0, Math.floor(input.wordIndex)),
    completed: input.completed ? 1 : 0,
    updatedAt: new Date().toISOString(),
  };
}

export function getReadingProgress(bookId: string): ReadingProgressRow | null {
  return (
    getDb()
      .select()
      .from(readingProgress)
      .where(eq(readingProgress.bookId, bookId))
      .get() ?? null
  );
}

/** Inserts or updates the reading progress for a book (one row per book). */
export function saveReadingProgress(input: ReadingProgressInput): void {
  getDb()
    .insert(readingProgress)
    .values(toValues(input))
    .onConflictDoUpdate({
      target: readingProgress.bookId,
      set: toValues(input),
    })
    .run();
}

export function deleteReadingProgress(bookId: string): void {
  getDb().delete(readingProgress).where(eq(readingProgress.bookId, bookId)).run();
}
