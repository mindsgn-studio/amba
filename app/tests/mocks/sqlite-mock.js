/**
 * expo-sqlite compatible database handle backed by Node's real SQLite
 * engine (`node:sqlite`). Used only in tests so repository code runs
 * against genuine SQLite semantics without native modules.
 */

const { DatabaseSync } = require('node:sqlite');

function createSqliteDatabase(name = ':memory:') {
  const db = new DatabaseSync(name);

  const statementFrom = (sql) => {
    const stmt = db.prepare(sql);
    return {
      executeSync(params = []) {
        const args = Array.isArray(params) ? params : [params];
        const info = stmt.run(...args);
        return {
          changes: Number(info.changes),
          lastInsertRowId: Number(info.lastInsertRowid ?? 0),
          getAllSync: () => stmt.all(...args),
          getFirstSync: () => stmt.get(...args),
        };
      },
      executeForRawResultSync(params = []) {
        const args = Array.isArray(params) ? params : [params];
        const rawStmt = db.prepare(sql);
        rawStmt.setReturnArrays(true);
        return {
          getAllSync: () => rawStmt.all(...args),
        };
      },
    };
  };

  return {
    databaseName: name,
    execSync(source) {
      db.exec(source);
    },
    prepareSync(sql) {
      return statementFrom(sql);
    },
    runSync(source, params = []) {
      const args = Array.isArray(params) ? params : [params];
      const info = db.prepare(source).run(...args);
      return {
        changes: Number(info.changes),
        lastInsertRowId: Number(info.lastInsertRowid ?? 0),
      };
    },
    getAllSync(source, params = []) {
      const args = Array.isArray(params) ? params : [params];
      return db.prepare(source).all(...args);
    },
    getFirstSync(source, params = []) {
      const args = Array.isArray(params) ? params : [params];
      return db.prepare(source).get(...args);
    },
    async getAllAsync(source, params = []) {
      return this.getAllSync(source, params);
    },
    async getFirstAsync(source, params = []) {
      return this.getFirstSync(source, params);
    },
    async runAsync(source, params = []) {
      return this.runSync(source, params);
    },
    async execAsync(source) {
      this.execSync(source);
    },
    closeSync() {
      db.close();
    },
    async closeAsync() {
      db.close();
    },
    isInTransactionSync() {
      return false;
    },
    withTransactionSync(fn) {
      db.exec('BEGIN');
      try {
        const result = fn();
        db.exec('COMMIT');
        return result;
      } catch (error) {
        db.exec('ROLLBACK');
        throw error;
      }
    },
  };
}

module.exports = { createSqliteDatabase };
