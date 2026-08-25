import { drizzle, type BetterSQLite3Database } from "drizzle-orm/expo-sqlite";
import { type SQLiteDatabase, openDatabaseSync } from "expo-sqlite";
import { migrate } from "drizzle-orm/expo-sqlite/migrator";
import * as schema from "./schema";
import migrations from "./migrations/migrations";

export const DATABASE_NAME = "amba.db";

let _sqlite: SQLiteDatabase | null = null;
let _db: BetterSQLite3Database<typeof schema> | null = null;

function openDb(name: string): void {
  _sqlite = openDatabaseSync(name);
  _sqlite.execSync("PRAGMA foreign_keys = ON;");
  _db = drizzle(_sqlite, { schema });
}

/**
 * Open (or reopen) the SQLite database and run Drizzle migrations.
 * Safe to call multiple times — the previous connection is closed first.
 */
export async function initDatabase(): Promise<void> {
  closeDatabase();
  openDb(process.env.EXPO_PUBLIC_DB_NAME ?? DATABASE_NAME);
  await migrate(_db!, migrations);
}

/** Close the current database connection. Safe to call when not initialised. */
export function closeDatabase(): void {
  if (_sqlite) {
    try {
      _sqlite.closeSync();
    } catch {
      // already closed — ignore
    }
    _sqlite = null;
    _db = null;
  }
}

/** Return the active Drizzle database handle. Throws if `initDatabase` was not called. */
export function getDb(): BetterSQLite3Database<typeof schema> {
  if (!_db) {
    throw new Error("Database not initialised — call initDatabase() first");
  }
  return _db;
}

// Open eagerly so _layout.tsx can call useMigrations on the handle
// before any hook has called initDatabase().
openDb(process.env.EXPO_PUBLIC_DB_NAME ?? DATABASE_NAME);

/*
remove initDatabase, closeDatabase, and getDb from client.ts and from tests.  client.ts code should be

remove initDatabase, closeDatabase, and getDb from client.ts and from tests.  client.ts code should be import { drizzle } from "drizzle-orm/expo-sqlite";
import { openDatabaseSync } from "expo-sqlite";
import * as schema from "./schema";

export const DATABASE_NAME = "amafa.db";

export const expo_sqlite = openDatabaseSync(DATABASE_NAME);
// Ensure ON DELETE CASCADE on components works (off by default in SQLite).
expo_sqlite.execSync("PRAGMA foreign_keys = ON;");
export const db = drizzle(expo_sqlite, { schema });

 now rewrite tests and @app/src/db/repositories/, and client.ts
*/
