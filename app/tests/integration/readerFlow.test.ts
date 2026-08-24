import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { closeDatabase, initDatabase } from '@/db/client';
import {
  getLocalBookById,
  saveBook,
  saveJob,
} from '@/db/repositories/bookRepository';
import { saveChapters } from '@/db/repositories/chapterRepository';
import { saveChunks } from '@/db/repositories/chunkRepository';
import {
  getReadingProgress,
  saveReadingProgress,
} from '@/db/repositories/progressRepository';
import type { ChunkRef } from '@/services/reader/readerService';
import {
  loadPlaylist,
  markBookCompletedInDb,
  persistProgress,
  resolveResumeLocation,
  syncBookFromBackend,
} from '@/services/reader/readerService';

function makeRef(chunkId: string, duration: number | null): ChunkRef {
  return {
    chapterId: 'ch1',
    chapterIndex: 0,
    chapterTitle: 'Chapter 1',
    chapterCount: 1,
    chunkId,
    chunkIndex: 0,
    audioUrl: `/api/files/${chunkId}.mp3`,
    timingUrl: `/api/files/${chunkId}.json`,
    duration,
  };
}

function seedCompletedBook(): void {
  saveBook({ id: 'abc123', title: 'Example Book', status: 'completed' });
  saveChapters([
    { id: 'chapter1', bookId: 'abc123', chapterIndex: 0, title: 'Chapter 1' },
    { id: 'chapter2', bookId: 'abc123', chapterIndex: 1, title: 'Chapter 2' },
  ]);
  saveChunks([
    {
      id: 'chunk1',
      chapterId: 'chapter1',
      chunkIndex: 0,
      duration: 10,
      audioUrl: '/api/files/books/abc123/chapters/001/chunk_001.mp3',
      timingUrl: '/api/files/books/abc123/chapters/001/chunk_001.json',
    },
    {
      id: 'chunk2',
      chapterId: 'chapter1',
      chunkIndex: 1,
      duration: 8,
      audioUrl: '/api/files/books/abc123/chapters/001/chunk_002.mp3',
      timingUrl: '/api/files/books/abc123/chapters/001/chunk_002.json',
    },
    {
      id: 'chunk3',
      chapterId: 'chapter2',
      chunkIndex: 0,
      duration: 6,
      audioUrl: '/api/files/books/abc123/chapters/002/chunk_001.mp3',
      timingUrl: '/api/files/books/abc123/chapters/002/chunk_001.json',
    },
  ]);
}

describe('reader flow', () => {
  beforeEach(async () => {
    closeDatabase();
    await initDatabase();
    (global.fetch as jest.Mock).mockReset();
    // Default failed response so tests must set up explicit mocks.
    (global.fetch as jest.Mock).mockResolvedValue({
      ok: false,
      status: 500,
      text: async () => '',
    });
  });

  afterEach(() => {
    closeDatabase();
  });

  describe('loadPlaylist', () => {
    it('builds an ordered playlist across chapters', () => {
      seedCompletedBook();
      const playlist = loadPlaylist('abc123');
      expect(playlist.map((ref) => ref.chunkId)).toEqual([
        'chunk1',
        'chunk2',
        'chunk3',
      ]);
      expect(playlist[0].chapterTitle).toBe('Chapter 1');
      expect(playlist[2].chapterTitle).toBe('Chapter 2');
    });

    it('returns empty when the book detail is not persisted yet', () => {
      expect(loadPlaylist('ghost')).toEqual([]);
    });
  });

  describe('resolveResumeLocation', () => {
    const playlist = [
      makeRef('c1', 10),
      makeRef('c2', 20),
      makeRef('c3', null),
    ];

    it('starts at the beginning for a new book', () => {
      const location = resolveResumeLocation(playlist, null);
      expect(location).toEqual({
        index: 0,
        positionSeconds: 0,
        wordIndex: 0,
        completed: false,
      });
    });

    it('restores a saved position in the middle of the book', () => {
      const progress = {
        bookId: 'b',
        chapterId: 'ch1',
        chunkId: 'c2',
        positionSeconds: 12.5,
        wordIndex: 42,
        completed: 0,
        id: 1,
        updatedAt: 'now',
      };
      const location = resolveResumeLocation(playlist, progress);
      expect(location.index).toBe(1);
      expect(location.positionSeconds).toBeCloseTo(12.5);
      expect(location.wordIndex).toBe(42);
    });

    it('clamps positions beyond chunk duration', () => {
      const progress = {
        bookId: 'b',
        chapterId: 'ch1',
        chunkId: 'c2',
        positionSeconds: 999,
        wordIndex: 0,
        completed: 0,
        id: 1,
        updatedAt: 'now',
      };
      expect(resolveResumeLocation(playlist, progress).positionSeconds).toBeCloseTo(
        19.75,
      );
    });

    it('detects completion on the final chunk', () => {
      const progress = {
        bookId: 'b',
        chapterId: 'ch3',
        chunkId: 'c3',
        positionSeconds: 0,
        wordIndex: 0,
        completed: 1,
        id: 1,
        updatedAt: 'now',
      };
      const location = resolveResumeLocation(playlist, progress);
      expect(location.completed).toBe(true);
    });

    it('falls back to the start when the saved chunk no longer exists', () => {
      const progress = {
        bookId: 'b',
        chapterId: 'chX',
        chunkId: 'gone',
        positionSeconds: 5,
        wordIndex: 3,
        completed: 0,
        id: 1,
        updatedAt: 'now',
      };
      expect(resolveResumeLocation(playlist, progress)).toMatchObject({
        index: 0,
        positionSeconds: 0,
      });
    });
  });

  describe('progress persistence round-trip', () => {
    afterEach(() => {
      delete process.env.EXPO_PUBLIC_DB_NAME;
    });

    it('saves and restores progress across "app restarts"', async () => {
      // Use a temporary FILE-backed database so that closing and reopening
      // it genuinely simulates an app restart instead of losing everything
      // like an in-memory database would.
      const dbPath = path.join(
        os.tmpdir(),
        `amba-readerflow-${process.pid}-${Date.now()}.db`,
      );
      process.env.EXPO_PUBLIC_DB_NAME = dbPath;

      closeDatabase();
      await initDatabase();
      seedCompletedBook();

      // Simulate playback reaching chunk 2 at 4.2s / word 17.
      persistProgress({
        bookId: 'abc123',
        ref: loadPlaylist('abc123')[1],
        positionSeconds: 4.2,
        wordIndex: 17,
      });

      closeDatabase(); // simulate process death
      await initDatabase(); // fresh app launch

      const restored = getReadingProgress('abc123');
      expect(restored?.chunkId).toBe('chunk2');
      expect(restored?.positionSeconds).toBeCloseTo(4.2);
      expect(restored?.wordIndex).toBe(17);

      const playlist = loadPlaylist('abc123');
      const location = resolveResumeLocation(playlist, restored);
      expect(location.index).toBe(1);
      expect(location.positionSeconds).toBeCloseTo(4.2);

      closeDatabase();
      fs.rmSync(dbPath, { force: true });
    });

    it('marks and reads completion', () => {
      seedCompletedBook();
      const playlist = loadPlaylist('abc123');
      markBookCompletedInDb('abc123', playlist[playlist.length - 1]);
      const location = resolveResumeLocation(
        playlist,
        getReadingProgress('abc123'),
      );
      expect(location.completed).toBe(true);
    });

    it('handles chunk transitions by pointing progress at the next chunk', () => {
      seedCompletedBook();
      const playlist = loadPlaylist('abc123');

      persistProgress({
        bookId: 'abc123',
        ref: playlist[0],
        positionSeconds: 9.9,
        wordIndex: 30,
      });
      // Chunk finished → transition saves the *next* chunk at position 0.
      persistProgress({
        bookId: 'abc123',
        ref: playlist[1],
        positionSeconds: 0,
        wordIndex: 0,
      });

      const progress = getReadingProgress('abc123');
      expect(progress?.chunkId).toBe('chunk2');
      expect(progress?.positionSeconds).toBeCloseTo(0);
    });
  });

  describe('syncBookFromBackend', () => {
    it('persists chapters and chunks from the API payload', async () => {
      saveBook({ id: 'abc123', title: 'Example Book', status: 'processing' });
      saveJob({ id: 'job123', bookId: 'abc123', status: 'processing' });

      const detailPayload = {
        id: 'abc123',
        title: 'Example Book',
        status: 'completed',
        chapters: [
          {
            id: 'chapter1',
            chapter_index: 0,
            title: 'Chapter 1',
            chunks: [
              {
                id: 'chunk1',
                chunk_index: 0,
                duration: 23.4,
                audio_url: '/api/files/books/abc123/chapters/001/chunk_001.mp3',
                timing_url: '/api/files/books/abc123/chapters/001/chunk_001.json',
              },
            ],
          },
        ],
      };

      (global.fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        status: 200,
        text: async () => JSON.stringify(detailPayload),
      });

      await syncBookFromBackend('abc123');

      const book = getLocalBookById('abc123');
      expect(book?.status).toBe('completed');
      const playlist = loadPlaylist('abc123');
      expect(playlist).toHaveLength(1);
      expect(playlist[0].audioUrl).toContain('chunk_001.mp3');
    });
  });
});
