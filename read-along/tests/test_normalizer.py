"""Normalization: whitespace, unicode, quotes, hyphenation, punctuation."""

from app.processing.normalizer import (
    dehyphenate,
    normalize_line,
    normalize_line_breaks,
    normalize_paragraphs,
)


def test_collapses_multiple_spaces():
    assert normalize_line("The   quick\t\tbrown  fox") == "The quick brown fox"


def test_joins_broken_lines():
    assert normalize_line_breaks("The quick\nbrown fox.") == "The quick brown fox."


def test_hyphenated_word_break_is_repaired():
    assert normalize_line_breaks("an incre-\ndibly lazy dog") == "an incredibly lazy dog"


def test_line_end_hyphen_is_always_joined():
    # Line-end hyphens are treated as soft breaks (displayed text always
    # mirrors the normalized text, so sync is unaffected either way).
    assert normalize_line_breaks("a well-\nknown fact") == "a wellknown fact"


def test_unicode_nfc_and_smart_quotes():
    result = normalize_line("\u201cHello\u201d \u2014 she said\u2026")
    assert result == '"Hello" - she said...'


def test_space_before_punctuation_removed():
    assert normalize_line("Hello , world !") == "Hello, world!"


def test_repeated_punctuation_collapsed():
    assert normalize_line("Wow!!!! Really??? Ok,, fine") == "Wow! Really? Ok, fine"


def test_dehyphenate_merges_split_words():
    assert dehyphenate(["exam-", "ple of merging"]) == ["example of merging"]


def test_normalize_paragraphs_drops_empty_blocks():
    assert normalize_paragraphs(["First.", "", "   ", "Second."]) == ["First.", "Second."]
