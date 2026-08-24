import { useCallback, useEffect, useMemo, useRef } from 'react';
import { createAudioController } from '@/services/audio/audioService';
import type {
  AudioController,
  PlaybackListener,
} from '@/services/audio/types';

export interface PlaybackCommands {
  load(url: string): Promise<void>;
  play(): Promise<void>;
  pause(): Promise<void>;
  stop(): Promise<void>;
  seek(seconds: number): Promise<void>;
  setRate(rate: number): Promise<void>;
  getCurrentTime(): number;
  getDuration(): number;
  /** Stops playback and releases native resources. */
  release(): void;
}

/**
 * Owns the AudioController lifecycle for one screen and forwards status
 * updates to the provided listener. The controller is created lazily and
 * released when the hook unmounts.
 */
export function usePlayback(onStatus: PlaybackListener): PlaybackCommands {
  const controllerRef = useRef<AudioController | null>(null);
  const listenerRef = useRef<PlaybackListener>(onStatus);
  listenerRef.current = onStatus;

  const getController = useCallback((): AudioController => {
    if (!controllerRef.current) {
      controllerRef.current = createAudioController();
    }
    return controllerRef.current;
  }, []);

  useEffect(() => {
    const controller = getController();
    const unsubscribe = controller.onStatusChange((status) =>
      listenerRef.current(status),
    );
    return () => {
      unsubscribe();
    };
  }, [getController]);

  useEffect(
    () => () => {
      controllerRef.current?.release();
      controllerRef.current = null;
    },
    [],
  );

  return useMemo<PlaybackCommands>(
    () => ({
      load: (url) => getController().load(url),
      play: () => getController().play(),
      pause: () => getController().pause(),
      stop: () => getController().stop(),
      seek: (seconds) => getController().seek(seconds),
      setRate: (rate) => getController().setRate(rate),
      getCurrentTime: () => getController().getCurrentTime(),
      getDuration: () => getController().getDuration(),
      release: () => {
        controllerRef.current?.release();
        controllerRef.current = null;
      },
    }),
    [getController],
  );
}
