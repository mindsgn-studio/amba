import { useEffect } from 'react';
import { initDatabase } from '@/db/client';
import { getLocalBook, getPendingJob } from '@/db/repositories/bookRepository';
import { getSavedProgress } from '@/services/reader/readerService';
import { useAppStore } from '@/store/appStore';
import { useBookStore } from '@/store/bookStore';
import { useProcessingStore } from '@/store/processingStore';
import type { BookStatus } from '@/schemas/book';
import type { JobStatus } from '@/schemas/job';

function errorMessage(error: unknown): string {
  return error instanceof Error ? error.message : String(error);
}

/**
 * Application startup sequence:
 *
 * SQLite init → load local book → load persisted job/progress →
 * hydrate Zustand → mark app ready.
 *
 * Runs exactly once, before any main UI is rendered.
 */
export function useAppInitialization(): void {
  useEffect(() => {
    const appStore = useAppStore.getState();
    const bookStore = useBookStore.getState();
    const processingStore = useProcessingStore.getState();

    const run = async () => {
      try {
        initDatabase();

        const book = getLocalBook();
        if (book) {
          bookStore.setBook({
            bookId: book.id,
            title: book.title,
            author: book.author,
            language: book.language,
            sourceType: book.sourceType,
            status: book.status as BookStatus,
          });

          const progress = getSavedProgress(book.id);
          bookStore.setHasProgress(progress !== null);
          if (progress?.completed) {
            bookStore.markCompleted();
          }

          if (book.status === 'queued' || book.status === 'processing') {
            const pendingJob = getPendingJob();
            if (pendingJob && pendingJob.bookId === book.id) {
              processingStore.startProcessing({
                jobId: pendingJob.id,
                bookId: pendingJob.bookId,
                status: pendingJob.status as JobStatus,
              });
              processingStore.setProgress(pendingJob.progress);
            }
          }
        }

        appStore.setStatus('ready');
      } catch (error) {
        appStore.setError(errorMessage(error));
        appStore.setStatus('error');
      }
    };

    void run();
  }, []);
}
