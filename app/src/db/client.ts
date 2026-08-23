import { drizzle } from 'drizzle-orm/expo-sqlite';
import { eq, sql } from 'drizzle-orm';
import { openDatabaseSync } from 'expo-sqlite';

import { readingSessionsTable, schema, settingsTable } from '@/db/schema';
import type { HomeStats, ReadingSessionResult, Settings } from '@/types/models';

const sqlite = openDatabaseSync('amba.db');

export const db = drizzle(sqlite, { schema });

let initialized = false;

const defaultSettings: Settings = {
  completedOnboarding: false,
  microphonePermission: 'undetermined',
  theme: 'sunrise',
};

export async function initializeDatabase() {
  if (initialized) {
    return;
  }

  await sqlite.execAsync(`
    PRAGMA journal_mode = WAL;
    CREATE TABLE IF NOT EXISTS settings (
      id INTEGER PRIMARY KEY NOT NULL,
      completed_onboarding INTEGER NOT NULL DEFAULT 0,
      microphone_permission TEXT NOT NULL DEFAULT 'undetermined',
      theme TEXT NOT NULL DEFAULT 'sunrise'
    );
    CREATE TABLE IF NOT EXISTS reading_sessions (
      id TEXT PRIMARY KEY NOT NULL,
      passage_id TEXT NOT NULL,
      expected_text TEXT NOT NULL,
      transcript TEXT NOT NULL,
      accuracy REAL NOT NULL,
      words_correct INTEGER NOT NULL,
      words_incorrect INTEGER NOT NULL,
      completion_percent REAL NOT NULL,
      duration_sec INTEGER NOT NULL,
      words_per_minute REAL NOT NULL,
      xp_earned INTEGER NOT NULL,
      created_at INTEGER NOT NULL,
      user_id TEXT
    );
    INSERT OR IGNORE INTO settings (
      id,
      completed_onboarding,
      microphone_permission,
      theme
    ) VALUES (1, 0, 'undetermined', 'sunrise');
  `);

  initialized = true;
}

export async function getSettings(): Promise<Settings> {
  await initializeDatabase();

  const record = await db.query.settingsTable.findFirst({
    where: eq(settingsTable.id, 1),
  });

  if (!record) {
    return defaultSettings;
  }

  return {
    completedOnboarding: record.completedOnboarding,
    microphonePermission: record.microphonePermission as Settings['microphonePermission'],
    theme: record.theme as Settings['theme'],
  };
}

export async function updateSettings(
  patch: Partial<Settings>,
): Promise<Settings> {
  await initializeDatabase();
  const current = await getSettings();

  const next: Settings = {
    ...current,
    ...patch,
  };

  await db
    .update(settingsTable)
    .set({
      completedOnboarding: next.completedOnboarding,
      microphonePermission: next.microphonePermission,
      theme: next.theme,
    })
    .where(eq(settingsTable.id, 1));

  return next;
}

export async function saveReadingSession(session: ReadingSessionResult) {
  await initializeDatabase();

  await db.insert(readingSessionsTable).values({
    id: session.id,
    passageId: session.passageId,
    expectedText: session.expectedText,
    transcript: session.transcript,
    accuracy: session.accuracy,
    wordsCorrect: session.wordsCorrect,
    wordsIncorrect: session.wordsIncorrect,
    completionPercent: session.completionPercent,
    durationSec: session.durationSec,
    wordsPerMinute: session.wordsPerMinute,
    xpEarned: session.xpEarned,
    createdAt: session.createdAt,
    userId: session.userId ?? null,
  });
}

export async function getReadingSessionById(sessionId: string) {
  await initializeDatabase();

  const record = await db.query.readingSessionsTable.findFirst({
    where: eq(readingSessionsTable.id, sessionId),
  });

  return record ?? null;
}

export async function listRecentSessions(limit = 3): Promise<ReadingSessionResult[]> {
  await initializeDatabase();

  const rows = await db.query.readingSessionsTable.findMany({
    orderBy: (sessions, { desc }) => desc(sessions.createdAt),
    limit,
  });

  return rows;
}

export async function getHomeStats(): Promise<HomeStats> {
  await initializeDatabase();

  const totals = await db
    .select({
      totalXp: sql<number>`coalesce(sum(${readingSessionsTable.xpEarned}), 0)`,
      averageAccuracy: sql<number>`coalesce(avg(${readingSessionsTable.accuracy}), 0)`,
      totalWordsRead: sql<number>`coalesce(sum(${readingSessionsTable.wordsCorrect}), 0)`,
      sessionsCompleted: sql<number>`count(${readingSessionsTable.id})`,
    })
    .from(readingSessionsTable);

  const row = totals[0];

  return {
    totalXp: row?.totalXp ?? 0,
    averageAccuracy: Number(row?.averageAccuracy ?? 0),
    totalWordsRead: row?.totalWordsRead ?? 0,
    sessionsCompleted: row?.sessionsCompleted ?? 0,
  };
}
