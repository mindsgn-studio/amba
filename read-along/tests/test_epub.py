"""EPUB extraction via ebooklib."""

from pathlib import Path

import pytest
from ebooklib import epub

from app.documents.base import DocumentParseError
from app.documents.epub import parse_epub


def _make_epub(path: Path) -> None:
    book = epub.EpubBook()
    book.set_identifier("test-book-1")
    book.set_title("Test Book")
    book.set_language("en")
    book.add_author("Jane Writer")

    chapter_1 = epub.EpubHtml(title="Chapter One", file_name="chap_01.xhtml", lang="en")
    chapter_1.content = (
        "<html><body>"
        "<h1>Chapter One</h1>"
        "<p>The quick brown fox jumps over the lazy dog.</p>"
        "<blockquote>Quote inside a chapter.</blockquote>"
        "<script>evil()</script><style>p{}</style>"
        "</body></html>"
    )
    chapter_2 = epub.EpubHtml(title="Chapter Two", file_name="chap_02.xhtml", lang="en")
    chapter_2.content = (
        "<html><body>"
        "<h2>Chapter Two</h2>"
        "<p>The sun was setting. The wind became colder.</p>"
        "<ul><li>First list item</li><li>Second list item</li></ul>"
        "</body></html>"
    )
    nav = epub.EpubNcx()
    nav_toc = epub.EpubNav()

    book.add_item(chapter_1)
    book.add_item(chapter_2)
    book.add_item(nav)
    book.add_item(nav_toc)
    book.spine = ["nav", chapter_1, chapter_2]
    epub.write_epub(str(path), book)


def test_epub_extraction(tmp_path: Path):
    epub_path = tmp_path / "test-book.epub"
    _make_epub(epub_path)

    document = parse_epub(epub_path)

    assert document.title == "Test Book"
    assert document.author == "Jane Writer"
    titles = [c.title for c in document.chapters]
    assert any("Chapter One" in t for t in titles)
    assert any("Chapter Two" in t for t in titles)

    all_text = " ".join(p for c in document.chapters for p in c.paragraphs)
    assert "quick brown fox" in all_text
    assert "Quote inside a chapter." in all_text
    assert "First list item" in all_text
    # Scripts and styles are dropped.
    assert "evil()" not in all_text


def test_corrupt_epub_raises_parse_error(tmp_path: Path):
    bad = tmp_path / "bad.epub"
    bad.write_bytes(b"not an epub at all")
    with pytest.raises(DocumentParseError):
        parse_epub(bad)
