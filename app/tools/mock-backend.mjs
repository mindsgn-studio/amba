#!/usr/bin/env node
/**
 * Deterministic mock backend for Maestro E2E runs.
 *
 * Implements the subset of the read-along API contract documented in
 * ../api.md that the mobile app consumes:
 *
 *   POST   /api/books          (JSON body: {title, text})
 *   POST   /api/books/upload   (multipart: file + optional title)
 *   GET    /api/books
 *   GET    /api/books/:id
 *   DELETE /api/books/:id
 *   GET    /api/jobs/:jobId
 *   GET    /api/files/*        (mp3 audio + timing json)
 *   GET    /health
 *
 * Determinism rules:
 *   - Every generated audiobook has 2 chapters x 3 chunks x 8 seconds.
 *   - Jobs finish ~12s after creation (time-based, not poll-count based).
 *   - Any book whose title/file name starts with "fail" produces a job
 *     that fails ~3s after creation with a fixed error message.
 *
 * Usage: node tools/mock-backend.mjs [port]   (default 8000)
 */

import http from 'node:http';

const PORT = Number(process.argv[2] ?? process.env.PORT ?? 8000);

const CHUNK_SECONDS = 8;
const CHUNKS_PER_CHAPTER = 3;
const CHAPTERS = 2;
const JOB_TOTAL_MS = 12_000;
const JOB_FAIL_MS = 3_000;
const FAILURE_MESSAGE = 'Intentional E2E failure: this book always fails.';
const COMPLETION_MESSAGE =
  'This book finished generating during an automated end-to-end run.';

// ---------------------------------------------------------------------------
// Tiny silent MP3 synthesis (valid MPEG-1 Layer III frames, all-zero payload)
// ---------------------------------------------------------------------------

function makeSilentMp3(durationSeconds) {
  const header = Buffer.from([0xff, 0xfb, 0x90, 0xc0]); // 128kbps 44.1kHz mono
  const frameSize = 417;
  const frameCount = Math.ceil(
    (durationSeconds * 128_000) / (frameSize * 8),
  );
  const frame = Buffer.concat([header, Buffer.alloc(frameSize - header.length)]);
  return Buffer.concat(Array.from({ length: frameCount }, () => frame));
}

const AUDIO_CACHE = new Map();
function audioFor(chunkId) {
  if (!AUDIO_CACHE.has(chunkId)) {
    AUDIO_CACHE.set(chunkId, makeSilentMp3(CHUNK_SECONDS));
  }
  return AUDIO_CACHE.get(chunkId);
}

// ---------------------------------------------------------------------------
// Deterministic book/timing content
// ---------------------------------------------------------------------------

const SENTENCES = [
  'The quiet harbor waited for morning light',
  'Small boats rested along the wooden pier',
  'A gentle wind moved across the open water',
  'Reading along with every spoken word',
];

const WORDS_PER_CHUNK = 12;

function chunkText(bookIndex, chapterIndex, chunkIndex) {
  const sentence =
    SENTENCES[(bookIndex + chapterIndex + chunkIndex) % SENTENCES.length];
  const words = [];
  for (let i = 0; i < WORDS_PER_CHUNK; i += 1) {
    words.push(sentence.split(' ')[i % sentence.split(' ').length]);
  }
  return words.join(' ');
}

function timingFor(bookIndex, chunk) {
  const text = chunkText(bookIndex, chunk.chapterIndex, chunk.chunkIndex);
  const tokens = text.split(' ');
  const slot = CHUNK_SECONDS / tokens.length;
  const words = tokens.map((token, i) => ({
    word: token,
    start_time: Number((i * slot).toFixed(3)),
    end_time: Number(((i + 1) * slot).toFixed(3)),
  }));
  return {
    chunk_id: chunk.id,
    text,
    words,
  };
}

// ---------------------------------------------------------------------------
// In-memory state
// ---------------------------------------------------------------------------

let counter = 0;

const books = new Map(); // bookId -> {detail, createdAt, jobId}
const jobs = new Map(); // jobId -> {id, bookId, createdAt, failing}

function pad(n, width = 3) {
  return String(n).padStart(width, '0');
}

function createBook(title, sourceType) {
  counter += 1;
  const n = counter;
  const bookId = `book_${pad(n, 4)}`;
  const jobId = `job_${pad(n, 4)}`;
  const failing = /^fail/i.test(title);

  const chapters = [];
  for (let c = 0; c < CHAPTERS; c += 1) {
    const chunks = [];
    for (let k = 0; k < CHUNKS_PER_CHAPTER; k += 1) {
      const base = `/api/files/books/${bookId}/chapters/${pad(c + 1)}`;
      chunks.push({
        id: `${bookId}_c${c}_k${k}`,
        chunk_index: k,
        duration: CHUNK_SECONDS,
        audio_url: `${base}/chunk_${pad(k + 1)}.mp3`,
        timing_url: `${base}/chunk_${pad(k + 1)}.json`,
      });
    }
    chapters.push({
      id: `${bookId}_ch${c}`,
      chapter_index: c,
      title: `Chapter ${c + 1}`,
      chunks,
    });
  }

  books.set(bookId, {
    createdAt: Date.now(),
    jobId,
    detail: {
      id: bookId,
      title,
      language: 'en',
      source_type: sourceType,
      status: 'processing',
      chapters,
    },
  });

  jobs.set(jobId, {
    id: jobId,
    bookId,
    createdAt: Date.now(),
    failing,
  });

  return { book_id: bookId, job_id: jobId, status: 'queued' };
}

function jobSnapshot(job) {
  const elapsed = Date.now() - job.createdAt;

  if (!job.failing && elapsed >= JOB_TOTAL_MS) {
    return {
      id: job.id,
      book_id: job.bookId,
      status: 'completed',
      progress: 100,
    };
  }
  if (job.failing && elapsed >= JOB_FAIL_MS) {
    return {
      id: job.id,
      book_id: job.bookId,
      status: 'failed',
      progress: 35,
      error: FAILURE_MESSAGE,
    };
  }

  const total = job.failing ? JOB_FAIL_MS : JOB_TOTAL_MS;
  const progress = Math.max(1, Math.min(99, Math.floor((elapsed / total) * 100)));
  const status = elapsed < 400 ? 'queued' : 'processing';
  return { id: job.id, book_id: job.bookId, status, progress };
}

// ---------------------------------------------------------------------------
// HTTP plumbing
// ---------------------------------------------------------------------------

function sendJson(res, status, body) {
  const payload = JSON.stringify(body);
  res.writeHead(status, {
    'content-type': 'application/json',
    'content-length': Buffer.byteLength(payload),
  });
  res.end(payload);
}

async function readJsonBody(req) {
  const chunks = [];
  for await (const part of req) chunks.push(part);
  try {
    return JSON.parse(Buffer.concat(chunks).toString('utf8'));
  } catch {
    return {};
  }
}

/** Minimal multipart/form-data parser: extracts fields and file metadata. */
async function readMultipart(req) {
  const contentType = req.headers['content-type'] ?? '';
  const boundaryMatch = /boundary=(?:"([^"]+)"|([^;]+))/i.exec(contentType);
  if (!boundaryMatch) return {};
  const boundary = `--${boundaryMatch[1] ?? boundaryMatch[2]}`;

  const parts = [];
  for await (const part of req) parts.push(part);
  const body = Buffer.concat(parts);

  const fields = {};
  let fileName = null;

  for (const segment of body.toString('binary').split(boundary)) {
    const headerEnd = segment.indexOf('\r\n\r\n');
    if (headerEnd === -1) continue;
    const headers = segment.slice(0, headerEnd);
    const nameMatch = /name="([^"]*)"/i.exec(headers);
    if (!nameMatch) continue;
    const fileMatch = /filename="([^"]*)"/i.exec(headers);
    if (fileMatch) {
      fileName = fileMatch[1];
    } else {
      const value = segment.slice(headerEnd + 4).replace(/\r\n--\s*$/, '');
      fields[nameMatch[1]] = value.replace(/\r\n$/, '');
    }
  }

  return { fields, fileName };
}

function deriveTitle(fileName, fallbackTitle) {
  const base = (fileName ?? '').replace(/\.[a-z0-9]+$/i, '');
  const title = base || fallbackTitle || 'Untitled upload';
  // Strip characters that are unsafe in titles.
  return title.replace(/[\r\n]+/g, ' ').slice(0, 200);
}

const server = http.createServer(async (req, res) => {
  const url = new URL(req.url, `http://${req.headers.host}`);
  const path = url.pathname;

  try {
    if (req.method === 'GET' && path === '/health') {
      return sendJson(res, 200, { status: 'ok' });
    }

    if (req.method === 'POST' && path === '/api/books') {
      const body = await readJsonBody(req);
      const title = deriveTitle(null, body.title ?? 'Pasted text');
      return sendJson(res, 202, createBook(title, 'txt'));
    }

    if (req.method === 'POST' && path === '/api/books/upload') {
      const { fileName } = await readMultipart(req);
      const title = deriveTitle(fileName, 'Uploaded book');
      return sendJson(res, 202, createBook(title, 'txt'));
    }

    if (req.method === 'GET' && path === '/api/books') {
      return sendJson(res, 200, {
        books: [...books.values()].map((b) => ({
          id: b.detail.id,
          title: b.detail.title,
          status: b.detail.status,
        })),
      });
    }

    const bookMatch = /^\/api\/books\/([^/]+)$/.exec(path);
    if (bookMatch) {
      const entry = books.get(decodeURIComponent(bookMatch[1]));
      if (!entry) return sendJson(res, 404, { detail: 'Book not found' });

      if (req.method === 'GET') {
        const job = jobs.get(entry.jobId);
        const snapshot = jobSnapshot(job);
        entry.detail.status =
          snapshot.status === 'completed'
            ? 'completed'
            : snapshot.status === 'failed'
              ? 'failed'
              : 'processing';
        return sendJson(res, 200, {
          ...entry.detail,
          description: COMPLETION_MESSAGE,
        });
      }
      if (req.method === 'DELETE') {
        jobs.delete(entry.jobId);
        books.delete(entry.detail.id);
        return sendJson(res, 200, { deleted: true, id: entry.detail.id });
      }
    }

    const jobMatch = /^\/api\/jobs\/([^/]+)$/.exec(path);
    if (req.method === 'GET' && jobMatch) {
      const job = jobs.get(decodeURIComponent(jobMatch[1]));
      if (!job) return sendJson(res, 404, { detail: 'Job not found' });
      return sendJson(res, 200, jobSnapshot(job));
    }

    const timingMatch = /^\/api\/files\/books\/([^/]+)\/chapters\/(\d+)\/(chunk_\d+)\.json$/.exec(
      path,
    );
    if (req.method === 'GET' && timingMatch) {
      const [, , chapterIndexRaw, chunkNameRaw] = timingMatch;
      const chapterIndex = Number(chapterIndexRaw) - 1;
      const chunkIndex = Number(chunkNameRaw.replace('chunk_', '')) - 1;
      const chunk = {
        id: `${timingMatch[1]}_c${chapterIndex}_k${chunkIndex}`,
        chapterIndex,
        chunkIndex,
      };
      return sendJson(res, 200, timingFor(counter, chunk));
    }

    const audioMatch = /^\/api\/files\/.*\.mp3$/.exec(path);
    if (req.method === 'GET' && audioMatch) {
      const audio = audioFor(path);
      res.writeHead(200, {
        'content-type': 'audio/mpeg',
        'content-length': audio.length,
        'accept-ranges': 'none',
      });
      return res.end(audio);
    }

    return sendJson(res, 404, { detail: 'Not found' });
  } catch (error) {
    return sendJson(res, 500, {
      detail: error instanceof Error ? error.message : String(error),
    });
  }
});

server.listen(PORT, () => {
  console.log(`Mock read-along backend listening on http://localhost:${PORT}`);
  console.log('Books whose title starts with "fail" intentionally fail.');
});
