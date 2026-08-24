import { fetchTiming } from '@/services/api/files';
import type { Timing } from '@/schemas/timing';

/**
 * Loads timing files for the current chunk (with a small cache so
 * preloading the next chunk is cheap). The full book is never loaded at
 * once — only individual chunk timings.
 */
class TimingService {
  private cache = new Map<string, Timing>();
  private inflight = new Map<string, Promise<Timing>>();

  async load(timingPathOrUrl: string): Promise<Timing> {
    const cached = this.cache.get(timingPathOrUrl);
    if (cached) {
      return cached;
    }

    const pending = this.inflight.get(timingPathOrUrl);
    if (pending) {
      return pending;
    }

    const promise = fetchTiming(timingPathOrUrl)
      .then((timing) => {
        this.cache.set(timingPathOrUrl, timing);
        return timing;
      })
      .finally(() => {
        this.inflight.delete(timingPathOrUrl);
      });

    this.inflight.set(timingPathOrUrl, promise);
    return promise;
  }

  /** Fire-and-forget preload of the next chunk's timing. */
  preload(timingPathOrUrl: string): void {
    void this.load(timingPathOrUrl).catch(() => {
      // Preloading is best-effort; the real load reports failures.
    });
  }

  getCached(timingPathOrUrl: string): Timing | undefined {
    return this.cache.get(timingPathOrUrl);
  }

  clearCache(): void {
    this.cache.clear();
    this.inflight.clear();
  }
}

export const timingService = new TimingService();
