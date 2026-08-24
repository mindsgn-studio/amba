import {
  BookDetailSchema,
  BookListResponseSchema,
  CreateBookResponseSchema,
  DeleteBookResponseSchema,
  type BookDetail,
  type BookListResponse,
  type CreateBookResponse,
  type DeleteBookResponse,
} from '@/schemas/book';
import { apiRequest } from '@/services/api/client';

export interface UploadFileInput {
  uri: string;
  name: string;
  mimeType: string;
}

/** POST /api/books — create an audiobook from raw text. */
export function createBookFromText(
  input: { title: string; text: string; language?: string },
): Promise<CreateBookResponse> {
  return apiRequest(
    '/api/books',
    CreateBookResponseSchema,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        title: input.title,
        text: input.text,
        ...(input.language ? { language: input.language } : {}),
      }),
    },
  );
}

/** POST /api/books/upload — create an audiobook from a picked document. */
export async function uploadBook(
  file: UploadFileInput,
  options: { title?: string; language?: string } = {},
): Promise<CreateBookResponse> {
  const formData = new FormData();
  formData.append('file', {
    uri: file.uri,
    name: file.name,
    type: file.mimeType,
  } as unknown as Blob);

  if (options.title) {
    formData.append('title', options.title);
  }
  if (options.language) {
    formData.append('language', options.language);
  }

  return apiRequest('/api/books/upload', CreateBookResponseSchema, {
    method: 'POST',
    body: formData,
  });
}

/** GET /api/books */
export function getBooks(): Promise<BookListResponse> {
  return apiRequest('/api/books', BookListResponseSchema);
}

/** GET /api/books/{book_id} */
export function getBook(bookId: string): Promise<BookDetail> {
  return apiRequest(
    `/api/books/${encodeURIComponent(bookId)}`,
    BookDetailSchema,
  );
}

/** DELETE /api/books/{book_id} */
export function deleteBook(bookId: string): Promise<DeleteBookResponse> {
  return apiRequest(
    `/api/books/${encodeURIComponent(bookId)}`,
    DeleteBookResponseSchema,
    { method: 'DELETE' },
  );
}
