import { createAudioPlayer, setAudioModeAsync, type AudioPlayer } from 'expo-audio';
import type {
  AudioController,
  PlaybackListener,
  PlaybackStatus,
} from '@/services/audio/types';

interface StatusSubscription {
  remove(): void;
}

let audioModeConfigured = false;

async function ensureAudioMode(): Promise<void> {
  if (audioModeConfigured) {
    return;
  }
  audioModeConfigured = true;
  try {
    await setAudioModeAsync({
      playsInSilentMode: true,
      shouldPlayInBackground: false,
      interruptionMode: 'doNotMix',
    });
  } catch {
    // Audio mode is a best-effort optimization; playback can proceed.
  }
}

/**
 * expo-audio backed controller. One native player is alive at a time;
 * load() replaces it so chunk transitions reuse the same object graph.
 */
export class ExpoAudioController implements AudioController {
  private player: AudioPlayer | null = null;
  private listeners = new Set<PlaybackListener>();
  private subscription: StatusSubscription | null = null;

  async load(url: string): Promise<void> {
    await ensureAudioMode();
    this.teardownPlayer();

    // 50ms position updates keep word highlighting responsive.
    const player = createAudioPlayer({ uri: url }, { updateInterval: 50 });
    this.player = player;
    this.subscription = player.addListener('playbackStatusUpdate', (status) => {
      const update: PlaybackStatus = {
        currentTime: status.currentTime,
        duration: status.duration ?? 0,
        didJustFinish: status.didJustFinish ?? false,
      };
      this.listeners.forEach((listener) => listener(update));
    });
  }

  async play(): Promise<void> {
    this.player?.play();
  }

  async pause(): Promise<void> {
    this.player?.pause();
  }

  async stop(): Promise<void> {
    if (!this.player) {
      return;
    }
    this.player.pause();
    await this.player.seekTo(0);
  }

  async seek(seconds: number): Promise<void> {
    if (!this.player) {
      return;
    }
    await this.player.seekTo(Math.max(0, seconds));
  }

  async setRate(rate: number): Promise<void> {
    if (!this.player) {
      return;
    }
    this.player.shouldCorrectPitch = true;
    this.player.setPlaybackRate(rate);
  }

  getCurrentTime(): number {
    return this.player?.currentTime ?? 0;
  }

  getDuration(): number {
    return this.player?.duration ?? 0;
  }

  onStatusChange(listener: PlaybackListener): () => void {
    this.listeners.add(listener);
    return () => {
      this.listeners.delete(listener);
    };
  }

  release(): void {
    this.teardownPlayer();
  }

  private teardownPlayer(): void {
    this.subscription?.remove();
    this.subscription = null;
    if (this.player) {
      try {
        this.player.remove();
      } catch {
        // Player may already be released.
      }
      this.player = null;
    }
  }
}

export function createAudioController(): AudioController {
  return new ExpoAudioController();
}
