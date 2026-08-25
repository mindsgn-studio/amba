import {
  index,
  integer,
  real,
  sqliteTable,
  text,
  uniqueIndex,
} from 'drizzle-orm/sqlite-core';

export const books = sqliteTable('books', {
  id: text('id').primaryKey(),
  title: text('title').notNull(),
  author: text('author'),
  language: text('language').notNull().default('en'),
  sourceType: text('source_type').notNull().default('text'),
  status: text('status').notNull().default('queued'),
  createdAt: text('created_at').notNull(),
  updatedAt: text('updated_at').notNull(),
});

export const chapters = sqliteTable(
  'chapters',
  {
    id: text('id').primaryKey(),
    bookId: text('book_id')
      .notNull()
      .references(() => books.id, { onDelete: 'cascade' }),
    chapterIndex: integer('chapter_index').notNull(),
    title: text('title').notNull(),
  },
  (table) => [
    index('chapters_book_id_idx').on(table.bookId),
    uniqueIndex('chapters_book_index_unique').on(
      table.bookId,
      table.chapterIndex,
    ),
  ],
);

/**
 * Audio chunks. Files themselves live on the backend; only URLs and
 * metadata are persisted locally.
 */
export const chunks = sqliteTable(
  'chunks',
  {
    id: text('id').primaryKey(),
    chapterId: text('chapter_id')
      .notNull()
      .references(() => chapters.id, { onDelete: 'cascade' }),
    chunkIndex: integer('chunk_index').notNull(),
    duration: real('duration'),
    audioUrl: text('audio_url'),
    timingUrl: text('timing_url'),
  },
  (table) => [
    index('chunks_chapter_id_idx').on(table.chapterId),
    uniqueIndex('chunks_chapter_index_unique').on(
      table.chapterId,
      table.chunkIndex,
    ),
  ],
);

/** Persisted reading position. One row per book. */
export const readingProgress = sqliteTable(
  'reading_progress',
  {
    id: integer('id').primaryKey({ autoIncrement: true }),
    bookId: text('book_id')
      .notNull()
      .references(() => books.id, { onDelete: 'cascade' }),
    chapterId: text('chapter_id').notNull(),
    chunkId: text('chunk_id').notNull(),
    positionSeconds: real('position_seconds').notNull().default(0),
    wordIndex: integer('word_index').notNull().default(0),
    /** 1 when the reader reached the end of the final chunk. */
    completed: integer('completed').notNull().default(0),
    updatedAt: text('updated_at').notNull(),
  },
  (table) => [uniqueIndex('reading_progress_book_unique').on(table.bookId)],
);

/**
 * Job information required to recover processing across app restarts.
 * Zustand state is lost when the process dies; this table survives.
 */
export const jobs = sqliteTable('jobs', {
  id: text('id').primaryKey(),
  bookId: text('book_id')
    .notNull()
    .references(() => books.id, { onDelete: 'cascade' }),
  status: text('status').notNull().default('queued'),
  progress: real('progress').notNull().default(0),
  error: text('error'),
  createdAt: text('created_at').notNull(),
  updatedAt: text('updated_at').notNull(),
});

export type BookRow = typeof books.$inferSelect;
export type ChapterRow = typeof chapters.$inferSelect;
export type ChunkRow = typeof chunks.$inferSelect;
export type ReadingProgressRow = typeof readingProgress.$inferSelect;
export type JobRow = typeof jobs.$inferSelect;
