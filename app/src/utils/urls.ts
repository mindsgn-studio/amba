import { API_BASE_URL } from '@/config/env';

/**
 * Resolves a backend path into an absolute URL.
 *
 * - Absolute URLs are returned unchanged.
 * - Paths such as "/api/files/books/abc/chunk_001.mp3" are prefixed with
 *   the configured API base URL.
 * - Paths without a leading slash are normalized.
 */
export function resolveApiUrl(pathOrUrl: string): string {
  if (/^https?:\/\//i.test(pathOrUrl)) {
    return pathOrUrl;
  }
  const path = pathOrUrl.startsWith('/') ? pathOrUrl : `/${pathOrUrl}`;
  return `${API_BASE_URL}${path}`;
}

/** Builds an API endpoint URL from a path such as "/api/books". */
export function apiUrl(path: string): string {
  const normalized = path.startsWith('/') ? path : `/${path}`;
  return `${API_BASE_URL}${normalized}`;
}
