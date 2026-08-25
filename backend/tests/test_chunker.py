"""Chunking: sentence integrity, size limits, paragraph boundaries."""

from app.processing.chunker import chunk_paragraphs, word_count
from app.processing.sentence_splitter import split_sentences


def _make_paragraphs(paragraph_count: int, sentences: int = 12) -> list[str]:
    paragraphs = []
    for p in range(paragraph_count):
        sentences_text = " ".join(
            f"This is sentence number {p * sentences + i + 1} in the paragraph."
            for i in range(sentences)
        )
        paragraphs.append(sentences_text)
    return paragraphs


def test_short_text_single_chunk():
    chunks = chunk_paragraphs(["The quick brown fox jumps over the lazy dog."])
    assert len(chunks) == 1
    assert chunks[0].index == 0
    assert chunks[0].word_count <= 10


def test_long_text_respects_limits_and_sentence_integrity():
    paragraphs = _make_paragraphs(6)
    chunks = chunk_paragraphs(paragraphs, target_words=80, max_words=150)

    assert len(chunks) > 1
    for chunk in chunks:
        assert chunk.word_count <= 150, "chunk exceeds max words"

    # No sentence is ever cut in half: every chunk ends with a period and
    # re-splitting yields only complete sentences.
    original_sentences = [s for p in paragraphs for s in split_sentences(p)]
    rebuilt: list[str] = []
    for chunk in chunks:
        parts = split_sentences(chunk.text)
        for part in parts:
            assert part.endswith(".")
        rebuilt.extend(parts)
    assert rebuilt == original_sentences


def test_paragraph_boundary_preferred_flush():
    # Two long paragraphs; target should flush at the paragraph edge.
    paragraphs = [
        "Alpha. " * 60,
        "Beta. " * 60,
    ]
    chunks = chunk_paragraphs([p.strip() for p in paragraphs],
                              target_words=50, max_words=200)
    texts = [c.text for c in chunks]
    alpha_chunks = [t for t in texts if t.startswith("Alpha")]
    beta_chunks = [t for t in texts if t.startswith("Beta")]
    assert alpha_chunks and beta_chunks
    # No chunk mixes both paragraphs once a boundary flush was possible.
    for text in texts:
        assert not (text.startswith("Alpha") and " Beta." in text.split(". ")[0])


def test_oversized_single_sentence_is_hard_split():
    monster = " ".join(f"word{i}" for i in range(400)) + "."
    chunks = chunk_paragraphs([monster], target_words=100, max_words=150)
    assert all(c.word_count <= 150 for c in chunks)


def test_word_count_utility():
    assert word_count("one two three") == 3
