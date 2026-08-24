/**
 * Global Jest setup: reusable mocks for native modules.
 *
 * - expo-sqlite is backed by Node's real SQLite engine (node:sqlite), so
 *   repository tests run against genuine SQLite semantics in isolation.
 * - expo-audio / expo-speech / expo-document-picker are replaced with
 *   deterministic fakes.
 */

jest.mock('expo-sqlite', () => {
  const { createSqliteDatabase } = require('./tests/mocks/sqlite-mock');
  return {
    // Tests use :memory: by default; a test may point EXPO_PUBLIC_DB_NAME
    // at a temporary file to exercise true persistence across "restarts".
    openDatabaseSync: jest.fn(() =>
      createSqliteDatabase(process.env.EXPO_PUBLIC_DB_NAME ?? ':memory:'),
    ),
    openDatabaseAsync: jest.fn(async () =>
      createSqliteDatabase(process.env.EXPO_PUBLIC_DB_NAME ?? ':memory:'),
    ),
    SQLiteProvider: ({ children }) => children,
  };
});

// ---------------------------------------------------------------------------
// expo-audio — fake player with inspectable calls and manual status events
// ---------------------------------------------------------------------------

const fakePlayers = [];

jest.mock('expo-audio', () => {
  return {
    createAudioPlayer: jest.fn(() => {
      const listeners = new Set();
      const player = {
        uri: null,
        currentTime: 0,
        duration: 0,
        playing: false,
        muted: false,
        loop: false,
        isLoaded: true,
        isBuffering: false,
        shouldCorrectPitch: false,
        playbackRate: 1,
        id: Math.floor(Math.random() * 1e6),
        play: jest.fn(() => {
          player.playing = true;
        }),
        pause: jest.fn(() => {
          player.playing = false;
        }),
        replace: jest.fn(),
        seekTo: jest.fn(async (seconds) => {
          player.currentTime = seconds;
        }),
        setPlaybackRate: jest.fn((rate) => {
          player.playbackRate = rate;
        }),
        remove: jest.fn(),
        addListener: jest.fn((_event, callback) => {
          listeners.add(callback);
          return { remove: () => listeners.delete(callback) };
        }),
        __emitStatus(status) {
          listeners.forEach((listener) => listener(status));
        },
      };
      fakePlayers.push(player);
      return player;
    }),
    setAudioModeAsync: jest.fn(async () => undefined),
    useAudioPlayer: jest.fn(),
    __getFakePlayers: () => fakePlayers,
  };
});

// ---------------------------------------------------------------------------
// expo-speech — kept available for accessibility/fallback use cases
// ---------------------------------------------------------------------------

jest.mock('expo-speech', () => ({
  speak: jest.fn(),
  stop: jest.fn(),
  isSpeakingAsync: jest.fn(async () => false),
  getAvailableVoicesAsync: jest.fn(async () => []),
}));

// ---------------------------------------------------------------------------
// expo-document-picker
// ---------------------------------------------------------------------------

jest.mock('expo-document-picker', () => ({
  getDocumentAsync: jest.fn(async () => ({ canceled: true, assets: [] })),
}));

// ---------------------------------------------------------------------------
// fetch — default no-op; tests override per case
// ---------------------------------------------------------------------------

global.fetch = jest.fn(async () => ({
  ok: false,
  status: 500,
  text: async () => JSON.stringify({ detail: 'fetch not mocked' }),
}));
