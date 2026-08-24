import { create } from 'zustand';
import type { BookStatus } from '@/schemas/book';

interface BookState {
  bookId: string | null;
  title: string | null;
  author: string | null;
  language: string | null;
  sourceType: string | null;
  status: BookStatus | null;
  /** True when a saved reading position exists for this book. */
  hasProgress: boolean;
  /** True when the reader reached the end of the final chunk. */
  completed: boolean;
  setBook: (book: {
    bookId: string;
    title: string;
    author?: string | null;
    language?: string | null;
    sourceType?: string | null;
    status: BookStatus;
  }) => void;
  updateStatus: (status: BookStatus) => void;
  setHasProgress: (hasProgress: boolean) => void;
  markCompleted: () => void;
  clear: () => void;
}

const initialState = {
  bookId: null,
  title: null,
  author: null,
  language: null,
  sourceType: null,
  status: null as BookStatus | null,
  hasProgress: false,
  completed: false,
};

export const useBookStore = create<BookState>((set) => ({
  ...initialState,
  setBook: (book) =>
    set({
      bookId: book.bookId,
      title: book.title,
      author: book.author ?? null,
      language: book.language ?? null,
      sourceType: book.sourceType ?? null,
      status: book.status,
    }),
  updateStatus: (status) => set({ status }),
  setHasProgress: (hasProgress) => set({ hasProgress }),
  markCompleted: () => set({ completed: true }),
  clear: () => set({ ...initialState }),
}));
