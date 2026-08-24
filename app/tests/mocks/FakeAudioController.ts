import type {
  AudioController,
  PlaybackListener,
  PlaybackStatus,
} from '@/services/audio/types';

/**
 * Deterministic in-memory AudioController used by tests. Playback time
 * only advances when the test explicitly ticks the controller.
 */
export class FakeAudioController implements AudioController {
  loadedUrls: string[] = [];
  played = 0;
  pausedCount = 0;
  stoppedCount = 0;
  released = false;
  rate = 1;

  private currentTimeValue = 0;
  private durationValue = 0;
  private listeners = new Set<PlaybackListener>();

  load(url: string): Promise<void> {
    this.loadedUrls.push(url);
    this.currentTimeValue = 0;
    this.durationValue = 0;
    return Promise.resolve();
  }

  play(): Promise<void> {
    this.played += 1;
    return Promise.resolve();
  }

  pause(): Promise<void> {
    this.pausedCount += 1;
    return Promise.resolve();
  }

  stop(): Promise<void> {
    this.stoppedCount += 1;
    this.currentTimeValue = 0;
    return Promise.resolve();
  }

  seek(seconds: number): Promise<void> {
    this.currentTimeValue = seconds;
    return Promise.resolve();
  }

  setRate(rate: number): Promise<void> {
    this.rate = rate;
    return Promise.resolve();
  }

  getCurrentTime(): number {
    return this.currentTimeValue;
  }

  getDuration(): number {
    return this.durationValue;
  }

  onStatusChange(listener: PlaybackListener): () => void {
    this.listeners.add(listener);
    return () => {
      this.listeners.delete(listener);
    };
  }

  release(): void {
    this.released = true;
  }

  // --- test controls -----------------------------------------------------

  setTime(seconds: number, duration?: number): void {
    this.currentTimeValue = seconds;
    if (duration != null) {
      this.durationValue = duration;
    }
  }

  emit(status?: Partial<PlaybackStatus>): void {
    const update: PlaybackStatus = {
      currentTime: this.currentTimeValue,
      duration: this.durationValue,
      didJustFinish: false,
      ...status,
    };
    this.listeners.forEach((listener) => listener(update));
  }

  finish(): void {
    this.emit({ didJustFinish: true });
  }
}
