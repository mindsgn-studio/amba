import { asc, eq, inArray } from 'drizzle-orm';
import { getDb } from '@/db/client';
import {
  chapters,
  chunks,
  type ChunkRow,
} from '@/db/schema';

export interface LocalChunkInput {
  id: string;
  chapterId: string;
  chunkIndex: number;
  duration?: number | null;
  audioUrl?: string | null;
  timingUrl?: string | null;
}

/**
 * Replaces the chunks of the given chapters with the supplied list.
 * Chunks are always saved together with their chapter ids.
 */
export function saveChunks(
  inputs: readonly (LocalChunkInput & { chapterId: string })[],
): void {
  const database = getDb();

  const chapterIds = [...new Set(inputs.map((input) => input.chapterId))];
  if (chapterIds.length > 0) {
    database.delete(chunks).where(inArray(chunks.chapterId, chapterIds)).run();
  }

  if (inputs.length === 0) {
    return;
  }

  database
    .insert(chunks)
    .values(
      inputs.map((input) => ({
        id: input.id,
        chapterId: input.chapterId,
        chunkIndex: input.chunkIndex,
        duration: input.duration ?? null,
        audioUrl: input.audioUrl ?? null,
        timingUrl: input.timingUrl ?? null,
      })),
    )
    .run();
}

/**
 * Returns chunks for the given chapters, ordered by chapter position and
 * then by chunk index — i.e. playback order.
 */
export function getChunks(chapterIds: readonly string[]): ChunkRow[] {
  if (chapterIds.length === 0) {
    return [];
  }
  return getDb()
    .select({
      id: chunks.id,
      chapterId: chunks.chapterId,
      chunkIndex: chunks.chunkIndex,
      duration: chunks.duration,
      audioUrl: chunks.audioUrl,
      timingUrl: chunks.timingUrl,
    })
    .from(chunks)
    .innerJoin(chapters, eq(chunks.chapterId, chapters.id))
    .where(inArray(chunks.chapterId, [...chapterIds]))
    .orderBy(asc(chapters.chapterIndex), asc(chunks.chunkIndex))
    .all();
}

export function getChunkById(chunkId: string): ChunkRow | null {
  return (
    getDb().select().from(chunks).where(eq(chunks.id, chunkId)).get() ??
    null
  );
}
