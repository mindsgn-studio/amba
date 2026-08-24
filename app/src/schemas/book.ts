import { z } from 'zod';
import { JobStatusSchema } from '@/schemas/job';

export const BOOK_STATUSES = ['queued', 'processing', 'completed', 'failed'] as const;

export const BookStatusSchema = z.enum(BOOK_STATUSES);

export type BookStatus = z.infer<typeof BookStatusSchema>;

/**
 * Book as returned by GET /api/books list items.
 */
export const BookSchema = z.object({
  id: z.string().min(1),
  title: z.string(),
  author: z.string().nullable().optional(),
  language: z.string(),
  source_type: z.string(),
  status: BookStatusSchema,
  created_at: z.string(),
  updated_at: z.string(),
});

export type Book = z.infer<typeof BookSchema>;

/** GET /api/books */
export const BookListResponseSchema = z.object({
  books: z.array(BookSchema),
});

export type BookListResponse = z.infer<typeof BookListResponseSchema>;

/** A chunk of synthesized audio with playable/parseable file URLs. */
export const ChunkSchema = z.object({
  id: z.string().min(1),
  chunk_index: z.number().int().nonnegative(),
  duration: z.number().nullable().optional(),
  audio_url: z.string().min(1),
  timing_url: z.string().min(1),
});

export type Chunk = z.infer<typeof ChunkSchema>;

export const ChapterSchema = z.object({
  id: z.string().min(1),
  chapter_index: z.number().int().nonnegative(),
  title: z.string(),
});

export type Chapter = z.infer<typeof ChapterSchema>;

/** Chapter with its chunks, as nested in GET /api/books/{book_id}. */
export const ChapterWithChunksSchema = ChapterSchema.extend({
  chunks: z.array(ChunkSchema),
});

export type ChapterWithChunks = z.infer<typeof ChapterWithChunksSchema>;

/**
 * Full book detail. The backend example response includes only
 * id/title/status/chapters, so descriptive fields are optional here.
 */
export const BookDetailSchema = z.object({
  id: z.string().min(1),
  title: z.string(),
  author: z.string().nullable().optional(),
  language: z.string().optional(),
  source_type: z.string().optional(),
  status: BookStatusSchema,
  created_at: z.string().optional(),
  updated_at: z.string().optional(),
  chapters: z.array(ChapterWithChunksSchema),
});

export type BookDetail = z.infer<typeof BookDetailSchema>;

/** POST /api/books and POST /api/books/upload */
export const CreateBookResponseSchema = z.object({
  book_id: z.string().min(1),
  job_id: z.string().min(1),
  status: JobStatusSchema,
});

export type CreateBookResponse = z.infer<typeof CreateBookResponseSchema>;

/** DELETE /api/books/{book_id} */
export const DeleteBookResponseSchema = z.object({
  deleted: z.boolean(),
  id: z.string().min(1),
});

export type DeleteBookResponse = z.infer<typeof DeleteBookResponseSchema>;
