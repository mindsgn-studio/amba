"""Chunking by semantic boundaries.

Preferred order: Chapter -> Paragraph -> Sentence -> Chunk.
Chunks are built from whole sentences; paragraph ends are preferred flush
points once the target size is reached, and no chunk exceeds `max_words`
unless a single sentence alone is longer than that.
"""

from __future__ import annotations

from dataclasses import dataclass

from app.processing.sentence_splitter import split_sentences


@dataclass
class Chunk:
    index: int
    text: str
    word_count: int


def word_count(text: str) -> int:
    return len(text.split())


def chunk_paragraphs(
    paragraphs: list[str],
    target_words: int = 800,
    max_words: int = 1500,
) -> list[Chunk]:
    """Group sentences from `paragraphs` into TTS-sized chunks."""
    chunks: list[Chunk] = []
    current_sentences: list[str] = []
    current_words = 0
    reached_target = False

    def flush() -> None:
        nonlocal current_sentences, current_words, reached_target
        if not current_sentences:
            return
        text = " ".join(current_sentences)
        chunks.append(Chunk(index=len(chunks), text=text, word_count=word_count(text)))
        current_sentences = []
        current_words = 0
        reached_target = False

    for paragraph in paragraphs:
        for sentence in split_sentences(paragraph):
            sentence_words = word_count(sentence)

            if sentence_words > max_words:
                # Oversized single sentence: flush what we have, then hard-split it.
                flush()
                for part in _hard_split_sentence(sentence, max_words):
                    if current_sentences and current_words + word_count(part) > max_words:
                        flush()
                    current_sentences.append(part)
                    current_words += word_count(part)
                continue

            if current_sentences and current_words + sentence_words > max_words:
                flush()

            current_sentences.append(sentence)
            current_words += sentence_words

            if current_words >= target_words:
                reached_target = True

        # Paragraph boundary: preferred place to stop once we hit the target.
        if reached_target and current_words >= target_words:
            flush()

    flush()
    return chunks


def _hard_split_sentence(sentence: str, limit: int) -> list[str]:
    """Last-resort split of a pathologically long sentence at word edges."""
    words = sentence.split()
    parts: list[str] = []
    buffer: list[str] = []
    count = 0
    for token in words:
        buffer.append(token)
        count += 1
        if count >= limit:
            parts.append(" ".join(buffer))
            buffer = []
            count = 0
    if buffer:
        parts.append(" ".join(buffer))
    return parts
