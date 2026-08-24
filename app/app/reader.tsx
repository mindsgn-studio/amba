import React, { useCallback } from 'react';
import { useRouter } from 'expo-router';
import { StyleSheet, View } from 'react-native';
import { ChapterHeader } from '@/components/reader/ChapterHeader';
import {
  PlaybackControls,
  PLAYBACK_RATES,
} from '@/components/reader/PlaybackControls';
import { ReaderHeader } from '@/components/reader/ReaderHeader';
import { ReaderText } from '@/components/reader/ReaderText';
import { ReadingProgress } from '@/components/reader/ReadingProgress';
import { ErrorState } from '@/components/ui/ErrorState';
import { Loading } from '@/components/ui/Loading';
import { spacing } from '@/components/ui/theme';
import { useReader } from '@/hooks/useReader';
import { useBookStore } from '@/store/bookStore';
import { useReaderStore } from '@/store/readerStore';

/**
 * The read-along experience. Composes hooks + small components; all
 * playback/timing/persistence logic lives in useReader.
 */
export default function ReaderScreen(): React.JSX.Element {
  const router = useRouter();
  const bookId = useBookStore((state) => state.bookId);

  const readerStatus = useReaderStore((state) => state.readerStatus);
  const error = useReaderStore((state) => state.error);
  const chapterTitle = useReaderStore((state) => state.chapterTitle);
  const chapterIndex = useReaderStore((state) => state.chapterIndex);
  const isPlaying = useReaderStore((state) => state.isPlaying);
  const rate = useReaderStore((state) => state.playbackRate);

  const { togglePlayback, seekBy, changeRate, exit } = useReader(bookId);

  const handleExit = useCallback(async (): Promise<void> => {
    // Stop behaviour: save progress, stop audio, release resources.
    await exit();
    if (router.canDismiss()) {
      router.dismiss();
    }
    router.replace('/');
  }, [exit, router]);

  const nextRate = useCallback((): number => {
    const index = PLAYBACK_RATES.indexOf(rate as (typeof PLAYBACK_RATES)[number]);
    const next = PLAYBACK_RATES[(index + 1) % PLAYBACK_RATES.length];
    return next ?? PLAYBACK_RATES[0];
  }, [rate]);

  const busy = readerStatus === 'loading' || readerStatus === 'idle';

  return (
    <View style={styles.container} testID="reader-screen">
      <ReaderHeader title={chapterTitle} onExit={() => void handleExit()} />

      {error ? (
        <ErrorState
          message={error}
          actionLabel="Back to Home"
          onAction={() => void handleExit()}
        />
      ) : busy ? (
        <Loading message="Loading your audiobook…" />
      ) : (
        <>
          <ChapterHeader chapterIndex={chapterIndex} title={chapterTitle} />
          <ReaderText />
          <ReadingProgress />
          <PlaybackControls
            isPlaying={isPlaying}
            disabled={readerStatus === 'completed'}
            rate={rate}
            onTogglePlayPause={togglePlayback}
            onSeekBackward={() => void seekBy(-15)}
            onSeekForward={() => void seekBy(15)}
            onRateChange={() => void changeRate(nextRate())}
          />
        </>
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    paddingHorizontal: spacing.lg,
    paddingTop: spacing.xl + 8,
    paddingBottom: spacing.md,
    gap: spacing.sm,
  },
});
