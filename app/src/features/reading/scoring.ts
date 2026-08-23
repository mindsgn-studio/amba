import type {
  ReadingPassage,
  ReadingSessionResult,
  ScorePayload,
  WordMatch,
} from '@/types/models';

export function normalizeText(text: string) {
  return text
    .toLowerCase()
    .replace(/[^\p{L}\p{N}'\s]/gu, ' ')
    .replace(/\s+/g, ' ')
    .trim();
}

export function tokenize(text: string) {
  const normalized = normalizeText(text);
  return normalized.length ? normalized.split(' ') : [];
}

export function compareWords(expectedText: string, transcript: string): WordMatch[] {
  const expected = tokenize(expectedText);
  const spoken = tokenize(transcript);

  const matches: WordMatch[] = [];
  let spokenIndex = 0;

  for (let expectedIndex = 0; expectedIndex < expected.length; expectedIndex += 1) {
    const expectedWord = expected[expectedIndex];
    const currentSpoken = spoken[spokenIndex];
    const nextExpected = expected[expectedIndex + 1];

    if (currentSpoken === expectedWord) {
      matches.push({
        expected: expectedWord,
        spoken: currentSpoken,
        status: 'correct',
      });
      spokenIndex += 1;
      continue;
    }

    if (currentSpoken && nextExpected === currentSpoken) {
      matches.push({
        expected: expectedWord,
        spoken: null,
        status: 'missing',
      });
      continue;
    }

    matches.push({
      expected: expectedWord,
      spoken: currentSpoken ?? null,
      status: currentSpoken ? 'incorrect' : 'missing',
    });

    if (currentSpoken) {
      spokenIndex += 1;
    }
  }

  return matches;
}

export function calculateXp(correctWords: number, difficulty: ReadingPassage['difficulty']) {
  const multiplier = difficulty === 'advanced' ? 3 : difficulty === 'intermediate' ? 2 : 1;
  return correctWords * multiplier;
}

export function buildFeedback(accuracy: number) {
  if (accuracy >= 95) {
    return 'Excellent control. Your pronunciation stayed clear from start to finish.';
  }

  if (accuracy >= 80) {
    return 'Strong reading session. A few linking words need another pass.';
  }

  if (accuracy >= 60) {
    return 'Good effort. Slow down a little and focus on every short word.';
  }

  return 'Keep going. Replay this passage and aim for steady, confident pacing.';
}

export function scoreReadingSession(args: {
  passage: ReadingPassage;
  transcript: string;
  durationSec: number;
  sessionId: string;
  userId?: string | null;
}): ScorePayload {
  const { durationSec, passage, sessionId, transcript, userId } = args;
  const matches = compareWords(passage.text, transcript);
  const wordsCorrect = matches.filter((match) => match.status === 'correct').length;
  const totalWords = matches.length;
  const wordsIncorrect = totalWords - wordsCorrect;
  const accuracy = totalWords ? Number(((wordsCorrect / totalWords) * 100).toFixed(1)) : 0;
  const completionPercent = passage.wordCount
    ? Number(((Math.min(tokenize(transcript).length, passage.wordCount) / passage.wordCount) * 100).toFixed(1))
    : 0;
  const wordsPerMinute = durationSec > 0 ? Number(((tokenize(transcript).length / durationSec) * 60).toFixed(1)) : 0;
  const xpEarned = calculateXp(wordsCorrect, passage.difficulty);
  const feedback = buildFeedback(accuracy);

  const result: ReadingSessionResult = {
    id: sessionId,
    passageId: passage.id,
    expectedText: passage.text,
    transcript,
    accuracy,
    wordsCorrect,
    wordsIncorrect,
    completionPercent,
    durationSec,
    wordsPerMinute,
    xpEarned,
    createdAt: Date.now(),
    userId,
  };

  return {
    ...result,
    feedback,
    matches,
  };
}
