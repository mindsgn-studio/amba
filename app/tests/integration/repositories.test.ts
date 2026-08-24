import { closeDatabase, getDb, initDatabase } from '@/db/client';
import { books } from '@/db/schema';
import {
  deleteLocalBook,
  getLocalBook,
  getLocalBookById,
  getPendingJob,
  saveBook,
  saveJob,
  updateBookStatus,
} from '@/db/repositories/bookRepository';
import {
  getChapters,
  getChapterById,
  saveChapters,
} from '@/db/repositories/chapterRepository';
import {
  getChunkById,
  getChunks,
  saveChunks,
} from '@/db/repositories/chunkRepository';
import {
  deleteReadingProgress,
  getReadingProgress,
  saveReadingProgress,
} from '@/db/repositories/progressRepository';

describe('repositories', () => {
  beforeEach(async () => {
    closeDatabase();
    await initDatabase();
  });

  afterEach(() => {
    closeDatabase();
  });

  describe('bookRepository', () => {
    it('returns null when no book exists', () => {
      expect(getLocalBook()).toBeNull();
    });

    it('saves and retrieves a book', () => {
      saveBook({ id: 'abc123', title: 'Example Book', status: 'queued' });
      const book = getLocalBook();
      expect(book).not.toBeNull();
      expect(book?.id).toBe('abc123');
      expect(book?.title).toBe('Example Book');
      expect(book?.status).toBe('queued');
    });

    it('updates an existing book without duplicating rows', () => {
      saveBook({ id: 'abc123', title: 'Old Title', status: 'processing' });
      saveBook({ id: 'abc123', title: 'New Title', status: 'completed' });
      const books = getDbBooksCount();
      expect(books).toBe(1);
      expect(getLocalBook()?.title).toBe('New Title');
    });

    it('enforces the single-book rule at the database boundary', () => {
      saveBook({ id: 'first', title: 'First', status: 'completed' });
      expect(() =>
        saveBook({ id: 'second', title: 'Second', status: 'queued' }),
      ).toThrow(/Another book already exists/);
      // Original data untouched.
      expect(getLocalBook()?.id).toBe('first');
    });

    it('looks up by id and updates status', () => {
      saveBook({ id: 'abc123', title: 'B', status: 'queued' });
      updateBookStatus('abc123', 'completed');
      expect(getLocalBookById('abc123')?.status).toBe('completed');
      expect(getLocalBookById('missing')).toBeNull();
    });

    it('deletes the book and cascades to related tables', () => {
      saveBook({ id: 'abc123', title: 'B', status: 'completed' });
      saveChapters([
        { id: 'ch1', bookId: 'abc123', chapterIndex: 0, title: 'Chapter 1' },
      ]);
      saveChunks([
        {
          id: 'ck1',
          chapterId: 'ch1',
          chunkIndex: 0,
          audioUrl: '/api/files/a.mp3',
          timingUrl: '/api/files/a.json',
        },
      ]);
      saveReadingProgress({
        bookId: 'abc123',
        chapterId: 'ch1',
        chunkId: 'ck1',
        positionSeconds: 10,
        wordIndex: 3,
      });
      saveJob({ id: 'job1', bookId: 'abc123', status: 'completed' });

      deleteLocalBook('abc123');

      expect(getLocalBook()).toBeNull();
      expect(getChapterById('ch1')).toBeNull();
      expect(getChunkById('ck1')).toBeNull();
      expect(getReadingProgress('abc123')).toBeNull();
      expect(getPendingJob()).toBeNull();
    });

    describe('jobs', () => {
      beforeEach(() => {
        // Jobs reference books, mirroring the real upload flow.
        saveBook({ id: 'b1', title: 'B1', status: 'queued' });
      });

      it('upserts job snapshots for recovery', () => {
        saveJob({ id: 'job1', bookId: 'b1', status: 'queued', progress: 0 });
        saveJob({ id: 'job1', bookId: 'b1', status: 'processing', progress: 40 });
        expect(getPendingJob()?.progress).toBe(40);
      });

      it('finds pending jobs but ignores terminal ones', () => {
        saveJob({ id: 'job1', bookId: 'b1', status: 'completed', progress: 100 });
        expect(getPendingJob()).toBeNull();
      });
    });
  });

  describe('chapterRepository', () => {
    it('saves chapters in order and replaces on re-save', () => {
      saveBook({ id: 'b1', title: 'B', status: 'completed' });
      saveChapters([
        { id: 'ch2', bookId: 'b1', chapterIndex: 1, title: 'Second' },
        { id: 'ch1', bookId: 'b1', chapterIndex: 0, title: 'First' },
      ]);
      let chapters = getChapters('b1');
      expect(chapters.map((c) => c.title)).toEqual(['First', 'Second']);

      saveChapters([
        { id: 'ch9', bookId: 'b1', chapterIndex: 0, title: 'Only' },
      ]);
      chapters = getChapters('b1');
      expect(chapters).toHaveLength(1);
      expect(chapters[0].title).toBe('Only');
    });

    it('returns an empty list for unknown books', () => {
      expect(getChapters('ghost')).toEqual([]);
    });
  });

  describe('chunkRepository', () => {
    beforeEach(() => {
      saveBook({ id: 'b1', title: 'B', status: 'completed' });
      saveChapters([
        { id: 'ch1', bookId: 'b1', chapterIndex: 0, title: 'One' },
        { id: 'ch2', bookId: 'b1', chapterIndex: 1, title: 'Two' },
      ]);
    });

    it('saves chunks with URLs and duration', () => {
      saveChunks([
        {
          id: 'ck1',
          chapterId: 'ch1',
          chunkIndex: 0,
          duration: 23.4,
          audioUrl: '/api/files/books/b1/c1.mp3',
          timingUrl: '/api/files/books/b1/c1.json',
        },
      ]);
      const chunk = getChunkById('ck1');
      expect(chunk?.duration).toBeCloseTo(23.4);
      expect(chunk?.audioUrl).toContain('/api/files/');
      expect(chunk?.timingUrl).toContain('/api/files/');
    });

    it('orders chunks across chapters by index', () => {
      saveChunks([
        { id: 'ck2', chapterId: 'ch1', chunkIndex: 1 },
        { id: 'ck1', chapterId: 'ch1', chunkIndex: 0 },
        { id: 'ck3', chapterId: 'ch2', chunkIndex: 0 },
      ]);
      const chunks = getChunks(['ch1', 'ch2']);
      expect(chunks.map((c) => c.id)).toEqual(['ck1', 'ck2', 'ck3']);
    });

    it('replaces chunks of a chapter on re-save', () => {
      saveChunks([{ id: 'ck1', chapterId: 'ch1', chunkIndex: 0 }]);
      saveChunks([
        { id: 'ckA', chapterId: 'ch1', chunkIndex: 0 },
        { id: 'ckB', chapterId: 'ch1', chunkIndex: 1 },
      ]);
      const chunks = getChunks(['ch1']);
      expect(chunks.map((c) => c.id)).toEqual(['ckA', 'ckB']);
    });

    it('returns empty for no chapters', () => {
      expect(getChunks([])).toEqual([]);
    });
  });

  describe('progressRepository', () => {
    beforeEach(() => {
      saveBook({ id: 'b1', title: 'B', status: 'completed' });
      saveChapters([{ id: 'ch1', bookId: 'b1', chapterIndex: 0, title: 'One' }]);
      saveChunks([{ id: 'ck1', chapterId: 'ch1', chunkIndex: 0 }]);
    });

    it('starts with zero progress for a new book', () => {
      expect(getReadingProgress('b1')).toBeNull();
    });

    it('saves reading position', () => {
      saveReadingProgress({
        bookId: 'b1',
        chapterId: 'ch1',
        chunkId: 'ck1',
        positionSeconds: 32.41,
        wordIndex: 103,
      });
      const progress = getReadingProgress('b1');
      expect(progress?.positionSeconds).toBeCloseTo(32.41);
      expect(progress?.wordIndex).toBe(103);
      expect(progress?.completed).toBe(0);
      expect(progress?.updatedAt).toBeTruthy();
    });

    it('updates the same row instead of duplicating', () => {
      saveReadingProgress({
        bookId: 'b1',
        chapterId: 'ch1',
        chunkId: 'ck1',
        positionSeconds: 1,
        wordIndex: 0,
      });
      saveReadingProgress({
        bookId: 'b1',
        chapterId: 'ch1',
        chunkId: 'ck1',
        positionSeconds: 5,
        wordIndex: 12,
      });
      const progress = getReadingProgress('b1');
      expect(progress?.positionSeconds).toBeCloseTo(5);
      expect(progress?.wordIndex).toBe(12);
    });

    it('persists completion state', () => {
      saveReadingProgress({
        bookId: 'b1',
        chapterId: 'ch1',
        chunkId: 'ck1',
        positionSeconds: 0,
        wordIndex: 0,
        completed: true,
      });
      expect(getReadingProgress('b1')?.completed).toBe(1);
    });

    it('deletes progress', () => {
      saveReadingProgress({
        bookId: 'b1',
        chapterId: 'ch1',
        chunkId: 'ck1',
        positionSeconds: 1,
        wordIndex: 0,
      });
      deleteReadingProgress('b1');
      expect(getReadingProgress('b1')).toBeNull();
    });

    it('clamps negative values defensively', () => {
      saveReadingProgress({
        bookId: 'b1',
        chapterId: 'ch1',
        chunkId: 'ck1',
        positionSeconds: -5,
        wordIndex: -3,
      });
      const progress = getReadingProgress('b1');
      expect(progress?.positionSeconds).toBeCloseTo(0);
      expect(progress?.wordIndex).toBe(0);
    });
  });
});

function getDbBooksCount(): number {
  return getDb().select().from(books).all().length;
}
