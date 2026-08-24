import { TimingSchema, WordTimingSchema } from '@/schemas/timing';
import timingFixture from '../../fixtures/timing.json';

describe('WordTimingSchema', () => {
  it('accepts valid words', () => {
    const word = { index: 0, text: 'The', start: 0.0, end: 0.21 };
    expect(WordTimingSchema.safeParse(word).success).toBe(true);
  });

  it('rejects negative indices', () => {
    expect(
      WordTimingSchema.safeParse({ index: -1, text: 'x', start: 0, end: 1 })
        .success,
    ).toBe(false);
  });

  it('rejects non-numeric times', () => {
    expect(
      WordTimingSchema.safeParse({ index: 0, text: 'x', start: '0', end: 1 })
        .success,
    ).toBe(false);
  });
});

describe('TimingSchema', () => {
  it('accepts the backend timing fixture', () => {
    const result = TimingSchema.parse(timingFixture);
    // The displayed text must be exactly what the backend supplied.
    expect(result.text).toBe(timingFixture.text);
    expect(result.words).toHaveLength(14);
  });

  it('rejects timings without words', () => {
    expect(
      TimingSchema.safeParse({ chunk_id: 'c1', text: 'hi', words: [] }).success,
    ).toBe(false);
  });

  it('rejects malformed API responses (words not an array)', () => {
    expect(
      TimingSchema.safeParse({ chunk_id: 'c1', text: 'hi', words: 'nope' })
        .success,
    ).toBe(false);
  });

  it('rejects missing text', () => {
    const { text, ...rest } = timingFixture;
    void text;
    expect(TimingSchema.safeParse(rest).success).toBe(false);
  });
});
