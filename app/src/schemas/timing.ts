import { z } from 'zod';

/** A single word within a chunk, timed against the synthesized audio. */
export const WordTimingSchema = z.object({
  index: z.number().int().nonnegative(),
  text: z.string(),
  start: z.number().min(0),
  end: z.number().min(0),
});

export type WordTiming = z.infer<typeof WordTimingSchema>;

/**
 * Timing file served from GET /api/files/... for a single chunk.
 * The `text` field must be rendered exactly as supplied by the backend.
 */
export const TimingSchema = z.object({
  chunk_id: z.string().min(1),
  text: z.string(),
  words: z.array(WordTimingSchema).min(1),
});

export type Timing = z.infer<typeof TimingSchema>;
