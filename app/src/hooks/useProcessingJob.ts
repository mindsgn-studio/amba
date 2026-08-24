import { useCallback, useEffect, useRef, useState } from 'react';
import { env } from '@/config/env';
import type { Job } from '@/schemas/job';
import { getJob } from '@/services/api/jobs';
import { persistJobUpdate } from '@/services/bookService';
import { syncBookFromBackend } from '@/services/reader/readerService';
import { useBookStore } from '@/store/bookStore';
import { useProcessingStore } from '@/store/processingStore';

const POLL_INTERVAL_MS = env.jobPollIntervalMs;
const MAX_CONSECUTIVE_FAILURES = 3;

function isActiveStatus(status: Job['status'] | null): boolean {
  return status === 'queued' || status === 'processing';
}

/**
 * Polls GET /api/jobs/{job_id} every few seconds while the job is active,
 * persists snapshots to SQLite for crash recovery, and finalizes the book
 * when the job completes or fails.
 */
export function useProcessingJob(): { connectionIssues: boolean } {
  const jobId = useProcessingStore((state) => state.jobId);
  const status = useProcessingStore((state) => state.status);
  const [connectionIssues, setConnectionIssues] = useState(false);
  const failuresRef = useRef(0);

  const handleJobUpdate = useCallback(async (job: Job): Promise<void> => {
    failuresRef.current = 0;
    setConnectionIssues(false);

    const processingStore = useProcessingStore.getState();
    processingStore.setStatus(job.status);
    processingStore.setProgress(job.progress);

    // Persist snapshot so a restart can resume polling.
    persistJobUpdate({
      jobId: job.id,
      bookId: job.book_id,
      status: job.status,
      progress: job.progress,
      error: job.error,
    });

    if (job.status === 'completed') {
      await syncBookFromBackend(job.book_id);
      const bookStore = useBookStore.getState();
      if (bookStore.bookId === job.book_id) {
        bookStore.updateStatus('completed');
      }
      useProcessingStore.getState().setStatus('completed');
    } else if (job.status === 'failed') {
      useProcessingStore
        .getState()
        .fail(job.error ?? 'Unable to generate this audiobook.');
    }
  }, []);

  useEffect(() => {
    if (!jobId || !isActiveStatus(status)) {
      return undefined;
    }

    let cancelled = false;
    let timer: ReturnType<typeof setTimeout> | null = null;

    const poll = async (): Promise<void> => {
      try {
        const job = await getJob(jobId);
        if (cancelled) {
          return;
        }
        await handleJobUpdate(job);
      } catch {
        if (cancelled) {
          return;
        }
        failuresRef.current += 1;
        if (failuresRef.current >= MAX_CONSECUTIVE_FAILURES) {
          setConnectionIssues(true);
        }
      }

      if (
        !cancelled &&
        isActiveStatus(useProcessingStore.getState().status)
      ) {
        timer = setTimeout(() => void poll(), POLL_INTERVAL_MS);
      }
    };

    void poll();

    return () => {
      cancelled = true;
      if (timer) {
        clearTimeout(timer);
      }
    };
  }, [jobId, status, handleJobUpdate]);

  return { connectionIssues };
}
