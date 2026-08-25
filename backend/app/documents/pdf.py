"""PDF extraction via PyMuPDF.

Pipeline: pages -> text blocks -> header/footer + page-number removal ->
hyphenation repair -> paragraph detection -> (light) chapter detection.

Scanned PDFs are rejected (no OCR in v1).
"""

from __future__ import annotations

import logging
import re
from collections import Counter
from pathlib import Path

import fitz  # PyMuPDF

from app.documents.base import Chapter, Document, UnsupportedDocumentError
from app.processing.normalizer import normalize_line_breaks

logger = logging.getLogger(__name__)

MIN_CHARS_PER_PAGE = 25  # below this a page counts as empty/scanned
PAGE_NUMBER_RE = re.compile(r"^\s*(?:page\s*)?[-–—\[(]?\s*\d{1,4}\s*[-–—\])]?\s*$", re.IGNORECASE)
CHAPTER_HEADING_RE = re.compile(
    r"^\s*((?:chapter|part|section|book)\s+(?:[ivxlcdm]+|\d+|[a-z]+))\b[.:—–-]?\s*(.*)$",
    re.IGNORECASE,
)
TERMINAL_PUNCT = tuple(".!?:;\"')]}…")


def _is_scanned(total_chars: int, page_count: int) -> bool:
    return page_count == 0 or total_chars < MIN_CHARS_PER_PAGE * page_count


def _strip_repeated_lines(pages_lines: list[list[str]]) -> None:
    """Remove headers/footers repeated across many pages, and bare page numbers."""
    if len(pages_lines) < 3:
        # Not enough pages to distinguish repetition from content.
        for lines in pages_lines:
            lines[:] = [l for l in lines if not PAGE_NUMBER_RE.match(l)]
        return

    def common_edge(position: int) -> set[str]:
        counter: Counter[str] = Counter()
        for lines in pages_lines:
            if len(lines) > position:
                counter[lines[position].strip().lower()] += 1
        threshold = max(3, int(len(pages_lines) * 0.6))
        return {text for text, n in counter.items() if n >= threshold and len(text) > 1}

    headers = common_edge(0)
    footers = common_edge(-1)

    for lines in pages_lines:
        cleaned: list[str] = []
        for i, line in enumerate(lines):
            stripped = line.strip()
            if PAGE_NUMBER_RE.match(stripped):
                continue
            if i == 0 and stripped.lower() in headers:
                continue
            if stripped.lower() in footers and (
                i >= len(lines) - 2 or PAGE_NUMBER_RE.match(stripped)
            ):
                continue
            cleaned.append(line)
        lines[:] = cleaned


def parse_pdf(path: Path, title_hint: str | None = None) -> Document:
    with fitz.open(path) as doc:
        page_count = doc.page_count
        raw_pages: list[list[str]] = []
        total_chars = 0
        for page in doc:
            text = page.get_text("text") or ""
            total_chars += len(text.strip())
            lines = [l for l in (ln.rstrip() for ln in text.splitlines()) if l.strip()]
            raw_pages.append(lines)

    if _is_scanned(total_chars, page_count):
        raise UnsupportedDocumentError(
            "PDF appears to be scanned (no extractable text). OCR is not supported yet."
        )

    _strip_repeated_lines(raw_pages)

    paragraphs: list[str] = []
    chapters: list[tuple[str | None, int]] = []  # (title, paragraph_index)

    for lines in raw_pages:
        for block in _blocks_from_lines(lines):
            normalized = normalize_line_breaks(block)
            if not normalized:
                continue
            heading = CHAPTER_HEADING_RE.match(normalized)
            if heading and len(normalized) <= 80:
                chapters.append((normalized.strip(), len(paragraphs)))
                paragraphs.append(normalized)
                continue
            if (
                paragraphs
                and not paragraphs[-1].endswith(TERMINAL_PUNCT)
                and normalized[:1].islower()
            ):
                # Paragraph continues across a page break.
                paragraphs[-1] = f"{paragraphs[-1]} {normalized}"
                continue
            paragraphs.append(normalized)

    document_chapters = _group_into_chapters(paragraphs, chapters, title_hint)
    logger.info("Parsed PDF pages=%d paragraphs=%d chapters=%d",
                page_count, len(paragraphs), len(document_chapters))
    return Document(
        title=title_hint or "Untitled",
        language="en",
        chapters=document_chapters,
    )


def _blocks_from_lines(lines: list[str]) -> list[str]:
    """Group consecutive lines into paragraph blocks.

    A block ends at a line with sentence-terminal punctuation, or right
    before a standalone chapter heading line.
    """
    blocks: list[str] = []
    current: list[str] = []

    def flush() -> None:
        nonlocal current
        if current:
            blocks.append(" ".join(current))
            current = []

    for line in lines:
        if CHAPTER_HEADING_RE.match(line.strip()) and len(line.strip()) <= 80:
            flush()
            blocks.append(line.strip())
            continue
        # Repair words broken across lines BEFORE whitespace-joining.
        if current and current[-1].rstrip().endswith("-"):
            current[-1] = current[-1].rstrip()[:-1] + line.strip()
        else:
            current.append(line)
        if line.rstrip().endswith(TERMINAL_PUNCT):
            flush()
    flush()
    return blocks


def _group_into_chapters(paragraphs: list[str], headings: list[tuple[str | None, int]],
                         title_hint: str | None) -> list[Chapter]:
    if not headings:
        return [Chapter(title=title_hint or "Content", paragraphs=paragraphs)]

    result: list[Chapter] = []
    boundaries = [(idx, title) for title, idx in headings]
    first_title = boundaries[0][1]
    if boundaries[0][0] > 0:
        result.append(Chapter(title="Front Matter" if not first_title else first_title,
                              paragraphs=paragraphs[: boundaries[0][0]]))
    for i, (start_idx, heading_title) in enumerate(boundaries):
        end_idx = boundaries[i + 1][0] if i + 1 < len(boundaries) else len(paragraphs)
        body = paragraphs[start_idx + 1: end_idx] if start_idx < end_idx else []
        chapter_body = [paragraphs[start_idx], *body] if not body else body
        if any(p.strip() for p in chapter_body):
            result.append(Chapter(title=heading_title or f"Section {i + 1}",
                                  paragraphs=chapter_body))
    return result or [Chapter(title=title_hint or "Content", paragraphs=paragraphs)]
