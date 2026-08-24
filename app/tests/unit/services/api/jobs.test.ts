import { getJob } from '@/services/api/jobs';
import { fetchTiming, resolveFileUrl } from '@/services/api/files';
import jobProcessing from '../../../fixtures/job-processing.json';
import timingFixture from '../../../fixtures/timing.json';

function mockJson(status: number, body: unknown): void {
  (global.fetch as jest.Mock).mockResolvedValueOnce({
    ok: status >= 200 && status < 300,
    status,
    text: async () => JSON.stringify(body),
  });
}

describe('jobs API', () => {
  it('fetches and validates job status', async () => {
    mockJson(200, jobProcessing);
    const job = await getJob('job123');
    expect(job.status).toBe('processing');
    expect(job.progress).toBe(64);
    expect(global.fetch).toHaveBeenCalledWith(
      'http://localhost:8000/api/jobs/job123',
      expect.anything(),
    );
  });

  it.each([404, 500])('throws for %p responses', async (status) => {
    mockJson(status, { detail: 'error' });
    await expect(getJob('job123')).rejects.toMatchObject({ status });
  });
});

describe('files API', () => {
  it('resolves file paths to absolute URLs', () => {
    expect(resolveFileUrl('/api/files/books/abc/chunk_001.mp3')).toBe(
      'http://localhost:8000/api/files/books/abc/chunk_001.mp3',
    );
  });

  it('fetches and validates timing files', async () => {
    mockJson(200, timingFixture);
    const timing = await fetchTiming(
      '/api/files/books/abc123/chapters/001/chunk_001.json',
    );
    expect(timing.chunk_id).toBe('chunk_001');
    expect(timing.text).toBe(timingFixture.text);
  });

  it('fails safely on corrupted timing payloads', async () => {
    mockJson(200, { words: 'garbage' });
    await expect(fetchTiming('/api/files/x.json')).rejects.toThrow();
  });

  it('throws for missing files', async () => {
    mockJson(404, { detail: 'Not Found' });
    await expect(fetchTiming('/api/files/x.json')).rejects.toMatchObject({
      status: 404,
    });
  });
});
