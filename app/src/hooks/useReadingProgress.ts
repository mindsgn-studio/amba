import { useCallback, useRef } from 'react';
import { env } from '@/config/env';
import type { ChunkRef } from '@/services/reader/readerService';
import { persistProgress } from '@/services/reader/readerService';

type SaveInput = {
  ref: ChunkRef;
  positionSeconds: number;
  wordIndex: number;
};

/**
 * Persists reading position to SQLite without writing on every audio
 * tick (~50ms). Throttled writes happen at most once per interval;
 * important moments (pause, exit, chunk transition) save immediately.
 */
export function useReadingProgress(bookId: string | null): {
  saveThrottled: (input: SaveInput) => void;
  saveNow: (input: SaveInput) => void;
} {
  const lastSavedAtRef = useRef(0);

  const saveNow = useCallback(
    ({ ref, positionSeconds, wordIndex }: SaveInput): void => {
      if (!bookId) {
        return;
      }
      lastSavedAtRef.current = Date.now();
      persistProgress({ bookId, ref, positionSeconds, wordIndex });
    },
    [bookId],
  );

  const saveThrottled = useCallback(
    (input: SaveInput): void => {
      if (Date.now() - lastSavedAtRef.current < env.progressSaveIntervalMs) {
        return;
      }
      saveNow(input);
    },
    [saveNow],
  );

  return { saveThrottled, saveNow };
}
