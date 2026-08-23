"""Word timing model: validation, distribution, lookup, persistence."""

import pytest

from app.timing.words import (
    TimingDocument,
    Word,
    distribute_word_timings,
    find_word_index_at,
    validate_words,
)


def test_distribute_word_timings_is_monotonic_and_covers_duration():
    words = distribute_word_timings("The quick brown fox jumps.", duration=3.0)
    assert len(words) == 5
    validate_words(words)
    assert words[0].start == 0.0
    assert words[-1].end == pytest.approx(3.0, abs=0.01)
    for a, b in zip(words, words[1:]):
        assert b.start >= a.start


def test_distribute_empty_text_returns_empty():
    assert distribute_word_timings("", 5.0) == []
    assert distribute_word_timings("hello", 0.0) == []


def test_validate_rejects_bad_ranges():
    with pytest.raises(ValueError):
        validate_words([Word(0, "x", start=1.0, end=1.0)])


def test_validate_rejects_non_monotonic():
    with pytest.raises(ValueError):
        validate_words([
            Word(0, "a", start=0.0, end=1.0),
            Word(1, "b", start=0.5, end=2.0),  # overlaps previous
        ])


def test_timing_document_save_load_roundtrip(tmp_path):
    doc = TimingDocument(
        chunk_id="chunk_001",
        text="The quick brown fox.",
        words=distribute_word_timings("The quick brown fox.", 2.0),
        duration=2.0,
    )
    path = tmp_path / "chunk.json"
    doc.save(path)

    loaded = TimingDocument.load(path)
    assert loaded.chunk_id == "chunk_001"
    assert loaded.text == "The quick brown fox."
    assert [w.text for w in loaded.words] == ["The", "quick", "brown", "fox."]
    assert loaded.duration == pytest.approx(2.0)


def test_find_word_index_at_forward_scan():
    words = [
        Word(0, "The", 0.00, 0.21),
        Word(1, "quick", 0.22, 0.54),
        Word(2, "brown", 0.55, 0.87),
        Word(3, "fox", 0.88, 1.10),
    ]
    hint = 0
    hint = find_word_index_at(words, 0.60, hint)
    assert words[hint].text == "brown"
    # Advancing further reuses the hint (no rescan).
    hint = find_word_index_at(words, 1.00, hint)
    assert words[hint].text == "fox"


def test_find_word_index_empty():
    assert find_word_index_at([], 1.0) == -1


def test_json_shape_matches_plan():
    word = Word(index=0, text="The", start=0.0, end=0.21)
    assert word.to_dict() == {"index": 0, "text": "The", "start": 0.0, "end": 0.21}
