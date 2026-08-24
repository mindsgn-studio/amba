# Read Along

An Expo (SDK 54) + TypeScript read-along audiobook app. Upload a book, wait
while the backend generates an audiobook, then listen while the exact word
being spoken is highlighted in sync with the audio. Your position is saved
continuously and restored on next launch.

Part of the `amba` repository:

- [`read-along/`](../read-along/) — Python/FastAPI backend that converts
  uploaded books into chapterized MP3 chunks plus per-chunk timing JSON.
- `app/` — this mobile app (the only part documented here).

## Architecture

State lives in three places, by lifetime:

| Layer | Store | Survives |
| --- | --- | --- |
| Session UI state | Zustand (`src/store/*`) | App restart: no |
| Persistent data | SQLite via Drizzle ORM (`src/db/*`) | App restart: yes |
| Media & jobs | FastAPI backend (`read-along/`) | Everything |

Key modules:

- `src/services/api/` — typed HTTP client for the backend contract
  (see [`../api.md`](../api.md)). Responses are validated with Zod before
  they touch local state.
- `src/db/` — Drizzle schema, generated migrations and repositories.
  The database stores exactly one book at a time (enforced at the
  repository boundary) plus its chapters/chunks, job snapshots for
  processing recovery and reading progress.
- `src/hooks/useReader.ts` — the playback orchestrator: loads the chunk
  playlist, fetches timing JSON, drives the expo-audio player, maps audio
  timestamps to word indices and persists progress (throttled to one write
  every 3 s, forced on pause/exit/chunk change/completion).
- `src/components/reader/ReaderText.tsx` — renders the chunk text as a
  single accessible `Text` node with a highlight around the current word;
  only re-renders when the word index changes, never on the 50 ms clock.

## Getting started

Requirements: Node 22+, npm, and either Expo Go or a dev build on a device
or emulator.

```bash
npm install

# Terminal 1 — start the backend API on :8000
cd ../read-along && <see read-along README for serving>

# Terminal 2 — start Metro
npm start
```

The app talks to `http://localhost:8000` by default. Override it at build
time with `EXPO_PUBLIC_API_URL`, e.g. for the Android emulator:

```bash
EXPO_PUBLIC_API_URL=http://10.0.2.2:8000 npm run android
```

(Or run `adb reverse tcp:8000 tcp:8000` so the emulator's `localhost`
reaches your machine.)

## Commands

```bash
npm run typecheck         # tsc --noEmit
npm run lint              # eslint (expo config)
npm test                  # all Jest tests
npm run test:unit         # unit + integration tests
npm run test:components   # React Native Testing Library component tests
npm run db:generate       # regenerate Drizzle migrations after schema edits
```

`db:generate` runs drizzle-kit and then `tools/build-migrations.mjs`, which
inlines the generated SQL into `src/db/migrations/bundle.ts`. The bundle (not
the raw `.sql` files) is what the app imports — Metro/Hermes cannot hand the
drizzle migrator usable text from a raw SQL asset import.

### Testing approach

Unit/integration tests run against **real SQLite** semantics: the
`expo-sqlite` module is mocked in `jest.setup.js` with a thin adapter over
Node's built-in `node:sqlite` engine, and Drizzle runs its real migrations.
Persistence across "app restarts" is covered by pointing
`EXPO_PUBLIC_DB_NAME` at a temporary file-backed database, closing it and
reopening it inside the test.

## End-to-end testing (Maestro)

A deterministic mock backend stands in for the Python service:

```bash
node tools/mock-backend.mjs          # serves http://localhost:8000
```

Behaviour: every book becomes 2 chapters × 3 chunks × 8 s of silence; jobs
complete ~12 s after creation; any title starting with `fail` fails ~3 s
after creation. See the header of `tools/mock-backend.mjs`.

Run the flows against an installed dev build (`com.readalong.app`):

```bash
maestro test .maestro
```

Flows, in intended order:

1. `01-home.yaml` — empty state on fresh install.
2. `02-upload.yaml` — picking `sample-book.txt` starts generation.
3. `03-processing.yaml` — polling completes and Home offers Start Reading.
4. `04-reader.yaml` — playback highlights words (`Currently reading: …`).
5. `05-resume.yaml` — position survives kill + relaunch.
6. `06-one-book-rule.yaml` — upload hidden while a book exists.
7. `07-error.yaml` — failed job shows error + Delete Book recovery.

Prerequisites per flow (also listed in each file): push the sample assets
and set up port forwarding first:

```bash
adb push tests/e2e-assets/sample-book.txt /sdcard/Download/
adb push tests/e2e-assets/fail-book.txt /sdcard/Download/
adb reverse tcp:8000 tcp:8000
```

## Project layout

```
app/
├── .maestro/            # E2E flows
├── app/                 # Routes (expo-router): index, processing, reader
├── src/
│   ├── components/      # ui/, books/, processing/, reader/
│   ├── config/env.ts    # EXPO_PUBLIC_* handling
│   ├── db/              # Drizzle client, schema, migrations, repositories
│   ├── hooks/           # useAppInitialization, useBook, useProcessingJob,
│   │                    # usePlayback, useReadingProgress, useReader
│   ├── schemas/         # Zod contracts mirroring ../api.md
│   ├── services/        # api/, audio/ (expo-audio), reader/
│   ├── store/           # Zustand stores
│   └── utils/           # urls, formatting, timing math
├── tests/
│   ├── components/      # Component tests (RNTL)
│   ├── e2e-assets/      # Files pushed to devices for Maestro flows
│   ├── fixtures/        # Backend response fixtures
│   ├── integration/     # Repository + reader-flow tests over real SQLite
│   ├── mocks/           # sqlite-mock.js, FakeAudioController, transformers
│   └── unit/            # Pure-function and service tests
└── tools/mock-backend.mjs
```
