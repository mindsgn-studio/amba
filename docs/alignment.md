# EPUB alignment

Amba can use an EPUB as the reading surface while keeping the audiobook's
existing transcript as the clock. The feature is optional: without an EPUB the
reader uses the transcript exactly as before.

## Pipeline

1. The audiobook is transcribed once by `internal/transcribe` and cached.
2. The EPUB is read by `internal/epub`, which preserves the printed text while
   also producing normalised words, sentences and byte offsets.
3. `internal/align` matches the ebook's words to the transcript's words with a
   monotonic sequence alignment.
4. `internal/readalong` turns the alignment into timed content, chapter by
   chapter.
5. The app caches the alignment in `books/<id>/alignment.json`.

Importing or changing an EPUB never starts another transcription. The
transcript cache is keyed by the audio files; the alignment cache is keyed by
the transcript hash, the extracted ebook hash, the algorithm version and the
alignment configuration.

## Word normalisation

`internal/textutil` lower-cases words, normalises Unicode to NFC, folds
typographic apostrophes and quotation marks, removes punctuation and hyphens,
treats non-breaking spaces as spaces, and reads `&` as "and". Normalisation is
only for matching. The original text is kept for display.

`align.WordSimilarity` compares two normalised words with Levenshtein distance:
identical words score 1, one edit in a long word scores highly, and unrelated
words score low. Empty words never match, and very short words must be equal.

## Anchors

The engine first finds exact n-gram anchors of 2, 3, 4 and 5 words that occur
once in both sequences. Larger anchors are preferred. Generic phrases such as
"and then", "in the end" and "he said" are ignored, and anchors that occur too
often are not used.

If exact anchors are not enough, the engine looks for fuzzy anchors. It indexes
target n-grams by their distinctive (non-generic) words, then compares only
candidate n-grams word by word. This finds passages where Whisper misheard a
word without comparing every ebook word with every audio word.

Candidates are ordered by a weighted longest-increasing-subsequence pass and
then filtered so anchors do not overlap. This preserves reading order and
avoids matching a repeated phrase to the wrong occurrence.

## Gaps and dynamic programming

The words between anchors are aligned with Needleman-Wunsch dynamic
programming using `WordSimilarity` scores. Insertions, deletions,
substitutions, punctuation differences and small spelling changes are allowed.
A gap larger than `Config.MaxDPCells` uses a bounded greedy forward scan
instead, which keeps memory bounded for front matter that the narrator skips.

`align.LocalAlign` also provides Smith-Waterman local alignment for finding the
best matching passage when chapter boundaries differ between editions.

## Confidence

All metrics are between 0 and 1:

- **Ebook coverage** — matched ebook words / all ebook words.
- **Audio coverage** — matched narration words / all narration words. Extra
  introductions or acknowledgements therefore count as unmatched, but do not
  by themselves make a good ebook alignment fail.
- **Quality** — average word similarity, reduced when only a small part of the
  shorter sequence matched.
- **Confidence** — `0.40*ebook + 0.30*audio + 0.30*quality`.

A chapter or book is accepted only when:

- confidence is at least `MinConfidence` (default 0.80),
- ebook coverage is at least `MinEbookCoverage` (default 0.80),
- quality is at least `MinQuality` (default 0.80),
- there is enough sequential evidence (default 3 matched words, or the whole
  chapter when it is shorter).

These are alignment-quality scores, not calibrated probabilities. All
thresholds are configurable through `align.Config`. Rejections include
diagnostics explaining which rule failed.

## Chapters and fallback

Alignment runs over the whole ebook so passage anchors can locate corresponding
sections even when chapter divisions differ. Metrics are then evaluated per
EPUB chapter. One bad chapter does not invalidate the rest.

Accepted chapters show the EPUB text and highlight words with the audiobook's
timestamps. A rejected chapter is replaced by the transcript words for the
narration region between the surrounding accepted chapters, with a clear notice
in the reader. If no chapter is accepted, the app uses the existing transcript
reading experience for the whole book.

Ebook words without a reliable match keep `Audio == -1`; the reader does not
invent a timestamp for them. Narration words without a display word keep the
previous display word lit so highlighting moves smoothly.

## Limits

- Alignment is lexical. It cannot match paraphrases that share no words.
- Different editions with heavily rewritten passages may be rejected chapter by
  chapter; those chapters fall back to the transcript.
- Repeated phrases are handled by unique anchors and monotonic ordering, but a
  book made almost entirely of repeated phrases has little distinctive signal.
- Hyphenation and contractions are normalised, but word-level alignment cannot
  recover a word that the narrator skips entirely.
- Timestamps come from the existing transcript. Amba does not measure the audio
  again when an EPUB is imported.
