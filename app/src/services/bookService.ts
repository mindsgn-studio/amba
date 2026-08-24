import type { UploadFileInput } from '@/services/api/books';
import { deleteBook as apiDeleteBook, uploadBook } from '@/services/api/books';
import {
  getLocalBook,
  saveJob,
  saveBook,
} from '@/db/repositories/bookRepository';
import { isSupportedUploadFile } from '@/config/env';

export class SingleBookViolationError extends Error {
  constructor() {
    super('A book already exists. Delete it before adding another one.');
    this.name = 'SingleBookViolationError';
  }
}

export class UnsupportedFileTypeError extends Error {
  constructor(fileName: string) {
    super(`Unsupported file type: ${fileName}`);
    this.name = 'UnsupportedFileTypeError';
  }
}

/**
 * Starts a new book upload. Enforces the single-book rule at the service
 * level (in addition to the UI and the database layer).
 */
export async function startBookUpload(
  file: UploadFileInput,
): Promise<{ bookId: string; jobId: string }> {
  if (!isSupportedUploadFile(file.name)) {
    throw new UnsupportedFileTypeError(file.name);
  }
  if (getLocalBook()) {
    throw new SingleBookViolationError();
  }

  const titleFromName = file.name.replace(/\.[^.]+$/, '');
  const response = await uploadBook(file, { title: titleFromName });

  saveBook({
    id: response.book_id,
    title: titleFromName,
    sourceType: extensionOf(file.name),
    status: 'queued',
  });
  saveJob({
    id: response.job_id,
    bookId: response.book_id,
    status: response.status,
  });

  return { bookId: response.book_id, jobId: response.job_id };
}

function extensionOf(fileName: string): string {
  const parts = fileName.split('.');
  return parts.length > 1 ? parts[parts.length - 1].toLowerCase() : 'txt';
}

/**
 * Deletes the current book everywhere. The backend is deleted first; if
 * that fails nothing is removed locally so the app never silently loses
 * track of a server-side book.
 */
export async function deleteCurrentBook(): Promise<void> {
  const local = getLocalBook();
  if (!local) {
    return;
  }
  await apiDeleteBook(local.id);
}

/** Marks the local job record as terminal (completed/failed). */
export function persistJobUpdate(input: {
  jobId: string;
  bookId: string;
  status: string;
  progress?: number;
  error?: string | null;
}): void {
  saveJob({
    id: input.jobId,
    bookId: input.bookId,
    status: input.status,
    progress: input.progress,
    error: input.error,
  });
}
