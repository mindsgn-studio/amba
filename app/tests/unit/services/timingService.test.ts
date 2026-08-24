import { timingService } from '@/services/reader/timingService';
import type { Timing } from '@/schemas/timing';
import timingFixture from '../../fixtures/timing.json';

function mockTimingResponse(body: unknown, status = 200): void {
  (global.fetch as jest.Mock).mockResolvedValueOnce({
    ok: status >= 200 && status < 300,
    status,
    text: async () => JSON.stringify(body),
  });
}

describe('timingService', () => {
  beforeEach(() => {
    (global.fetch as jest.Mock).mockClear();
    timingService.clearCache();
  });

  it('loads, validates and caches a timing file', async () => {
    mockTimingResponse(timingFixture);
    const timing = await timingService.load('/api/files/a.json');
    expect((timing as Timing).chunk_id).toBe('chunk_001');
    expect(global.fetch).toHaveBeenCalledTimes(1);

    // Second load is served from cache — no extra network call.
    const again = await timingService.load('/api/files/a.json');
    expect(again.chunk_id).toBe('chunk_001');
    expect(global.fetch).toHaveBeenCalledTimes(1);
  });

  it('deduplicates concurrent loads of the same URL', async () => {
    mockTimingResponse(timingFixture);
    const [a, b] = await Promise.all([
      timingService.load('/api/files/b.json'),
      timingService.load('/api/files/b.json'),
    ]);
    expect(a.chunk_id).toBe(b.chunk_id);
    expect(global.fetch).toHaveBeenCalledTimes(1);
  });

  it('rejects invalid payloads instead of trusting the API', async () => {
    mockTimingResponse({ words: 'garbage' });
    await expect(timingService.load('/api/files/c.json')).rejects.toThrow();
  });

  it('preloads without surfacing failures', async () => {
    (global.fetch as jest.Mock).mockRejectedValueOnce(new Error('offline'));
    expect(() => timingService.preload('/api/files/d.json')).not.toThrow();
    await new Promise((resolve) => setTimeout(resolve, 0));
  });

  it('clears the cache', async () => {
    mockTimingResponse(timingFixture);
    await timingService.load('/api/files/e.json');
    timingService.clearCache();
    expect(timingService.getCached('/api/files/e.json')).toBeUndefined();
    mockTimingResponse(timingFixture);
    await timingService.load('/api/files/e.json');
    expect(global.fetch).toHaveBeenCalledTimes(2);
  });
});
