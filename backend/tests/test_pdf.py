"""PDF extraction: headers/footers, page numbers, hyphenation, chapters."""

from pathlib import Path

import fitz
import pytest

from app.documents.base import UnsupportedDocumentError
from app.documents.pdf import parse_pdf


def _make_pdf(path: Path) -> None:
    doc = fitz.open()
    body = [
        "Chapter 1",
        "The sun was setting over the quiet village. The wind became "
        "colder as evening approached the little houses on the hill.",
        "Everyone hurried indoors before dark.",
        "Chapter 2",
        "Morning arrived with pale light. A thin mist covered the fields "
        "and the roosters began their noisy chorus at first sun.",
    ]
    header = "My Book - Draft Edition"
    for page_number in range(1, 4):
        page = doc.new_page()  # A4-ish default
        page.insert_text((200, 30), header, fontsize=9)
        y = 80
        for paragraph in body:
            # Wrap each paragraph into short lines to simulate real layout.
            words = paragraph.split()
            line = ""
            for word in words:
                if len(line) + len(word) > 40:
                    page.insert_text((72, y), line, fontsize=11)
                    y += 14
                    line = word
                else:
                    line = f"{line} {word}".strip()
            if line:
                page.insert_text((72, y), line, fontsize=11)
                y += 20
        page.insert_text((290, 800), str(page_number), fontsize=9)
    doc.save(str(path))
    doc.close()


def test_pdf_extraction(tmp_path: Path):
    pdf_path = tmp_path / "book.pdf"
    _make_pdf(pdf_path)

    document = parse_pdf(pdf_path, title_hint="Test Book")
    text = " ".join(p for c in document.chapters for p in c.paragraphs)

    # Content survives.
    assert "sun was setting" in text
    assert "roosters began" in text
    # Repeated header removed.
    assert "Draft Edition" not in text
    # Bare page numbers removed.
    for paragraph in [p for c in document.chapters for p in c.paragraphs]:
        assert not paragraph.strip().isdigit()
    # Chapter headings detected.
    titles = [c.title.lower() for c in document.chapters]
    assert any("chapter 1" in t for t in titles)
    assert any("chapter 2" in t for t in titles)


def test_pdf_hyphenated_line_break_is_joined(tmp_path: Path):
    doc = fitz.open()
    page = doc.new_page()
    page.insert_text((72, 100), "The quick brown fox jumped over an incre-")
    page.insert_text((72, 120), "dibly lazy dog yesterday afternoon.")
    doc.save(str(tmp_path / "hyphen.pdf"))
    doc.close()

    document = parse_pdf(tmp_path / "hyphen.pdf", title_hint="H")
    joined = " ".join(document.chapters[0].paragraphs)
    assert "incredibly lazy dog" in joined


def test_scanned_pdf_rejected(tmp_path: Path):
    doc = fitz.open()
    page = doc.new_page()
    # Blank image, no text layer -> looks like a scanned page.
    page.insert_image(page.rect, pixmap=fitz.Pixmap(fitz.csRGB, page.rect.irect, False))
    doc.save(str(tmp_path / "scan.pdf"))
    doc.close()

    with pytest.raises(UnsupportedDocumentError):
        parse_pdf(tmp_path / "scan.pdf", title_hint="Scanned")
