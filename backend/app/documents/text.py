"""Raw string / .txt input.

Text inputs bypass heavy document extraction: they only get paragraph
splitting on blank lines. Normalization happens later in the pipeline.
"""

from __future__ import annotations

import re

from app.documents.base import Chapter, Document


def parse_text(raw: str, title: str = "Untitled") -> Document:
    paragraphs = [
        re.sub(r"\s+", " ", block).strip()
        for block in re.split(r"\n\s*\n", raw)
    ]
    paragraphs = [p for p in paragraphs if p]
    return Document(
        title=title,
        chapters=[Chapter(title="Content", paragraphs=paragraphs)],
    )
