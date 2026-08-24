import React, { useCallback, useState } from 'react';
import { useRouter } from 'expo-router';
import { ScrollView, StyleSheet } from 'react-native';
import { BookCard } from '@/components/books/BookCard';
import { UploadCard } from '@/components/books/UploadCard';
import { ProcessingProgress } from '@/components/processing/ProcessingProgress';
import { Button } from '@/components/ui/Button';
import { ErrorState } from '@/components/ui/ErrorState';
import { AppText } from '@/components/ui/Text';
import { spacing } from '@/components/ui/theme';
import { useBook, type HomeState } from '@/hooks/useBook';
import { useAppStore } from '@/store/appStore';
import { useBookStore } from '@/store/bookStore';
import { useProcessingStore } from '@/store/processingStore';

const SUBTITLES: Partial<Record<HomeState, string>> = {
  BOOK_READY_NOT_STARTED: 'Ready to read',
  BOOK_IN_PROGRESS: 'In progress',
  BOOK_COMPLETED: 'Completed',
};

function bookCardAction(homeState: HomeState): 'continue' | 'start' | 'completed' {
  if (homeState === 'BOOK_COMPLETED') {
    return 'completed';
  }
  return homeState === 'BOOK_READY_NOT_STARTED' ? 'start' : 'continue';
}

export default function HomeScreen(): React.JSX.Element {
  const router = useRouter();
  const appReady = useAppStore((state) => state.status === 'ready');
  const {
    title,
    homeState,
    uploadDocument,
    removeBook,
    status,
  } = useBook();
  const jobProgress = useProcessingStore((state) => state.progress);

  const [uploading, setUploading] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);

  const handleUpload = useCallback(async (): Promise<void> => {
    setActionError(null);
    // Single-book rule enforced again at action level.
    if (useBookStore.getState().bookId) {
      return;
    }
    setUploading(true);
    try {
      const result = await uploadDocument();
      if (result) {
        router.push('/processing');
      }
    } catch (error) {
      setActionError(
        error instanceof Error
          ? `Upload failed: ${error.message}`
          : 'Upload failed. Please try again.',
      );
    } finally {
      setUploading(false);
    }
  }, [router, uploadDocument]);

  const handleDelete = useCallback(async (): Promise<void> => {
    setActionError(null);
    setDeleting(true);
    try {
      await removeBook();
    } catch (error) {
      setActionError(
        error instanceof Error
          ? `Could not delete the book: ${error.message}`
          : 'Could not delete the book.',
      );
    } finally {
      setDeleting(false);
    }
  }, [removeBook]);

  if (!appReady) {
    return <ScrollView contentContainerStyle={styles.container} />;
  }

  let content: React.ReactNode;
  switch (homeState) {
    case 'NO_BOOK':
      content = (
        <UploadCard onUpload={() => void handleUpload()} busy={uploading} />
      );
      break;
    case 'BOOK_PROCESSING':
      content = (
        <>
          <ProcessingProgress
            title={title ?? 'Your book'}
            progress={jobProgress}
            message={
              uploading
                ? 'Uploading book…'
                : status === 'failed'
                  ? 'Something went wrong. Unable to generate this audiobook.'
                  : null
            }
            isError={status === 'failed'}
          />
          {status === 'failed' ? (
            <Button
              label="Delete Book"
              variant="danger"
              onPress={() => void handleDelete()}
              accessibilityLabel="Delete Book"
              testID="delete-book-button"
            />
          ) : (
            <Button
              label="View Progress"
              onPress={() => router.push('/processing')}
              accessibilityLabel="View generation progress"
              testID="view-progress-button"
            />
          )}
        </>
      );
      break;
    default:
      content = (
        <BookCard
          title={title ?? 'Your book'}
          action={bookCardAction(homeState)}
          subtitle={SUBTITLES[homeState]}
          onOpen={() => router.push('/reader')}
          onDelete={() => void handleDelete()}
        />
      );
  }

  return (
    <ScrollView
      style={styles.scroll}
      contentContainerStyle={styles.container}
      testID="home-screen"
    >
      <AppText variant="title" weight="bold" style={styles.heading}>
        Read Along
      </AppText>
      {content}
      {actionError ? (
        <ErrorState
          title={homeState === 'BOOK_PROCESSING' ? 'Delete failed' : 'Upload failed'}
          message={actionError}
        />
      ) : null}
      {deleting ? (
        <AppText variant="muted" testID="deleting-label">
          Deleting book…
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
  heading: {
    marginBottom: spacing.sm,
  },
});
