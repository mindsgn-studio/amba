# Read-Along Audiobook System — Backend Implementation

Local audiobook generation: text / TXT / PDF / EPUB → clean chunks → Kokoro TTS
→ MP3 + word-level timing JSON → served over HTTP for a read-along mobile player.

Architecture and API reference live in the repo root:
[README.md](../README.md) · [api.md](../api.md)

## Quick Start

```bash
./scripts/install.sh     # deps, venv, Kokoro model prefetch, DB init, health test
./scripts/test.sh        # full test suite
./scripts/start.sh       # start API + worker under PM2
```

Then:

```bash
curl -s http://localhost:8000/health

# Generate an audiobook from a string
curl -s -X POST http://localhost:8000/api/books \
  -H 'Content-Type: application/json' \
  -d '{"title":"Example","text":"The quick brown fox jumps over the lazy dog."}'

# Or upload a .txt / .pdf / .epub
curl -s -X POST http://localhost:8000/api/books/upload -F "file=@book.epub"
```

Poll `GET /api/jobs/{job_id}` until `completed`, then fetch chapter/chunk URLs
from `GET /api/books/{book_id}`:

```json
{
  "chunks": [
    {
      "audio_url":  "/api/files/books/abc/chapters/001/chunk_001.mp3",
      "timing_url": "/api/files/books/abc/chapters/001/chunk_001.json"
    }
  ]
}
```

## Running Without PM2

```bash
source .venv/bin/activate
uvicorn app.api.main:app --host 0.0.0.0 --port 8000   # terminal 1
python -m app.worker.main                              # terminal 2
```

## Configuration (`.env`, see `.env.example`)

| Variable | Default | Purpose |
| --- | --- | --- |
| `DATABASE_URL` | sqlite data/app.db | SQLite location |
| `DATA_DIR` | ./data | Generated books/audio/timings |
| `TTS_PROVIDER` | kokoro | `kokoro` \| `fake` (offline test engine) |
| `TTS_VOICE` | af_heart | af_bella, am_adam, bf_emma, ... |
| `AUDIO_FORMAT` | mp3 | mp3 needs FFmpeg; falls back to wav |
| `CHUNK_TARGET_WORDS` / `CHUNK_MAX_WORDS` | 800 / 1500 | Chunk sizing |
| `API_HOST` / `API_PORT` | 0.0.0.0 / 8000 | HTTP bind |
| `LOG_LEVEL` | INFO | Structured logs |

## Project Layout

```text
app/
├── api/
│   ├── main.py               # FastAPI app factory (/health, routers, CORS)
│   ├── routes/books.py       # POST string/upload, GET list/detail, DELETE
│   ├── routes/jobs.py        # GET job status + progress
│   ├── routes/files.py       # GET /api/files/{path} (traversal-guarded)
│   ├── models/schemas.py     # Pydantic request/response models
│   └── services/book_service.py
├── worker/
│   ├── main.py               # polling loop, SIGTERM-safe
│   ├── processor.py          # extract → normalize → chunk → TTS → timings
│   └── jobs.py               # claim/requeue helpers
├── documents/                # base Document model + pdf/epub/text parsers
├── processing/               # normalizer, sentence splitter, chunker
├── tts/                      # base provider, kokoro engine, fake engine
├── timing/words.py           # Word/TimingDocument, validation, lookup helper
├── database/                 # SQLAlchemy models, engine (WAL), repository
├── media/converter.py        # WAV → MP3 via FFmpeg
└── config.py                 # .env-driven settings
tests/                        # 59 tests + integration suite
scripts/                      # install / test / start / stop / deploy
data/                         # runtime output (gitignored)
ecosystem.config.js           # PM2: audiobook-api + audiobook-worker
```

## Pipeline Guarantees

- **Never one giant TTS call** — chapters split into ≤1500-word chunks on
  sentence boundaries.
- **Restart-safe** — startup requeues jobs stuck in `processing`; completed
  chunks are skipped, never regenerated.
- **Progress** = `completed_chunks / total_chunks * 100`, updated per chunk.
- **Timing ↔ audio integrity** — timings generated from the WAV before MP3
  conversion; conversion preserves duration so timings stay valid.
- **Displayed text == spoken text** — chunk text files are stored verbatim
  inside each timing JSON (`text` field).
- **Scanned PDFs rejected** with a clear error (no OCR in v1).

## Timing JSON Format

```json
{
  "chunk_id": "4c5d086969d8_ch001_s001",
  "text": "The wind began to blow through the trees.",
  "duration": 7.675,
  "words": [
    {"index": 0, "text": "The",   "start": 0.275, "end": 0.4},
    {"index": 1, "text": "wind",  "start": 0.4,   "end": 0.8}
  ]
}
```

Mobile playback loop (see api.md): keep a persistent word index, advance while
`word.end < audio_position`, highlight the current word, scale time by
playback speed client-side (`display_time = audio_time / speed`).

## Tests

```bash
./scripts/test.sh                    # unit + pipeline + e2e (fake TTS, offline)
.venv/bin/pytest -m integration      # adds real-Kokoro synthesis checks
```

Coverage includes: text/PDF/EPUB extraction (headers, footers, page numbers,
hyphenation), normalization, sentence splitting (abbreviations, decimals,
initials), chunking limits and sentence integrity, timing validation
(monotonicity, ordering), repository/job claiming/stale-job recovery, worker
pipeline, restart safety, API endpoints, path traversal guards, and the full
end-to-end flow: POST → job → worker → audio + timings served via HTTP.

## Deployment

```bash
./scripts/deploy.sh   # pull → install → TESTS MUST PASS → pm2 reload → health
```

Tests failing aborts the deploy without touching running processes.
