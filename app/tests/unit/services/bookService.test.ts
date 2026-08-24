import { closeDatabase, initDatabase } from '@/db/client';
import { getLocalBook, saveBook } from '@/db/repositories/bookRepository';
import {
  SingleBookViolationError,
  deleteCurrentBook,
  startBookUpload,
} from '@/services/bookService';

const EPUB_FILE = {
  uri: 'file:///cache/test-book.epub',
  name: 'test-book.epub',
  mimeType: 'application/epub+zip',
};

function mockUploadSuccess(): void {
  (global.fetch as jest.Mock).mockResolvedValueOnce({
    ok: 202,
    status: 202,
    text: async () =>
      JSON.stringify({
        book_id: 'abc123',
        job_id: 'job123',
        status: 'queued',
      }),
  });
}

function mockDelete(status: number): void {
  (global.fetch as jest.Mock).mockResolvedValueOnce({
    ok: status < 300,
    status,
    text: async () =>
      JSON.stringify(status < 300 ? { deleted: true, id: 'abc123' } : { detail: 'boom' }),
  });
}

describe('bookService', () => {
  beforeEach(async () => {
    (global.fetch as jest.Mock).mockClear();
    closeDatabase();
    await initDatabase();
  });

  afterEach(() => {
    closeDatabase();
  });

  describe('startBookUpload', () => {
    it('uploads the file and persists book + job locally', async () => {
      mockUploadSuccess();
      const result = await startBookUpload(EPUB_FILE);
      expect(result.bookId).toBe('abc123');
      expect(result.jobId).toBe('job123');

      const local = getLocalBook();
      expect(local?.id).toBe('abc123');
      expect(local?.status).toBe('queued');
      expect(local?.sourceType).toBe('epub');
    });

    it('rejects unsupported file types before contacting the server', async () => {
      await expect(
        startBookUpload({ ...EPUB_FILE, name: 'virus.exe', mimeType: 'application/octet-stream' }),
      ).rejects.toThrow(/Unsupported file type/);
      expect(global.fetch).not.toHaveBeenCalled();
      expect(getLocalBook()).toBeNull();
    });

    it('enforces the single-book rule at the service level', async () => {
      saveBook({ id: 'existing', title: 'Existing', status: 'completed' });
      await expect(startBookUpload(EPUB_FILE)).rejects.toBeInstanceOf(
        SingleBookViolationError,
      );
      // No upload happened.
      expect(global.fetch).not.toHaveBeenCalled();
      expect(getLocalBook()?.id).toBe('existing');
    });

    it('does not persist anything when the backend rejects the upload', async () => {
      (global.fetch as jest.Mock).mockResolvedValueOnce({
        ok: false,
        status: 415,
        text: async () => JSON.stringify({ detail: 'unsupported' }),
      });
      await expect(startBookUpload(EPUB_FILE)).rejects.toMatchObject({
        status: 415,
      });
      expect(getLocalBook()).toBeNull();
    });
  });

  describe('deleteCurrentBook', () => {
    it('deletes the backend book first; local cleanup happens after success', async () => {
      saveBook({ id: 'abc123', title: 'B', status: 'completed' });
      mockDelete(500); // backend failure
      await expect(deleteCurrentBook()).rejects.toMatchObject({ status: 500 });
      // Local record must NOT be silently deleted on backend failure.
      expect(getLocalBook()?.id).toBe('abc123');

      mockDelete(200);
      await expect(deleteCurrentBook()).resolves.not.toThrow();
    });

    it('is a no-op when no book exists', async () => {
      await expect(deleteCurrentBook()).resolves.toBeUndefined();
      expect(global.fetch).not.toHaveBeenCalled();
    });
  });
});
