"""Raw string / .txt input -> Document."""

from app.documents.text import parse_text


def test_simple_string_becomes_single_chapter():
    doc = parse_text("The quick brown fox jumps over the lazy dog.", title="Example")
    assert doc.title == "Example"
    assert len(doc.chapters) == 1
    assert doc.chapters[0].title == "Content"
    assert doc.chapters[0].paragraphs == ["The quick brown fox jumps over the lazy dog."]


def test_blank_lines_split_paragraphs():
    doc = parse_text("First paragraph.\n\nSecond paragraph.\n\n\nThird one.")
    assert len(doc.chapters[0].paragraphs) == 3
    assert doc.chapters[0].paragraphs[1] == "Second paragraph."


def test_single_newlines_do_not_split_paragraphs():
    doc = parse_text("Line one\nline two still same paragraph.\n\nNext.")
    paragraphs = doc.chapters[0].paragraphs
    assert paragraphs[0] == "Line one line two still same paragraph."
    assert paragraphs[1] == "Next."


def test_empty_input_yields_chapter_without_paragraphs():
    doc = parse_text("   \n \n ")
    assert doc.chapters[0].paragraphs == []
