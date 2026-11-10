# Amba

Amba is a desktop read-along player. It plays an audiobook and shows the words
on screen, highlighting the word being narrated. When an EPUB sits beside the
audio, the reader shows the book's own text and follows the narration through
it; otherwise it shows the narration's transcript.

It is built with [mygo](https://github.com/egoist/mygo) and Go.

## The three screens

1. **Loading** — while the app finds the audiobooks on disk.
2. **Library** — the books it found, with covers, authors and how far you got
   in each one. Selecting a book opens it.
3. **Read along** — the book's text, the word currently being read, playback
   controls, a progress bar, chapter navigation, playback speed and a
   *Follow* switch that keeps the active paragraph in view. When an EPUB is
   attached, **Show ebook** opens a drawer beside the reader with the printed
   book, page by page.

Progress (position, duration and speed) is saved per book and restored the
next time the book is opened.

## The ebook drawer

The drawer is for checking how much of the printed book the narration actually
reads. Each word is coloured by its place in the audiobook:

- **green** — the word is matched to the narration and the reader highlights it;
- **red** — the word has no reliable match;
- punctuation-only tokens stay muted, because they carry no narration.

Pages turn with **Previous page** and **Next page**, and the drawer opens at
the chapter being heard. A rejected chapter is still shown for inspection, but
its words are red because the reader falls back to the transcript there.

**Remove EPUB** detaches the book without touching the audiobook, the player or
your progress. An EPUB imported from elsewhere is forgotten. An EPUB that sits
beside the audiobook is never deleted: Amba records that you removed it, so the
next scan leaves it detached. Importing an EPUB again clears that record.

## Getting started

### Requirements

- Go 1.27 or newer.
- `ffmpeg` and `ffprobe` on `PATH`. `ffprobe` reads audio durations instantly;
  `ffmpeg` cuts audio into chunks for transcription.
- A sherpa-onnx Whisper **tiny.en** model. If `models/sherpa-onnx-whisper-tiny.en`
  is missing, the app downloads and installs it on first use.

### Books

Put each audiobook in its own folder under `audiobooks/`:

```
audiobooks/
  Neuromancer/
    01.mp3
    02.mp3
    Neuromancer.epub      # optional
```

Files in a folder are one book, in natural order (`2` before `10`). An EPUB
beside them supplies the title, author, cover and text. An EPUB elsewhere can
also be imported from the reader with **Import EPUB**. Playable audio formats
are the ones mygo decodes: **MP3, OGG Vorbis and WAV**. Other containers
(M4A, M4B, MP4, FLAC) are not shown, because they would fail to play.

### Run

```sh
go run .
```

Build and check everything with:

```sh
go build ./...
go vet ./...
go test ./...
go tool github.com/egoist/mygo/cmd/mygo vet
```

## How the words are timed

The first time a book is opened, Amba transcribes it with sherpa-onnx's Whisper
tiny.en model. Audio is converted to mono 16 kHz WAV chunks of 28 seconds and
each chunk is recognised. The resulting timed words are cached as JSON, keyed
by the audio files' paths, sizes and modification times, so a book is
transcribed only once.

The standard Whisper tiny.en export has no cross-attention weights, so sherpa
cannot supply exact token timestamps for it. Amba still asks for them (a model
exported with attention outputs gets exact word times), and when they are
absent it estimates them: a simple energy detector finds the spoken stretches
of each chunk, and the recognised words are spread over those stretches in
proportion to their length. Pauses therefore interrupt the highlighting rather
than stretching words across them. Word order is exact either way, which is
what the EPUB alignment needs.

When an EPUB exists, `internal/align` matches the book's words to the narration:

- exact anchors of 2 to 5 words that occur once in both sequences define the
  correspondence, and fuzzy anchors recover passages with small Whisper errors;
- generic and frequently repeated phrases are ignored;
- the short segments between anchors are aligned with dynamic programming using
  word similarity (or a bounded greedy scan for very large gaps, such as front
  matter the narrator skips);
- the result is monotonic, so reading order is preserved.

Alignment is evaluated chapter by chapter. A chapter is accepted only when its
confidence, ebook coverage and quality are each at least 80% by default; these
thresholds are configurable. Rejected chapters fall back to the narration's
transcript between the surrounding accepted chapters, so one different edition
chapter does not make the whole book unusable. If no chapter is accepted, the
reader uses the transcript exactly as it did before EPUB support.

`internal/readalong` maps narration words to displayed words, filling forward
across passages the narrator reads but the book does not show. The reader uses
that map to highlight the active word and to keep the chapter and paragraph in
view. Ebook words without a reliable match never receive an invented timestamp.

See [`docs/alignment.md`](docs/alignment.md) for the algorithm, confidence
formula and limitations.

## Where things are stored

- `os.UserCacheDir()/Amba` — transcripts (`books/<id>/transcript.json`),
  alignment results (`books/<id>/alignment.json`), imported EPUB paths
  (`books/<id>/epub-import.json`), removed EPUB records
  (`books/<id>/epub-removed.json`) and listening progress
  (`progress/<id>.json`). On macOS that is `~/Library/Caches/Amba`.
- `models/sherpa-onnx-whisper-tiny.en` — the speech model, installed from the
  official sherpa-onnx release when it is not present.
- `audiobooks/`, `models/` and `.mygo/` are ignored by git; media and models
  are not part of the repository.

## Architecture

`main.go` is only a composition root: it creates the app, attaches the window
and starts it. Everything else lives in `internal/`:

| Package | Responsibility |
| --- | --- |
| `app` | Screens, routing, background preparation, reader state, the ebook drawer and progress saving. Playback is an interface, so the reader can be tested without audio hardware. |
| `library` | Finds books on disk, reads EPUB metadata and stores per-book progress. |
| `epub` | Reads EPUB metadata, cover, table of contents, reading order, sentences, words, offsets and plain text with the standard library and `x/net/html`. |
| `transcribe` | Whisper transcription, model installation, audio conversion and the transcript cache. |
| `media` | Audio durations via `ffprobe`. |
| `player` | Plays a list of files as one timeline, with position, seeking, speed and track advance. |
| `readalong` | Builds timed content, evaluates chapter acceptance, persists alignment results, lays the printed book out for the drawer and falls back to the transcript. |
| `align` | Fuzzy monotonic alignment with multi-size anchors, word similarity, confidence scoring and local alignment. |
| `textutil` | Word normalisation shared by alignment and display. |
| `store` | Atomic JSON cache files. |

The reader never decodes audio itself and the UI never parses books: each
screen asks a package for data, and long work (scanning, transcription,
alignment) runs off the main thread, publishing results through
`App.Update`.

## Limits

- Whisper tiny.en is an English, small model: transcription is approximate and
  the first run of a long audiobook takes a while. Cached runs are instant.
- With the standard model export, word times are estimated from the narration's
  pauses rather than measured, so a highlighted word can lead or lag the voice
  slightly. A whisper model exported with attention outputs is used verbatim
  when it is installed.
- EPUB sync tolerates minor transcription errors and small edition differences.
  Heavily rewritten passages may be rejected chapter by chapter and fall back
  to the transcript. Images, tables and footnotes are reduced to reading-order
  text.
- Alignment is lexical: paraphrases that share no words cannot be matched.
- Only the audio formats mygo can decode are playable; the library hides the
  rest rather than showing a book that cannot be opened.
