import { asc, eq } from 'drizzle-orm';
import { getDb } from '@/db/client';
import {
  chapters,
  type ChapterRow,
} from '@/db/schema';

export interface LocalChapterInput {
  id: string;
  bookId: string;
  chapterIndex: number;
  title: string;
}

/** Replaces all chapters for a book with the supplied list. */
export function saveChapters(inputs: readonly LocalChapterInput[]): void {
  const database = getDb();

  if (inputs.length > 0) {
    database.delete(chapters).where(eq(chapters.bookId, inputs[0].bookId)).run();
  }

  if (inputs.length === 0) {
    return;
  }

  database
    .insert(chapters)
    .values(
      inputs.map((input) => ({
        id: input.id,
        bookId: input.bookId,
        chapterIndex: input.chapterIndex,
        title: input.title,
      })),
    )
    .run();
}

export function getChapters(bookId: string): ChapterRow[] {
  return getDb()
    .select()
    .from(chapters)
    .where(eq(chapters.bookId, bookId))
    .orderBy(asc(chapters.chapterIndex))
    .all();
}

export function getChapterById(chapterId: string): ChapterRow | null {
  return (
    getDb().select().from(chapters).where(eq(chapters.id, chapterId)).get() ??
    null
  );
}
