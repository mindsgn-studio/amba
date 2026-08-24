import {
  BookSchema,
  BookDetailSchema,
  BookListResponseSchema,
  ChapterWithChunksSchema,
  ChunkSchema,
  CreateBookResponseSchema,
  DeleteBookResponseSchema,
} from '@/schemas/book';
import bookFixture from '../../fixtures/book.json';
import bookDetailFixture from '../../fixtures/book-detail.json';

describe('BookSchema', () => {
  it('accepts a valid book', () => {
    const result = BookSchema.safeParse(bookFixture.books[0]);
    expect(result.success).toBe(true);
  });

  it('rejects invalid status values', () => {
    const result = BookSchema.safeParse({
      ...bookFixture.books[0],
      status: 'unknown-status',
    });
    expect(result.success).toBe(false);
  });

  it('rejects missing id', () => {
    const { id, ...withoutId } = bookFixture.books[0];
    void id;
    const result = BookSchema.safeParse(withoutId);
    expect(result.success).toBe(false);
  });

  it('rejects non-string timestamps', () => {
    const result = BookSchema.safeParse({
      ...bookFixture.books[0],
      created_at: 12345,
    });
    expect(result.success).toBe(false);
  });
});

describe('BookListResponseSchema', () => {
  it('accepts the list payload', () => {
    expect(BookListResponseSchema.safeParse(bookFixture).success).toBe(true);
  });

  it('rejects when books is not an array', () => {
    expect(BookListResponseSchema.safeParse({ books: {} }).success).toBe(false);
  });
});

describe('ChunkSchema', () => {
  it('accepts valid chunks', () => {
    const chunk = bookDetailFixture.chapters[0].chunks[0];
    expect(ChunkSchema.safeParse(chunk).success).toBe(true);
  });

  it('allows missing duration', () => {
    const chunk = { ...bookDetailFixture.chapters[0].chunks[0], duration: undefined };
    expect(ChunkSchema.safeParse(chunk).success).toBe(true);
  });
});

describe('ChapterWithChunksSchema', () => {
  it('accepts chapters with nested chunks', () => {
    expect(
      ChapterWithChunksSchema.safeParse(bookDetailFixture.chapters[0]).success,
    ).toBe(true);
  });

  it('requires the chunks array', () => {
    const { chunks, ...chapter } = bookDetailFixture.chapters[0];
    void chunks;
    expect(ChapterWithChunksSchema.safeParse(chapter).success).toBe(false);
  });
});

describe('BookDetailSchema', () => {
  it('accepts the full detail payload', () => {
    expect(BookDetailSchema.safeParse(bookDetailFixture).success).toBe(true);
  });

  it('accepts minimal payloads without descriptive fields', () => {
    const minimal = {
      id: 'abc123',
      title: 'Example Book',
      status: 'completed',
      chapters: [],
    };
    expect(BookDetailSchema.safeParse(minimal).success).toBe(true);
  });
});

describe('CreateBookResponseSchema', () => {
  it('accepts queued responses', () => {
    expect(
      CreateBookResponseSchema.safeParse({
        book_id: 'abc123',
        job_id: 'job123',
        status: 'queued',
      }).success,
    ).toBe(true);
  });

  it('rejects empty ids', () => {
    expect(
      CreateBookResponseSchema.safeParse({ book_id: '', job_id: 'j', status: 'queued' })
        .success,
    ).toBe(false);
  });
});

describe('DeleteBookResponseSchema', () => {
  it('accepts deletion confirmations', () => {
    expect(
      DeleteBookResponseSchema.safeParse({ deleted: true, id: 'abc123' }).success,
    ).toBe(true);
  });

  it('rejects malformed payloads', () => {
    expect(
      DeleteBookResponseSchema.safeParse({ deleted: 'yes', id: 1 }).success,
    ).toBe(false);
  });
});
