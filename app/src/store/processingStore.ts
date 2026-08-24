import { create } from 'zustand';
import type { JobStatus } from '@/schemas/job';

interface ProcessingState {
  jobId: string | null;
  bookId: string | null;
  status: JobStatus | null;
  progress: number;
  error: string | null;
  startProcessing: (input: {
    jobId: string;
    bookId: string;
    status: JobStatus;
  }) => void;
  setStatus: (status: JobStatus) => void;
  setProgress: (progress: number) => void;
  setError: (error: string | null) => void;
  complete: () => void;
  fail: (error: string) => void;
  reset: () => void;
}

const initialState = {
  jobId: null,
  bookId: null,
  status: null as JobStatus | null,
  progress: 0,
  error: null as string | null,
};

export const useProcessingStore = create<ProcessingState>((set) => ({
  ...initialState,
  startProcessing: ({ jobId, bookId, status }) =>
    set({ jobId, bookId, status, progress: 0, error: null }),
  setStatus: (status) => set({ status }),
  setProgress: (progress) => set({ progress: Math.max(0, Math.min(100, progress)) }),
  setError: (error) => set({ error }),
  complete: () =>
    set({ status: 'completed', progress: 100, error: null }),
  fail: (error) => set({ status: 'failed', error }),
  reset: () => set({ ...initialState }),
}));
