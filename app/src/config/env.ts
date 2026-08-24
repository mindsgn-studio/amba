const DEFAULT_API_URL = 'http://localhost:8000';

function stripTrailingSlash(url: string): string {
  return url.replace(/\/+$/, '');
}

function resolveApiBaseUrl(): string {
  const raw = process.env.EXPO_PUBLIC_API_URL;
  if (!raw || raw.trim().length === 0) {
    return DEFAULT_API_URL;
  }
  return stripTrailingSlash(raw.trim());
}

export const API_BASE_URL = resolveApiBaseUrl();

export const env = {
  apiUrl: API_BASE_URL,
  requestTimeoutMs: 15000,
  jobPollIntervalMs: 3000,
  progressSaveIntervalMs: 3000,
} as const;

export const SUPPORTED_UPLOAD_EXTENSIONS = ['pdf', 'epub', 'txt'] as const;

export type UploadExtension = (typeof SUPPORTED_UPLOAD_EXTENSIONS)[number];

export function getFileExtension(fileName: string): string {
  const parts = fileName.toLowerCase().split('.');
  const ext = parts[parts.length - 1];
  return parts.length > 1 ? ext : '';
}

export function isSupportedUploadFile(fileName: string): boolean {
  return (SUPPORTED_UPLOAD_EXTENSIONS as readonly string[]).includes(
    getFileExtension(fileName),
  );
}
