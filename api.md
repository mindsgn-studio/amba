# Read-Along Audiobook System — API Reference

The API is intentionally small. It manages books, asynchronous processing jobs, and serving of generated files (audio + word timings). All expensive processing happens in the worker — never inside an HTTP request.

Base URL: `http://localhost:8000`

All responses are `application/json` unless the endpoint serves a file.

## Endpoints

```text
POST   /api/books            create audiobook from raw text string
POST   /api/books/upload     create audiobook from .pdf / .epub / .txt file
GET    /api/books            list all books
GET    /api/books/{book_id}  book metadata with chapters and chunks
DELETE /api/books/{book_id}  delete a book
GET    /api/jobs/{job_id}    job status and progress
GET    /api/files/{path}     serve generated audio/timing files
GET    /health               health check
```

---

## POST /api/books

Create an audiobook from a raw text string.

### Request

```http
POST /api/books
Content-Type: application/json
```

```json
{
  "title": "Example",
  "text": "The quick brown fox...",
  "language": "en",
  "voice": "af_heart"
}
```

| Field | Type | Required | Notes |
| --- | --- | --- | --- |
| `title` | string | yes | Display title |
| `text` | string | yes | Raw text; bypasses document extraction entirely |
| `language` | string | no | BCP-47 style code, e.g. `en`. Defaults from config |
| `voice` | string | no | TTS voice, e.g. `af_heart`. Defaults from config |

### Response

```http
202 Accepted
```

```json
{
  "book_id": "abc123",
  "job_id": "job123",
  "status": "queued"
}
```

The audio is **not** generated in this request. A book record and a queued job are created; the worker picks the job up.

---

## POST /api/books/upload

Create an audiobook from an uploaded file.

### Request

```http
POST /api/books/upload
Content-Type: multipart/form-data
```

| Field | Type | Required | Notes |
| --- | --- | --- | --- |
| `file` | binary | yes | One of `.pdf`, `.epub`, `.txt` |
| `title` | string | no | Falls back to filename or document metadata |
| `language` | string | no | e.g. `en` |
| `voice` | string | no | e.g. `af_heart` |

```bash
curl -X POST http://localhost:8000/api/books/upload \
  -F "file=@test-book.epub" \
  -F "title=Test Book"
```

### Behavior

1. Saves the uploaded file to `data/books/{book_id}/source/`
2. Creates the book record
3. Creates a queued job
4. Returns immediately

### Response

```json
{
  "book_id": "abc123",
  "job_id": "job123",
  "status": "queued"
}
```

### Errors

- `415 Unsupported Media Type` — file type not in `.pdf`, `.epub`, `.txt`
- `422 Unprocessable Entity` — invalid form payload

Note: scanned PDFs (too little extractable text) fail during processing, not upload. The job will be marked `failed`.

---

## GET /api/books

List all books.

### Response

```json
{
  "books": [
    {
      "id": "abc123",
      "title": "Example Book",
      "author": "Author Name",
      "language": "en",
      "source_type": "epub",
      "status": "completed",
      "created_at": "2026-08-23T20:15:01Z",
      "updated_at": "2026-08-23T20:31:44Z"
    }
  ]
}
```

Book `status`: `queued | processing | completed | failed`.

---

## GET /api/books/{book_id}

Full book metadata including chapters, chunks, and playable URLs.

### Response

```json
{
  "id": "abc123",
  "title": "Example Book",
  "status": "completed",
  "chapters": [
    {
      "id": "chapter1",
      "chapter_index": 0,
      "title": "Chapter 1",
      "chunks": [
        {
          "id": "chunk1",
          "chunk_index": 0,
          "duration": 23.4,
          "audio_url": "/api/files/books/abc123/chapters/001/chunk_001.mp3",
          "timing_url": "/api/files/books/abc123/chapters/001/chunk_001.json"
        }
      ]
    }
  ]
}
```

Chunk fields:

| Field | Description |
| --- | --- |
| `audio_url` | MP3 for this chunk |
| `timing_url` | Word-timing JSON for this chunk |
| `duration` | Audio duration in seconds |

Errors: `404 Not Found` if the book does not exist.

---

## DELETE /api/books/{book_id}

Delete a book: removes its database records (book, chapters, chunks) and generated files under `data/books/{book_id}/`.

### Response

```json
{ "deleted": true, "id": "abc123" }
```

Errors: `404 Not Found` if the book does not exist.

---

## GET /api/jobs/{job_id}

Poll job status. The mobile app can display generation progress from this.

### Response

```json
{
  "id": "job123",
  "type": "generate_book",
  "book_id": "abc123",
  "status": "processing",
  "progress": 64,
  "error": null,
  "created_at": "2026-08-23T20:15:01Z",
  "started_at": "2026-08-23T20:15:02Z",
  "completed_at": null
}
```

| Field | Description |
| --- | --- |
| `status` | `queued | processing | completed | failed` |
| `progress` | `0–100`, computed as `completed_chunks / total_chunks * 100` |
| `error` | Human-readable error message when `status = failed` |

Errors: `404 Not Found` if the job does not exist.

---

## GET /api/files/{path}

Serve generated files from the data directory.

```http
GET /api/files/books/abc123/chapters/001/chunk_001.mp3   → audio/mpeg
GET /api/files/books/abc123/chapters/001/chunk_001.json  → application/json
```

Paths are exactly those returned by `GET /api/books/{book_id}` (`audio_url`, `timing_url`). The MVP serves files directly from the API; if traffic grows, move files to object storage/CDN later.

Path traversal outside `DATA_DIR` is rejected with `403 Forbidden`. Missing files return `404 Not Found`.

---

## GET /health

Liveness check used by `deploy.sh` after restarts.

### Response

```json
{
  "status": "ok",
  "version": "0.1.0"
}
```

---

## Timing File Format

Each chunk has a timing JSON bound to its exact source text:

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

Guarantees:

- `words` are ordered by time and by index
- `start < end` for every word
- timestamps are monotonic across words within a chunk
- `text` is exactly what was synthesized — render it as-is so displayed and spoken content match

## Mobile Synchronization Guide

The app maintains three values per chapter: `currentChunk`, `currentTime` (audio position), `currentWord`.

### Word lookup

Do not scan every word on every tick (~50 ms). Keep a persistent `currentWordIndex` and advance forward:

```python
while current_word.end < current_time:
    current_word_index += 1
```

This is extremely cheap. On seek backward, reset the index to 0 or binary search.

Highlight the word where `word.start <= current_time < word.end`. Example at position `32.41s` against words ending `30.4 / 31.2 / 32.8 / 33.4` → word 103 (`31.3–32.8`) is highlighted.

When crossing into a new chunk, load that chunk's timing JSON and reset the local word index.

### Playback speed

Timing JSON always corresponds to normal speed (`1.0x`). For other speeds, scale display time only — never regenerate audio/timings:

```text
display_time = audio_time / speed
```

e.g. at `2x`, a word timed `0.55–0.87` highlights during wall-clock `0.275–0.435`.

### Auto-scroll

Scroll only when the highlighted word approaches the bottom of the comfortable reading area, using smooth scrolling. Do not scroll for every word.

## Authentication

For local/private deployments the API may run without auth. Before exposing publicly, protect it with an API key or JWT — especially `POST /api/books/upload`.
