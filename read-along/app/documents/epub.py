"""EPUB extraction via ebooklib + BeautifulSoup.

EPUB is already structured: we walk the spine, parse each XHTML document,
drop scripts/styles/navigation, and keep headings + paragraphs + quotes.
"""

from __future__ import annotations

import logging
import re
from pathlib import Path

import ebooklib
from bs4 import BeautifulSoup
from ebooklib import epub

from app.documents.base import Chapter, Document, DocumentParseError

logger = logging.getLogger(__name__)

DROP_TAGS = ("script", "style", "nav", "aside", "form", "head")
TEXT_TAGS = ("p", "blockquote", "li")
HEADING_TAGS = ("h1", "h2", "h3")


def _clean_text(value: str | None) -> str:
    return re.sub(r"\s+", " ", value or "").strip()


def parse_epub(path: Path) -> Document:
    try:
        book = epub.read_epub(str(path), options={"ignore_ncx": True})
    except Exception as exc:  # noqa: BLE001 - surface any epublib failure clearly
        raise DocumentParseError(f"Failed to read EPUB: {exc}") from exc

    meta_title = book.get_metadata("DC", "title")
    meta_author = book.get_metadata("DC", "creator")
    meta_language = book.get_metadata("DC", "language")

    title = _clean_text(meta_title[0][0]) if meta_title else path.stem.replace("_", " ").title()
    author = _clean_text(meta_author[0][0]) if meta_author else None
    language = (_clean_text(meta_language[0][0]) or "en") if meta_language else "en"

    documents = [
        item for item in book.get_items_of_type(ebooklib.ITEM_DOCUMENT)
        if not _is_nav_document(item)
    ]

    chapters: list[Chapter] = []
    for order, item in enumerate(documents, start=1):
        soup = BeautifulSoup(item.get_content(), "html.parser")
        for tag in soup.find_all(DROP_TAGS):
            tag.decompose()

        heading = soup.find(HEADING_TAGS)
        chapter_title = _clean_text(heading.get_text()) if heading else ""
        if not chapter_title:
            toc_title = _toc_title(book, item)
            chapter_title = toc_title or f"Chapter {order}"

        paragraphs = _extract_paragraphs(soup)
        if paragraphs:
            chapters.append(Chapter(title=chapter_title, paragraphs=paragraphs))

    if not chapters:
        raise DocumentParseError("No readable text found in EPUB")

    logger.info("Parsed EPUB title=%r chapters=%d words=%d", title, len(chapters),
                sum(c.word_count for c in chapters))
    return Document(title=title, author=author, language=language, chapters=chapters)


def _is_nav_document(item) -> bool:
    name = (item.get_name() or "").lower()
    props = getattr(item, "properties", None) or []
    return "nav" in props or name.endswith(("nav.xhtml", "toc.xhtml", "titlepage.xhtml"))


def _extract_paragraphs(soup: BeautifulSoup) -> list[str]:
    paragraphs: list[str] = []
    for element in soup.find_all(TEXT_TAGS):
        parent = element.parent.name.lower() if element.parent else ""
        # Skip nested duplicates (e.g. <p> inside <blockquote>, <ul> inside <li>).
        if parent in TEXT_TAGS:
            continue
        text = _clean_text(element.get_text(" "))
        if text:
            paragraphs.append(text)
    return paragraphs


def _toc_title(book: epub.EpubBook, item) -> str | None:
    href = (item.get_name() or "").split("/")[-1]
    for link in book.toc:
        target = link.href if isinstance(link, epub.Link) else None
        if target and target.split("/")[-1] == href:
            return _clean_text(link.title) or None
    return None
