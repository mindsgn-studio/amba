import { integer, real, sqliteTable, text } from 'drizzle-orm/sqlite-core';

export const settingsTable = sqliteTable('settings', {
  id: integer('id').primaryKey(),
  completedOnboarding: integer('completed_onboarding', { mode: 'boolean' })
    .notNull()
    .default(false),
  microphonePermission: text('microphone_permission').notNull().default('undetermined'),
  theme: text('theme').notNull().default('sunrise'),
});

export const readingSessionsTable = sqliteTable('reading_sessions', {
  id: text('id').primaryKey(),
  passageId: text('passage_id').notNull(),
  expectedText: text('expected_text').notNull(),
  transcript: text('transcript').notNull(),
  accuracy: real('accuracy').notNull(),
  wordsCorrect: integer('words_correct').notNull(),
  wordsIncorrect: integer('words_incorrect').notNull(),
  completionPercent: real('completion_percent').notNull(),
  durationSec: integer('duration_sec').notNull(),
  wordsPerMinute: real('words_per_minute').notNull(),
  xpEarned: integer('xp_earned').notNull(),
  createdAt: integer('created_at').notNull(),
  userId: text('user_id'),
});

export const schema = {
  settingsTable,
  readingSessionsTable,
};
