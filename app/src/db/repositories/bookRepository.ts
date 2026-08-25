import { eq } from 'drizzle-orm';
import type { BookStatus } from '@/schemas/book';
import { getDb } from '@/db/client';
import { books, jobs, type BookRow, type JobRow } from '@/db/schema';

export interface LocalBookInput {
  id: string;
  title: string;
  author?: string | null;
  language?: string;
  sourceType?: string;
  status: BookStatus;
}

function toRow(input: LocalBookInput): typeof books.$inferInsert {
  const now = new Date().toISOString();
  return {
    id: input.id,
    title: input.title,
    author: input.author ?? null,
    language: input.language ?? 'en',
    sourceType: input.sourceType ?? 'text',
    status: input.status,
    createdAt: now,
    updatedAt: now,
  };
}

/** Returns the single local book, or null when none exists. */
export function getLocalBook(): BookRow | null {
  return getDb().select().from(books).get() ?? null;
}

export function getLocalBookById(id: string): BookRow | null {
  return getDb().select().from(books).where(eq(books.id, id)).get() ?? null;
}

/**
 * Saves (inserts or updates) the local book. Enforces the single-book
 * rule at the database boundary: a different book cannot be stored while
 * another one exists.
 */
export function saveBook(input: LocalBookInput): BookRow {
  const database = getDb();
  const existing = database.select().from(books).get();

  if (existing && existing.id !== input.id) {
    throw new Error(
      `Another book already exists (${existing.id}). Delete it before adding a new one.`,
    );
  }

  if (existing) {
    database
      .update(books)
      .set({
        title: input.title,
        author: input.author ?? existing.author,
        language: input.language ?? existing.language,
        sourceType: input.sourceType ?? existing.sourceType,
        status: input.status,
        updatedAt: new Date().toISOString(),
      })
      .where(eq(books.id, existing.id))
      .run();
    return database.select().from(books).where(eq(books.id, existing.id)).get()!;
  }

  database.insert(books).values(toRow(input)).run();
  return database.select().from(books).where(eq(books.id, input.id)).get()!;
}

export function updateBookStatus(
  bookId: string,
  status: BookStatus,
): void {
  getDb()
    .update(books)
    .set({ status, updatedAt: new Date().toISOString() })
    .where(eq(books.id, bookId))
    .run();
}

/** Deletes the local book row; chapters/chunks/jobs/progress cascade. */
export function deleteLocalBook(bookId: string): void {
 getDb().delete(books).where(eq(books.id, bookId)).run();
}

// ---------------------------------------------------------------------------
// Jobs (processing recovery)
// ---------------------------------------------------------------------------

export interface LocalJobInput {
  id: string;
  bookId: string;
  status: string;
  progress?: number;
  error?: string | null;
}

export function saveJob(input: LocalJobInput): JobRow {
  const database = getDb();
  const now = new Date().toISOString();
  const values = {
    id: input.id,
    bookId: input.bookId,
    status: input.status,
    progress: input.progress ?? 0,
    error: input.error ?? null,
    createdAt: now,
    updatedAt: now,
  };

  database
    .insert(jobs)
    .values(values)
    .onConflictDoUpdate({
      target: jobs.id,
      set: {
        status: values.status,
        progress: values.progress,
        error: values.error,
        updatedAt: now,
      },
    })
    .run();

  return database.select().from(jobs).where(eq(jobs.id, input.id)).get()!;
}

export function getJob(jobId: string): JobRow | null {
  return getDb().select().from(jobs).where(eq(jobs.id, jobId)).get() ?? null;
}

/** Returns a pending job (queued/processing), used for startup recovery. */
export function getPendingJob(): JobRow | null {
  const rows = getDb().select().from(jobs).all();
  return (
    rows.find((row) => row.status === 'queued' || row.status === 'processing') ??
    null
  );
}
