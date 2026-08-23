# Read-Along Audiobook System

A local audiobook-generation system that converts raw text strings, `.txt`, `.pdf`, and `.epub` inputs into:

1. Clean, structured text
2. Locally generated audiobook audio
3. Word-level timing information
4. Chapter/section metadata
5. A format a mobile application can use to highlight words while audio plays

The final user experience is similar to reading lyrics: as the audio progresses, the highlighted word changes and the reader auto-scrolls.

```text
The quick brown fox jumps over the lazy dog.
          ^^^^^
       current word
```

## Core Principle

**Document → normalized text → chunks → TTS → audio + word timings → API → read-along player.**

Everything else is built around that pipeline.

## Architecture

```text
                    Mobile App
                        │
                        │ HTTP
                        ▼
                ┌─────────────────┐
                │   Python API    │
                │    FastAPI      │
                └────────┬────────┘
                         │
                         ▼
                ┌─────────────────┐
                │     SQLite      │
                │ books/chapters  │
                │  chunks/jobs    │
                └────────┬────────┘
                         │
                         ▼
                ┌─────────────────┐
                │ Python Worker   │
                │ Extract/Clean   │
                │ Chunk/TTS/Timing│
                └────────┬────────┘
              ┌──────────┴──────────┐
              ▼                     ▼
        Audio files             Timing JSON
        .mp3                    word timings
```

Two processes managed by PM2:

```text
PM2
 ├── audiobook-api      (uvicorn app.api.main:app)
 └── audiobook-worker   (python -m app.worker.main)
```

## Technology Stack

| Layer | Technology |
| --- | --- |
| Backend | Python 3.11+, FastAPI, Uvicorn, SQLite, SQLAlchemy/SQLModel, Pydantic |
| PDF parsing | PyMuPDF |
| EPUB parsing | ebooklib + BeautifulSoup |
| TTS | Kokoro-82M (local, Apache-2.0) |
| Audio conversion | FFmpeg (WAV → MP3) |
| Process management | PM2 |

### Why Kokoro?

Small model, fast inference, good narration quality, runs locally on CPU (GPU-accelerated), multiple voices, and existing support for word-level timing. The system needs *stable, clear narration + reliable timing* — not arbitrary conversational voice generation — which makes Kokoro an excellent fit.

## Key Design Decisions

### Never generate an entire book in one TTS request

Books are split into chunks of **500–1,500 words**, aligned to sentence boundaries:

```text
Book
 ├── Chapter 1
 │    ├── Chunk 1
 │    ├── Chunk 2
 │    └── Chunk 3
 ├── Chapter 2
 │    └── ...
```

This gives smaller TTS requests, easier retries, lower memory usage, partial availability (users can start reading before completion), and independent regeneration of failed chunks.

### Word-level timing is the core feature

Each chunk produces timing data bound to its exact source text:

```json
{
  "chunk_id": "chunk_001",
  "text": "The quick brown fox jumps over the lazy dog.",
  "words": [
    { "index": 0, "text": "The",   "start": 0.00, "end": 0.21 },
    { "index": 1, "text": "quick", "start": 0.22, "end": 0.54 },
    { "index": 2, "text": "brown", "start": 0.55, "end": 0.87 }
  ]
}
```

The mobile app never performs speech recognition. It asks: given `audio_position = 0.60s`, find the word where `start <= t < end`.

### Timing and audio must stay together

Audio is never modified after generating timings unless timings are regenerated too. Playback speed changes are handled client-side by scaling time: `display_time = audio_time / speed`.

### TTS is abstracted behind an interface

```python
class TTSProvider:
    def synthesize(self, text: str, voice: str): ...
```

Kokoro is only one implementation (`TTS_PROVIDER=kokoro`). Piper, Chatterbox, or cloud providers can be added without touching the audiobook pipeline.

### All input formats converge on one document model

```python
Document(
    title="Example Book",
    author="Author",
    language="en",
    chapters=[
        Chapter(title="Chapter 1", paragraphs=["...", "..."])
    ]
)
```

PDF, EPUB, TXT, and raw strings all normalize into this model before chunking/TTS.

### TTS is decoupled from the API

The API never generates audio inside an HTTP request. It creates a book record plus a queued job and returns immediately; the worker does the expensive processing. This keeps the API responsive.

### The mobile app is decoupled from Kokoro

The app only knows about `audio_url`, `timing_url`, and `text`. The TTS engine can be replaced without any client changes.

## Document Processing Pipeline

```text
                 Input
                   │
       ┌───────────┼───────────┐
       ▼           ▼           ▼
      PDF         EPUB        TEXT
       │           │           │
   PDF Parser  EPUB Parser     │
       └───────────┴───────────┘
                   │
             Normalization
                   │
             Document Model
                   │
            Sentence splitting
                   │
               Chunking
```

### PDF (PyMuPDF)

Handles page breaks, repeated headers/footers, page numbers, excessive whitespace, hyphenated line breaks, and empty pages. Scanned PDFs (insufficient extracted text) are marked unsupported — no OCR in v1.

### EPUB (ebooklib + BeautifulSoup)

Extracts HTML/XHTML into the DOM, preserving headings, paragraphs, lists, quotes, and blockquotes where useful; ignores scripts, styles, navigation markup, and embedded ads.

### Normalization

Whitespace, newlines, Unicode, smart quotes, em dashes, repeated punctuation, broken words, page artifacts, HTML artifacts. Normalization is deliberately conservative — displayed text must match spoken text exactly.

### Chunking

Chunk boundaries respect semantic order: **Chapter → Paragraph → Sentence → Chunk**. Sentences are split first so chunks never cut sentences in half.

## Storage

SQLite stores metadata only. Filesystem stores binaries:

```text
data/
 ├── app.db
 └── books/
      └── book_id/
           ├── source/
           ├── chapters/
           │    ├── 001/
           │    │    ├── chunk_001.mp3
           │    │    ├── chunk_001.json
           │    │    └── chunk_002.mp3
           │    └── 002/
           └── metadata.json
```

Database schema (simplified):

- `books(id, title, author, language, source_type, source_path, status, created_at, updated_at)`
- `chapters(id, book_id, chapter_index, title, status)`
- `chunks(id, chapter_id, chunk_index, text_path, audio_path, timing_path, duration, status)`
- `jobs(id, type, book_id, status, progress, error, created_at, started_at, completed_at)`

Statuses: `queued → processing → completed | failed`.

## Worker

Polls SQLite for jobs, processes them, and is restart-safe:

```text
Job
 → Load book
 → Extract document
 → Normalize
 → Create chapters
 → Create chunks
 → For each chunk:
      Generate TTS + timestamps
      Save audio, save timing, update DB
 → Mark book completed
```

Completed chunks are skipped on restart — work is never regenerated unnecessarily. Progress = `completed_chunks / total_chunks * 100`.

Audio pipeline per chunk: `Kokoro → WAV → FFmpeg → MP3` (timing generated before conversion; conversion is lossless w.r.t. timing).

## Configuration

All paths/settings come from `.env` (see `.env.example`) — nothing hard-coded:

```env
APP_ENV=production

DATABASE_URL=sqlite:///./data/app.db

DATA_DIR=./data

TTS_PROVIDER=kokoro
TTS_VOICE=af_heart

API_HOST=0.0.0.0
API_PORT=8000

LOG_LEVEL=INFO
```

For local/private deployments authentication can be minimal. If exposed publicly, add API key or JWT auth before exposing — especially `POST /api/books/upload`.

## Getting Started

```bash
git clone <repository>
cd read-along

./scripts/install.sh    # checks deps, venv, Kokoro models, data dirs, DB init
./scripts/test.sh       # full test suite
./scripts/deploy.sh     # git pull → install → test → pm2 restart → health check
```

Process management:

```bash
pm2 start ecosystem.config.js
pm2 restart ecosystem.config.js
pm2 stop ecosystem.config.js
```

After deployment:

- API: `http://server:8000`
- Health check: `GET /health`

`deploy.sh` will **not** restart production if tests fail.

## API Summary

Full reference: [api.md](api.md)

```text
POST   /api/books            create audiobook from raw string
POST   /api/books/upload     create audiobook from .pdf/.epub/.txt file
GET    /api/books            list books
GET    /api/books/{id}       book metadata, chapters, chunks, URLs
DELETE /api/books/{id}       delete a book
GET    /api/jobs/{id}        job status + progress
GET    /api/files/{path}     serve generated audio/timing files
GET    /health               health check
```

## Project Structure

```text
read-along/
├── app/
│   ├── api/                  # FastAPI app, routes, models, services
│   │   ├── main.py
│   │   ├── routes/           # books.py, jobs.py, files.py
│   │   ├── models/
│   │   └── services/
│   ├── worker/               # job loop and processor
│   │   ├── main.py
│   │   ├── processor.py
│   │   └── jobs.py
│   ├── documents/            # parsers → Document Model
│   │   ├── base.py
│   │   ├── pdf.py
│   │   ├── epub.py
│   │   └── text.py
│   ├── processing/           # normalizer, sentence_splitter, chunker
│   ├── tts/                  # base.py (interface), kokoro.py
│   ├── timing/               # words.py
│   ├── database/             # models, database, repository
│   └── config.py
├── tests/                    # unit + integration + e2e
├── scripts/                  # install.sh, test.sh, start.sh, stop.sh, deploy.sh
├── data/                     # SQLite + generated audio/timings
├── .env.example
├── requirements.txt
├── ecosystem.config.js
└── README.md
```

## Logging

Structured worker logs, e.g.:

```text
2026-08-23 20:15:01 INFO Job started job=abc123
2026-08-23 20:15:03 INFO Chapters found count=12
2026-08-23 20:15:04 INFO Creating chunks count=84
2026-08-23 20:15:15 INFO Chunk complete duration=23.4
ERROR TTS generation failed job=abc123 chunk=17 error="..."
```

## Testing

Each stage is tested independently: text/PDF/EPUB extraction, normalization (line breaks, spaces, hyphenation, Unicode, quotes, punctuation), chunking (short/long text, paragraph/sentence boundaries), TTS (audio exists), timing (start < end, monotonic ordering).

An integration test runs a small `test-book.epub` through the entire chain (API → SQLite → worker → Kokoro → audio → timings → completed) verifying: status completed, files exist, word count > 0, duration > 0, timestamps monotonic. The end-to-end test proves `POST /api/books → job → worker → GET /api/books/{id}` with playable audio and valid timings.

See `scripts/test.sh`.

## Roadmap

| Milestone | Scope |
| --- | --- |
| MVP 1 | String → Kokoro → audio + word timestamps → React Native read-along UI |
| MVP 2 | TXT / PDF / EPUB via unified document model |
| MVP 3 | SQLite, jobs, worker, progress |
| MVP 4 | PM2, install.sh, test.sh, deploy.sh — deployable backend |
| MVP 5 | Mobile UX: highlighting, auto-scroll, play/pause, seek, speed, chapter nav, resume |

Later: voice selection (`af_heart`, `af_bella`, `am_adam`, `bf_emma`, ...), playback speeds (0.75x–2x), sentence/karaoke highlight modes, translation pipeline, multi-language support (treated as configurable capability, not assumed).

## Development Order

```text
Kokoro experiment → word timestamps → minimal player → string→audiobook
→ normalization → chunking → PDF → EPUB → SQLite → worker → FastAPI
→ jobs/progress → PM2 → install/test/deploy scripts → mobile integration
→ auto-scroll/resume/speed
```

The first technical milestone is **not** PDF support. It is:
**Kokoro → audio + word timestamps → React Native read-along UI.**

If that works reliably, the rest of the backend becomes straightforward document processing and job management.

## Definition of Done (v1)

- [ ] Submit a text string through the API
- [ ] Upload a PDF
- [ ] Upload an EPUB
- [ ] Store book metadata in SQLite
- [ ] Create asynchronous processing jobs
- [ ] Worker picks up jobs
- [ ] Extract document text, detect chapters, normalize, chunk
- [ ] Generate local speech (Kokoro) with word-level timestamps
- [ ] Store audio files and timing JSON
- [ ] Track processing progress
- [ ] Resume failed/interrupted jobs
- [ ] Retrieve completed audiobook metadata
- [ ] Serve audio + timing files through the API
- [ ] Run API and worker under PM2
- [ ] Install via `install.sh`, test via `test.sh`, deploy via `deploy.sh`
- [ ] Mobile app plays audio, highlights the correct word, auto-scrolls
