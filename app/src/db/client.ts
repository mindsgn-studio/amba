import { drizzle, type ExpoSQLiteDatabase } from 'drizzle-orm/expo-sqlite';
import { migrate } from 'drizzle-orm/expo-sqlite/migrator';
import * as SQLite from 'expo-sqlite';
import migrationBundle from './migrations/bundle';
import * as schema from './schema';

export type AppDatabase = ExpoSQLiteDatabase<typeof schema>;

/**
 * Database file name. Tests may override it to point at a temporary
 * file-backed database (EXPO_PUBLIC_DB_NAME); production always uses the
 * default application database.
 */
function databaseName(): string {
  return process.env.EXPO_PUBLIC_DB_NAME ?? 'readalong.db';
}

let db: AppDatabase | null = null;

/**
 * Opens the SQLite database and applies Drizzle migrations.
 *
 * The database is never reset during normal startup: books, jobs and
 * reading progress must survive app restarts and device restarts.
 */
export async function initDatabase(): Promise<AppDatabase> {
  if (db) {
    return db;
  }
  const sqlite = SQLite.openDatabaseSync(databaseName());
  const database = drizzle(sqlite, { schema });
  await migrate(database, migrationBundle);
  db = database;
  return database;
}

/** Returns the shared database instance; throws when not initialized. */
export function getDb(): AppDatabase {
  if (!db) {
    throw new Error('Database not initialized. Call initDatabase() first.');
  }
  return db;
}

/** Test-only helper to swap the database instance between tests. */
export function setDbForTesting(testDb: AppDatabase): void {
  db = testDb;
}

/** Closes the database if open (used by tests). */
export function closeDatabase(): void {
  db = null;
}
