import {
  createBookFromText,
  deleteBook,
  getBook,
  getBooks,
  uploadBook,
} from '@/services/api/books';
import bookFixture from '../../../fixtures/book.json';
import bookDetailFixture from '../../../fixtures/book-detail.json';
import createBookFixture from '../../../fixtures/create-book.json';

function mockJson(status: number, body: unknown): void {
  (global.fetch as jest.Mock).mockResolvedValueOnce({
    ok: status >= 200 && status < 300,
    status,
    text: async () => JSON.stringify(body),
  });
}

describe('books API', () => {
  describe('createBookFromText', () => {
    it('POSTs raw text and parses the response', async () => {
      mockJson(202, createBookFixture);
      const response = await createBookFromText({
        title: 'Example',
        text: 'The quick brown fox.',
      });
      expect(response).toEqual(createBookFixture);
      const [url, init] = (global.fetch as jest.Mock).mock.calls[0];
      expect(url).toBe('http://localhost:8000/api/books');
      expect(init.method).toBe('POST');
      expect(JSON.parse(init.body)).toEqual({
        title: 'Example',
        text: 'The quick brown fox.',
      });
    });

    it('surfaces backend validation errors (422)', async () => {
      mockJson(422, { detail: 'title required' });
      await expect(
        createBookFromText({ title: '', text: '' }),
      ).rejects.toThrow(/title required/);
    });
  });

  describe('uploadBook', () => {
    it('uploads multipart form data', async () => {
      mockJson(202, createBookFixture);
      const response = await uploadBook(
        { uri: 'file:///tmp/book.epub', name: 'book.epub', mimeType: 'application/epub+zip' },
        { title: 'My Book' },
      );
      expect(response.book_id).toBe('abc123');

      const calls = (global.fetch as jest.Mock).mock.calls;
      const [url, init] = calls[calls.length - 1];
      expect(url).toBe('http://localhost:8000/api/books/upload');
      expect(init.method).toBe('POST');
      expect(init.body).toBeInstanceOf(FormData);
      const form = init.body as FormData;
      expect(form.get('title')).toBe('My Book');
    });

    it('rejects unsupported media types (415)', async () => {
      mockJson(415, { detail: 'unsupported file type' });
      await expect(
        uploadBook({ uri: 'file:///tmp/x.docx', name: 'x.docx', mimeType: 'application/msword' }),
      ).rejects.toThrow(/unsupported file type/);
    });
  });

  describe('getBooks', () => {
    it('parses the list payload', async () => {
      mockJson(200, bookFixture);
      const response = await getBooks();
      expect(response.books).toHaveLength(1);
      expect(response.books[0].id).toBe('abc123');
    });
  });

  describe('getBook', () => {
    it('fetches full detail with chapters and chunks', async () => {
      mockJson(200, bookDetailFixture);
      const detail = await getBook('abc123');
      expect(detail.chapters).toHaveLength(2);
      expect(detail.chapters[0].chunks[0].audio_url).toContain('/api/files/');
    });

    it('throws ApiError 404 for missing books', async () => {
      mockJson(404, { detail: 'Not Found' });
      const promise = getBook('missing');
      await expect(promise).rejects.toMatchObject({ status: 404 });
    });
  });

  describe('deleteBook', () => {
    it('DELETEs the book and parses the confirmation', async () => {
      mockJson(200, { deleted: true, id: 'abc123' });
      const response = await deleteBook('abc123');
      expect(response.deleted).toBe(true);
      const calls = (global.fetch as jest.Mock).mock.calls;
      const [url, init] = calls[calls.length - 1];
      expect(url).toBe('http://localhost:8000/api/books/abc123');
      expect(init.method).toBe('DELETE');
    });

    it('propagates server errors (500)', async () => {
      mockJson(500, { detail: 'boom' });
      await expect(deleteBook('abc123')).rejects.toMatchObject({ status: 500 });
    });
  });
});
