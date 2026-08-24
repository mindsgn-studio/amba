import { useCallback, useMemo } from 'react';
import * as DocumentPicker from 'expo-document-picker';
import type { UploadExtension } from '@/config/env';
import {
  deleteCurrentBook,
  startBookUpload,
} from '@/services/bookService';
import { clearProgress } from '@/services/reader/readerService';
import { useBookStore } from '@/store/bookStore';
import { useProcessingStore } from '@/store/processingStore';
import type { BookStatus } from '@/schemas/book';

/** Derived Home state, computed from persistent book state. */
export type HomeState =
  | 'NO_BOOK'
  | 'BOOK_PROCESSING'
  | 'BOOK_READY_NOT_STARTED'
  | 'BOOK_IN_PROGRESS'
  | 'BOOK_COMPLETED';

const MIME_TYPES: Record<UploadExtension, string> = {
  pdf: 'application/pdf',
  epub: 'application/epub+zip',
  txt: 'text/plain',
};

function mimeTypeFor(name: string): string {
  const ext = name.toLowerCase().split('.').pop() ?? '';
  return MIME_TYPES[ext as UploadExtension] ?? 'application/octet-stream';
}

interface PickedFile {
  uri: string;
  name: string;
  mimeType: string;
}

async function pickDocument(): Promise<PickedFile | null> {
  const result = await DocumentPicker.getDocumentAsync({
    type: Object.values(MIME_TYPES),
    copyToCacheDirectory: true,
  });
  if (result.canceled) {
    return null;
  }
  const asset = result.assets[0];
  if (!asset) {
    return null;
  }
  return {
    uri: asset.uri,
    name: asset.name ?? 'book.txt',
    mimeType: asset.mimeType ?? mimeTypeFor(asset.name ?? ''),
  };
}

export function deriveHomeState(
  status: BookStatus | null,
  hasProgress: boolean,
  completed: boolean,
): HomeState {
  switch (status) {
    case null:
      return 'NO_BOOK';
    case 'queued':
    case 'processing':
    case 'failed':
      return 'BOOK_PROCESSING';
    case 'completed':
      if (completed) {
        return 'BOOK_COMPLETED';
      }
      return hasProgress ? 'BOOK_IN_PROGRESS' : 'BOOK_READY_NOT_STARTED';
    default:
      return 'NO_BOOK';
  }
}

export function useBook() {
  const bookId = useBookStore((state) => state.bookId);
  const title = useBookStore((state) => state.title);
  const author = useBookStore((state) => state.author);
  const status = useBookStore((state) => state.status);
  const hasProgress = useBookStore((state) => state.hasProgress);
  const completed = useBookStore((state) => state.completed);

  const homeState = useMemo(
    () => deriveHomeState(status, hasProgress, completed),
    [status, hasProgress, completed],
  );

  const uploadDocument =
    useCallback(async (): Promise<{ jobId: string } | null> => {
      const picked = await pickDocument();
      if (!picked) {
        return null;
      }

      const response = await startBookUpload(picked);

      useBookStore.getState().setBook({
        bookId: response.bookId,
        title: picked.name.replace(/\.[^.]+$/, ''),
        status: 'queued',
      });
      useProcessingStore.getState().startProcessing({
        jobId: response.jobId,
        bookId: response.bookId,
        status: 'queued',
      });

      return { jobId: response.jobId };
    }, []);

  const removeBook = useCallback(async (): Promise<void> => {
    const bookStore = useBookStore.getState();
    const currentId = bookStore.bookId;
    if (!currentId) {
      return;
    }

    // Backend first — on failure nothing is removed locally.
    await deleteCurrentBook();

    clearProgress(currentId);
    bookStore.clear();
    useProcessingStore.getState().reset();
  }, []);

  return {
    bookId,
    title,
    author,
    status,
    hasProgress,
    completed,
    homeState,
    uploadDocument,
    removeBook,
    canUpload: status === null,
  };
}
