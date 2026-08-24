/**
 * Abstraction over the native audio engine (expo-audio).
 *
 * The Reader only talks to AudioController, which keeps expo-audio out of
 * components/hooks and allows deterministic fakes in Jest tests.
 */

export interface PlaybackStatus {
  currentTime: number;
  duration: number;
  didJustFinish: boolean;
}

export type PlaybackListener = (status: PlaybackStatus) => void;

export interface AudioController {
  /** Loads a URL, replacing any previously loaded source. */
  load(url: string): Promise<void>;
  play(): Promise<void>;
  pause(): Promise<void>;
  /** Stops playback and rewinds to the beginning. */
  stop(): Promise<void>;
  seek(seconds: number): Promise<void>;
  setRate(rate: number): Promise<void>;
  getCurrentTime(): number;
  getDuration(): number;
  onStatusChange(listener: PlaybackListener): () => void;
  /** Stops playback and releases all native resources. */
  release(): void;
}
