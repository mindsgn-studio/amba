import type { ReadingPassage } from '@/types/models';

const buildPassage = (
  passage: Omit<ReadingPassage, 'wordCount' | 'estimatedDurationSec'>,
): ReadingPassage => {
  const wordCount = passage.text.trim().split(/\s+/).length;
  return {
    ...passage,
    wordCount,
    estimatedDurationSec: Math.max(20, Math.round((wordCount / 90) * 60)),
  };
};

export const readingPassages: ReadingPassage[] = [
  buildPassage({
    id: 'daily-market',
    title: 'Morning at the Market',
    category: 'Daily Challenge',
    difficulty: 'beginner',
    text: 'I greet the fruit seller every morning before I buy bananas, oranges, and warm bread for breakfast.',
  }),
  buildPassage({
    id: 'bus-stop',
    title: 'Waiting for the Bus',
    category: 'Conversations',
    difficulty: 'beginner',
    text: 'Good afternoon. Is this the bus that goes to town, or should I wait for the next one near the blue sign?',
  }),
  buildPassage({
    id: 'study-circle',
    title: 'Study Circle',
    category: 'Intermediate',
    difficulty: 'intermediate',
    text: 'Our study group meets after class to practise speaking slowly, listen carefully, and help each other pronounce difficult words with confidence.',
  }),
  buildPassage({
    id: 'football-story',
    title: 'After the Match',
    category: 'Stories',
    difficulty: 'intermediate',
    text: 'After the match, the captain thanked the crowd, praised the team for staying calm, and promised they would train harder before the final.',
  }),
  buildPassage({
    id: 'future-dreams',
    title: 'Future Dreams',
    category: 'Advanced',
    difficulty: 'advanced',
    text: 'I want to become a confident speaker who can explain complex ideas clearly, encourage others kindly, and keep learning from every conversation.',
  }),
];

export const categories = Array.from(
  new Set(readingPassages.map((passage) => passage.category)),
);

export const getPassageById = (passageId: string) =>
  readingPassages.find((passage) => passage.id === passageId);
