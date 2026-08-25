"""Text normalization.

Deliberately conservative: the text stored here is exactly what the mobile
app renders and what the TTS speaks, so transformations must be predictable.
"""

from __future__ import annotations

import re
import unicodedata

# Smart punctuation -> ASCII equivalents (displayed text stays consistent).
CHAR_MAP = {
    "\u2018": "'", "\u2019": "'", "\u201a": ",",          # single quotes
    "\u201c": '"', "\u201d": '"',                          # double quotes
    "\u2013": "-", "\u2014": "-", "\u2212": "-",
    "\u2026": "...",                                       # ellipsis
    "\u00a0": " ", "\u2007": " ", "\u202f": " ",           # nbsp variants
    "\ufb00": "ff", "\ufb01": "fi", "\ufb02": "fl",        # ligatures
    "\ufb03": "ffi", "\ufb04": "ffl",
    "\u200b": "", "\u200c": "", "\ufeff": "",              # zero-width
    "\u00ad": "",                                          # soft hyphen
}

MULTI_SPACE_RE = re.compile(r"[ \t\r\f\v]+")
SPACE_BEFORE_PUNCT_RE = re.compile(r"\s+([,.!?;:])")
REPEAT_BANG_RE = re.compile(r"!{2,}")
REPEAT_QUERY_RE = re.compile(r"\?{2,}")
REPEAT_COMMA_RE = re.compile(r",{2,}")
DASH_SPACING_RE = re.compile(r"\s*-\s*")
LEADING_DASH_RE = re.compile(r"(?<=\w)-(?=\w)")  # placeholder, kept for clarity


def normalize_chars(text: str) -> str:
    for src, dst in CHAR_MAP.items():
        text = text.replace(src, dst)
    return unicodedata.normalize("NFC", text)


def normalize_line(text: str) -> str:
    """Normalize a single line: unicode, spacing, punctuation repeats."""
    text = normalize_chars(text)
    text = MULTI_SPACE_RE.sub(" ", text)
    text = SPACE_BEFORE_PUNCT_RE.sub(r"\1", text)
    text = REPEAT_BANG_RE.sub("!", text)
    text = REPEAT_QUERY_RE.sub("?", text)
    text = REPEAT_COMMA_RE.sub(",", text)
    return text.strip()


def dehyphenate(lines: list[str]) -> list[str]:
    """Join lines that were split mid-word with a hyphen: 'exam-' + 'ple'.

    Line-end hyphens are treated as word breaks and removed when joining.
    This matches the pipeline requirement (plan section 11); the rare hard
    compound broken across lines loses its hyphen, which is harmless here
    because displayed text always mirrors the normalized text exactly.
    """
    merged: list[str] = []
    for line in lines:
        line = line.strip()
        if merged and merged[-1].endswith("-"):
            merged[-1] = merged[-1][:-1] + line
        else:
            merged.append(line)
    return merged


def normalize_line_breaks(block: str) -> str:
    """Turn a block of wrapped lines into flowing paragraph text."""
    lines = [l.strip() for l in block.splitlines() if l.strip()]
    if not lines:
        return ""
    joined = " ".join(dehyphenate(lines))
    return normalize_line(joined)


def normalize_paragraphs(paragraphs: list[str]) -> list[str]:
    result: list[str] = []
    for paragraph in paragraphs:
        normalized = normalize_line_breaks(paragraph)
        if normalized:
            result.append(normalized)
    return result


def normalize_document_paragraphs(paragraphs: list[str]) -> list[str]:
    """Public alias used by the worker pipeline."""
    return normalize_paragraphs(paragraphs)
