export type PassageDifficulty = 'beginner' | 'intermediate' | 'advanced';

export type ReadingCategory =
  | 'Beginner'
  | 'Intermediate'
  | 'Advanced'
  | 'Stories'
  | 'Conversations'
  | 'Daily Challenge';

export type MicrophonePermissionState =
  | 'undetermined'
  | 'granted'
  | 'denied'
  | 'blocked';

export type SessionControllerState =
  | 'idle'
  | 'preparing'
  | 'recording'
  | 'processing'
  | 'scored'
  | 'error';

export type AppTheme = 'sunrise';

export type ReadingPassage = {
  id: string;
  title: string;
  category: ReadingCategory;
  difficulty: PassageDifficulty;
  text: string;
  wordCount: number;
  estimatedDurationSec: number;
};

export type ReadingSessionResult = {
  id: string;
  passageId: string;
  expectedText: string;
  transcript: string;
  accuracy: number;
  wordsCorrect: number;
  wordsIncorrect: number;
  completionPercent: number;
  durationSec: number;
  wordsPerMinute: number;
  xpEarned: number;
  createdAt: number;
  userId?: string | null;
};

export type Settings = {
  completedOnboarding: boolean;
  microphonePermission: MicrophonePermissionState;
  theme: AppTheme;
};

export type UserProfile = {
  uid: string;
  isAnonymous: boolean;
  createdAt: number;
  mode: 'firebase' | 'local';
};

export type HomeStats = {
  totalXp: number;
  averageAccuracy: number;
  totalWordsRead: number;
  sessionsCompleted: number;
};

export type WordMatch = {
  expected: string;
  spoken: string | null;
  status: 'correct' | 'incorrect' | 'missing';
};

export type ScorePayload = ReadingSessionResult & {
  feedback: string;
  matches: WordMatch[];
};
