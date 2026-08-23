"""Internal document model shared by every input format.

PDF, EPUB, TXT and raw strings all normalize into a `Document`, so the
audiobook engine never needs to know where the text came from.
"""

from __future__ import annotations

from dataclasses import dataclass, field
from pathlib import Path

SUPPORTED_EXTENSIONS = {".pdf": "pdf", ".epub": "epub", ".txt": "text"}


class UnsupportedDocumentError(ValueError):
    """Raised for scanned PDFs or otherwise unusable inputs."""


class DocumentParseError(ValueError):
    """Raised when a document cannot be parsed at all."""


@dataclass
class Chapter:
    title: str
    paragraphs: list[str] = field(default_factory=list)

    @property
    def word_count(self) -> int:
        return sum(len(p.split()) for p in self.paragraphs)


@dataclass
class Document:
    title: str = "Untitled"
    author: str | None = None
    language: str = "en"
    chapters: list[Chapter] = field(default_factory=list)

    @property
    def word_count(self) -> int:
        return sum(c.word_count for c in self.chapters)


def detect_source_type(filename: str) -> str:
    suffix = Path(filename).suffix.lower()
    source_type = SUPPORTED_EXTENSIONS.get(suffix)
    if source_type is None:
        raise UnsupportedDocumentError(
            f"Unsupported file type {suffix!r}. Supported: .pdf .epub .txt"
        )
    return source_type


def parse_document(source_type: str, path: Path | None = None, raw_text: str | None = None,
                   title_hint: str | None = None) -> Document:
    """Dispatch to the right parser based on `source_type`."""
    # Imported lazily so heavy dependencies (fitz/ebooklib) only load when used.
    if source_type == "text":
        from app.documents.text import parse_text

        if raw_text is None:
            if path is not None and Path(path).is_file():
                raw_text = Path(path).read_text(encoding="utf-8", errors="replace")
            else:
                raise DocumentParseError("Text parsing requires raw text or a file path")
        return parse_text(raw_text, title=title_hint or "Untitled")
    if source_type == "pdf":
        from app.documents.pdf import parse_pdf

        if path is None:
            raise DocumentParseError("PDF parsing requires a file path")
        return parse_pdf(path, title_hint)
    if source_type == "epub":
        from app.documents.epub import parse_epub

        if path is None:
            raise DocumentParseError("EPUB parsing requires a file path")
        return parse_epub(path)
    raise UnsupportedDocumentError(f"Unknown source type: {source_type!r}")
