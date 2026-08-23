import { readingPassages } from '@/constants/passages';
import {
  buildFeedback,
  compareWords,
  normalizeText,
  scoreReadingSession,
} from '@/features/reading/scoring';

describe('reading scoring', () => {
  it('normalizes punctuation and spaces', () => {
    expect(normalizeText(' Hello,   WORLD! ')).toBe('hello world');
  });

  it('marks missing words without collapsing later alignment', () => {
    expect(compareWords('hello my name is john', 'hello my name john')).toEqual([
      { expected: 'hello', spoken: 'hello', status: 'correct' },
      { expected: 'my', spoken: 'my', status: 'correct' },
      { expected: 'name', spoken: 'name', status: 'correct' },
      { expected: 'is', spoken: null, status: 'missing' },
      { expected: 'john', spoken: 'john', status: 'correct' },
    ]);
  });

  it('scores an MVP reading session', () => {
    const score = scoreReadingSession({
      passage: readingPassages[0],
      transcript: readingPassages[0].text,
      durationSec: 30,
      sessionId: 'session-1',
      userId: 'user-1',
    });

    expect(score.accuracy).toBe(100);
    expect(score.wordsIncorrect).toBe(0);
    expect(score.xpEarned).toBe(score.wordsCorrect);
    expect(score.feedback).toBe(buildFeedback(100));
  });
});
