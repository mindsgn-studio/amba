import React, { useCallback, useEffect, useState } from 'react';
import { useRouter } from 'expo-router';
import { ScrollView, StyleSheet } from 'react-native';
import { ProcessingProgress } from '@/components/processing/ProcessingProgress';
import { Button } from '@/components/ui/Button';
import { AppText } from '@/components/ui/Text';
import { spacing } from '@/components/ui/theme';
import { useBook } from '@/hooks/useBook';
import { useProcessingJob } from '@/hooks/useProcessingJob';
import { useBookStore } from '@/store/bookStore';
import { useProcessingStore } from '@/store/processingStore';

/**
 * Audiobook generation screen. Polls the backend job, survives app
 * restarts (job info is persisted in SQLite), and navigates home when
 * the audiobook is ready or shows a clear failure state with delete.
 */
export default function ProcessingScreen(): React.JSX.Element {
  const router = useRouter();
  const jobId = useProcessingStore((state) => state.jobId);
  const jobStatus = useProcessingStore((state) => state.status);
  const progress = useProcessingStore((state) => state.progress);
  const jobError = useProcessingStore((state) => state.error);
  const title = useBookStore((state) => state.title);
  const bookStatus = useBookStore((state) => state.status);

  const { connectionIssues } = useProcessingJob();
  const { removeBook } = useBook();
  const [deleteError, setDeleteError] = useState<string | null>(null);

  // Completed: clear transient processing state and go home.
  useEffect(() => {
    if (jobStatus === 'completed') {
      useProcessingStore.getState().reset();
      router.replace('/');
    }
  }, [jobStatus, router]);

  const failed = jobStatus === 'failed' || bookStatus === 'failed';

  const handleDelete = useCallback(async (): Promise<void> => {
    setDeleteError(null);
    try {
      // removeBook deletes backend first, then local SQLite + stores.
      await removeBook();
      router.replace('/');
    } catch (error) {
      setDeleteError(
        error instanceof Error ? error.message : 'Could not delete the book.',
      );
    }
  }, [removeBook, router]);

  return (
    <ScrollView
      style={styles.scroll}
      contentContainerStyle={styles.container}
      testID="processing-screen"
    >
      <ProcessingProgress
        title={title ?? 'Your book'}
        progress={progress}
        message={
          !jobId && bookStatus === 'completed'
            ? 'This audiobook is ready.'
            : failed
              ? (jobError ?? 'Something went wrong. Unable to generate this audiobook.')
              : connectionIssues
                ? 'Reconnecting to the server…'
                : null
        }
        isError={failed || (!jobId && bookStatus !== 'completed')}
      />

      {!jobId && bookStatus !== 'completed' && !failed ? (
        <>
          <AppText variant="muted">
            No processing job found for this book.
          </AppText>
          <Button label="Go Home" onPress={() => router.replace('/')} />
        </>
      ) : null}

      {failed ? (
        <>
          <Button
            label="Delete Book"
            variant="danger"
            onPress={() => void handleDelete()}
            accessibilityLabel="Delete Book"
            testID="delete-book-button"
          />
          <AppText variant="muted" style={styles.hint}>
            You can upload another book after deleting this one.
          </AppText>
        </>
      ) : null}

      {deleteError ? (
        <AppText variant="muted" style={styles.error}>
          {`Could not delete the book: ${deleteError}`}
        </AppText>
      ) : null}
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  scroll: {
    flex: 1,
  },
  container: {
    alignItems: 'center',
    padding: spacing.lg,
    paddingTop: spacing.xl + 12,
    gap: spacing.lg,
  },
  hint: {
    textAlign: 'center',
  },
  error: {
    color: '#B3402F',
    textAlign: 'center',
  },
});
