import { render, screen } from '@testing-library/react-native';

import { ScoreSummary } from '@/components/score-summary';
import type { ScorePayload } from '@/types/models';

const score: ScorePayload = {
  id: 'session-1',
  passageId: 'daily-market',
  expectedText: 'hello world',
  transcript: 'hello world',
  accuracy: 92,
  wordsCorrect: 11,
  wordsIncorrect: 1,
  completionPercent: 100,
  durationSec: 12,
  wordsPerMinute: 55,
  xpEarned: 11,
  createdAt: 1,
  feedback: 'Strong reading session.',
  matches: [],
};

describe('ScoreSummary', () => {
  it('renders the hero score and metrics', () => {
    render(<ScoreSummary score={score} />);

    expect(screen.getByText('92% Accuracy')).toBeTruthy();
    expect(screen.getByText('Correct words')).toBeTruthy();
    expect(screen.getByText('+11')).toBeTruthy();
  });
});
