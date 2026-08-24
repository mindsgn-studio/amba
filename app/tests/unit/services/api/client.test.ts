import { ApiError, apiRequest } from '@/services/api/client';
import { z } from 'zod';

const Schema = z.object({ ok: z.boolean() });

function mockResponse(
  status: number,
  body: unknown,
  options: { raw?: string } = {},
): void {
  const text =
    options.raw ?? (body === undefined ? '' : JSON.stringify(body));
  (global.fetch as jest.Mock).mockResolvedValueOnce({
    ok: status >= 200 && status < 300,
    status,
    text: async () => text,
  });
}

describe('apiRequest', () => {
  it('parses and validates 200 responses', async () => {
    mockResponse(200, { ok: true });
    await expect(apiRequest('/api/test', Schema)).resolves.toEqual({ ok: true });
    expect(global.fetch).toHaveBeenCalledWith(
      'http://localhost:8000/api/test',
      expect.objectContaining({ method: 'GET' }),
    );
  });

  it('accepts 202 responses', async () => {
    mockResponse(202, { ok: true });
    await expect(apiRequest('/api/test', Schema)).resolves.toEqual({ ok: true });
  });

  it.each([400, 404, 415, 422, 500])(
    'throws ApiError for %p responses',
    async (status) => {
      mockResponse(status, { detail: 'Something specific' });
      const promise = apiRequest('/api/test', Schema);
      await expect(promise).rejects.toBeInstanceOf(ApiError);
      await promise.catch((error: ApiError) => {
        expect(error.status).toBe(status);
        expect(error.message).toBe('Something specific');
      });
    },
  );

  it('falls back to a generic message without a detail field', async () => {
    mockResponse(500, { unexpected: true });
    const promise = apiRequest('/api/test', Schema);
    await expect(promise).rejects.toThrow(/status 500/);
  });

  it('treats empty bodies on success as invalid JSON payloads', async () => {
    mockResponse(200, undefined, { raw: '' });
    // Empty payload fails schema validation rather than silently passing.
    await expect(apiRequest('/api/test', Schema)).rejects.toThrow();
  });

  it('wraps network failures as connection errors', async () => {
    (global.fetch as jest.Mock).mockRejectedValueOnce(new Error('offline'));
    const promise = apiRequest('/api/test', Schema);
    await expect(promise).rejects.toThrow(/Unable to connect to server/);
    await promise.catch((error: ApiError) => {
      expect(error.status).toBe(0);
    });
  });

  it('never blindly trusts API JSON — invalid shapes throw', async () => {
    mockResponse(200, { ok: 'not-a-boolean' });
    await expect(apiRequest('/api/test', Schema)).rejects.toThrow();
  });

  it('sends POST bodies and headers', async () => {
    mockResponse(200, { ok: true });
    await apiRequest('/api/test', Schema, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ a: 1 }),
    });
    expect(global.fetch).toHaveBeenCalledWith(
      'http://localhost:8000/api/test',
      expect.objectContaining({
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ a: 1 }),
      }),
    );
  });
});
