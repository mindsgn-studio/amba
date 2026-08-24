import { findCurrentWord, validateTimingWords } from '@/utils/timing';
import type { WordTiming } from '@/schemas/timing';

const words: WordTiming[] = [
  { index: 0, text: 'The', start: 0.0, end: 0.2 },
  { index: 1, text: 'quick', start: 0.21, end: 0.5 },
  { index: 2, text: 'fox', start: 0.51, end: 0.8 },
];

describe('findCurrentWord', () => {
  describe('core highlight matrix', () => {
    it.each([
      [0.0, 0], // first word starts
      [0.1, 0],
      [0.199, 0],
      [0.2, -1], // exactly at word end → no active word
      [0.205, -1], // gap between words
      [0.21, 1],
      [0.4, 1],
      [0.5, -1], // boundary again
      [0.51, 2],
      [0.799, 2],
      [0.8, -1],
      [0.9, -1], // after last word
    ])('time %p → word index %p', (time, expected) => {
      expect(findCurrentWord(words, time, -1)).toBe(expected);
    });
  });

  describe('cursor maintenance (no full scan)', () => {
    it('advances from the previous index', () => {
      expect(findCurrentWord(words, 0.55, 1)).toBe(2);
    });

    it('stays on the current word', () => {
      expect(findCurrentWord(words, 0.3, 1)).toBe(1);
    });
  });

  describe('backward seeking', () => {
    it('recalculates when seeking far back from a high cursor', () => {
      expect(findCurrentWord(words, 0.05, 2)).toBe(0);
    });

    it('lands in the middle when seeking backwards', () => {
      expect(findCurrentWord(words, 0.35, 2)).toBe(1);
    });

    it('returns -1 for a backwards seek into a gap', () => {
      expect(findCurrentWord(words, 0.203, 2)).toBe(-1);
    });
  });

  describe('boundaries', () => {
    it('handles empty arrays', () => {
      expect(findCurrentWord([], 0.5, -1)).toBe(-1);
    });

    it('handles single word timing', () => {
      const single: WordTiming[] = [{ index: 0, text: 'Hi', start: 1, end: 2 }];
      expect(findCurrentWord(single, 0.99, -1)).toBe(-1);
      expect(findCurrentWord(single, 1.5, -1)).toBe(0);
      expect(findCurrentWord(single, 2, -1)).toBe(-1);
    });

    it('clamps out-of-range cursors instead of crashing', () => {
      expect(findCurrentWord(words, 0.05, 99)).toBe(0);
      expect(findCurrentWord(words, 0.05, -5)).toBe(0);
    });
  });

  describe('gaps inside the list', () => {
    const gapped: WordTiming[] = [
      { index: 0, text: 'a', start: 0, end: 1 },
      { index: 1, text: 'b', start: 3, end: 4 }, // long silence 1–3s
    ];

    it('returns -1 inside the gap with a stale cursor', () => {
      expect(findCurrentWord(gapped, 2, 0)).toBe(-1);
    });

    it('returns -1 inside the gap without a cursor', () => {
      expect(findCurrentWord(gapped, 2, -1)).toBe(-1);
    });

    it('finds the next word after the gap', () => {
      expect(findCurrentWord(gapped, 3.5, 0)).toBe(1);
    });
  });

  it('is consistent between fresh search and cursor search', () => {
    for (let t = 0; t <= 1.0; t += 0.01) {
      const fresh = findCurrentWord(words, t, -1);
      const prev = Math.max(0, Math.min(words.length - 1, Math.floor(t * 3)));
      expect(findCurrentWord(words, t, prev)).toBe(fresh);
    }
  });
});

describe('validateTimingWords', () => {
  it('accepts well-formed timings', () => {
    expect(() => validateTimingWords(words)).not.toThrow();
  });

  it('rejects start >= end', () => {
    const invalid: WordTiming[] = [{ index: 0, text: 'x', start: 1, end: 1 }];
    expect(() => validateTimingWords(invalid)).toThrow(/start must be < end/);
  });

  it('rejects out-of-order indices', () => {
    const invalid: WordTiming[] = [
      { index: 1, text: 'b', start: 0, end: 0.5 },
      { index: 0, text: 'a', start: 0.6, end: 1 },
    ];
    expect(() => validateTimingWords(invalid)).toThrow(/ordered by index/);
  });

  it('accepts monotonic timestamps across words', () => {
    const ok: WordTiming[] = [
      { index: 0, text: 'a', start: 0, end: 0.5 },
      { index: 1, text: 'b', start: 0.5, end: 1 },
    ];
    expect(() => validateTimingWords(ok)).not.toThrow();
  });

  it('rejects non-monotonic timestamps', () => {
    const invalid: WordTiming[] = [
      { index: 0, text: 'a', start: 0.5, end: 1 },
      { index: 1, text: 'b', start: 0.2, end: 0.3 },
    ];
    expect(() => validateTimingWords(invalid)).toThrow(/monotonic/);
  });
});
