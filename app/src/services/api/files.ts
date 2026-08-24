import { TimingSchema, type Timing } from '@/schemas/timing';
import { apiRequest } from '@/services/api/client';
import { resolveApiUrl } from '@/utils/urls';

/**
 * Resolves an API file path (e.g. "/api/files/books/abc/chunk_001.mp3")
 * into an absolute URL usable by the audio player.
 */
export function resolveFileUrl(path: string): string {
  return resolveApiUrl(path);
}

/** Fetches and validates a timing JSON file for one chunk. */
export function fetchTiming(timingPathOrUrl: string): Promise<Timing> {
  return apiRequest(timingPathOrUrl, TimingSchema);
}
