import { env } from '@/config/env';
import { resolveApiUrl } from '@/utils/urls';

export class ApiError extends Error {
  readonly status: number;
  readonly payload?: unknown;

  constructor(status: number, message: string, payload?: unknown) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
    this.payload = payload;
  }
}

export type HttpMethod = 'GET' | 'POST' | 'DELETE';

interface RequestOptions {
  method?: HttpMethod;
  body?: BodyInit;
  headers?: Record<string, string>;
}

function extractErrorMessage(payload: unknown, fallback: string): string {
  if (
    payload &&
    typeof payload === 'object' &&
    'detail' in payload &&
    typeof (payload as { detail?: unknown }).detail === 'string'
  ) {
    return (payload as { detail: string }).detail;
  }
  return fallback;
}

/**
 * Performs a JSON request against the backend and validates the response
 * with the provided Zod schema. Non-2xx responses throw an ApiError.
 */
export async function apiRequest<T>(
  path: string,
  schema: { parse: (data: unknown) => T },
  options: RequestOptions = {},
): Promise<T> {
  const url = resolveApiUrl(path);
  let response: Response;

  try {
    response = await fetch(url, {
      method: options.method ?? 'GET',
      headers: options.headers,
      body: options.body,
      // signal: AbortSignal.timeout(env.requestTimeoutMs),
    });
  } catch (error) {
    if (error instanceof ApiError) {
      throw error;
    }
    throw new ApiError(0, `Unable to connect to server (${url}): ${error instanceof Error ? error.message : String(error)}`);
  }

  const text = await response.text();
  let payload: unknown = undefined;
  if (text.length > 0) {
    try {
      payload = JSON.parse(text);
    } catch {
      payload = undefined;
    }
  }

  if (!response.ok) {
    throw new ApiError(
      response.status,
      extractErrorMessage(
        payload,
        `Request failed with status ${response.status}`,
      ),
      payload,
    );
  }

  return schema.parse(payload);
}
