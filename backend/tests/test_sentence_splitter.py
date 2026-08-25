"""Sentence splitting: abbreviations, decimals, initials, ellipses."""

from app.processing.sentence_splitter import split_sentences


def test_basic_split():
    assert split_sentences("The sun was setting. The wind became colder.") == [
        "The sun was setting.",
        "The wind became colder.",
    ]


def test_abbreviations_do_not_split():
    assert split_sentences("Dr. Smith arrived early. He left soon.") == [
        "Dr. Smith arrived early.",
        "He left soon.",
    ]
    assert len(split_sentences("Mr. and Mrs. Lee went home.")) == 1


def test_decimals_do_not_split():
    assert len(split_sentences("It costs $3.50 today. That is cheap.")) == 2


def test_initials_do_not_split():
    assert len(split_sentences("John F. Kennedy spoke. The crowd cheered.")) == 2
    assert len(split_sentences("John F. Kennedy spoke.")) == 1


def test_ellipsis_and_exclamation():
    sentences = split_sentences("Wait... what happened?! I don't know.")
    assert len(sentences) == 3


def test_no_terminal_punctuation_returns_whole_text():
    assert split_sentences("just some words here") == ["just some words here"]


def test_empty_input():
    assert split_sentences("") == []
    assert split_sentences("   ") == []


def test_sentences_preserve_text_when_rejoined():
    original = "One sentence here. Another one follows! A third asks: why not?"
    joined = " ".join(split_sentences(original))
    assert joined.replace(" ", "") == original.replace(" ", "")
