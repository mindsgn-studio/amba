import type { WordTiming } from '@/schemas/timing';

/**
 * Returns the index of the word that is active at `time`, or -1 when the
 * time falls in a gap between words, before the first word, or after the
 * last word.
 *
 * A word is active when `word.start <= time < word.end`.
 *
 * The algorithm keeps a persistent cursor (`previousIndex`) and only scans
 * forward from it, so repeated calls during playback are O(1). When time
 * moves backwards (seek), a binary search is used instead.
 */
export function findCurrentWord(
  words: readonly WordTiming[],
  time: number,
  previousIndex: number,
): number {
  if (words.length === 0) {
    return -1;
  }

  if (
    previousIndex < 0 ||
    previousIndex >= words.length ||
    time < words[previousIndex].start
  ) {
    return findCurrentWordFromStart(words, time);
  }

  let index = previousIndex;
  while (index < words.length && words[index].end <= time) {
    index += 1;
  }

  if (index >= words.length) {
    return -1;
  }

  const word = words[index];
  return word.start <= time ? index : -1;
}

function findCurrentWordFromStart(
  words: readonly WordTiming[],
  time: number,
): number {
  let low = 0;
  let high = words.length - 1;
  let candidate = -1;

  while (low <= high) {
    const mid = Math.floor((low + high) / 2);
    if (words[mid].start <= time) {
      candidate = mid;
      low = mid + 1;
    } else {
      high = mid - 1;
    }
  }

  if (candidate === -1) {
    return -1;
  }

  return words[candidate].end > time ? candidate : -1;
}

export interface TimingValidationIssue {
  message: string;
}

/**
 * Semantic guarantees for backend timing files:
 * - words are ordered by index and by time
 * - start < end for every word
 * - timestamps are monotonic across the chunk
 */
export function validateTimingWords(words: readonly WordTiming[]): void {
  for (let i = 0; i < words.length; i += 1) {
    const word = words[i];
    assert(word.start < word.end, `word ${word.index}: start must be < end`);

    if (i > 0) {
      const previous = words[i - 1];
      assert(
        word.index > previous.index,
        'words must be ordered by index',
      );
      assert(
        word.start >= previous.end || word.start >= previous.start,
        'word timestamps must be monotonic',
      );
    }
  }
}

function assert(condition: boolean, message: string): void {
  if (!condition) {
    throw new Error(`Invalid timing data: ${message}`);
  }
}
