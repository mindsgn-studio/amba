"""Sentence splitting.

Regex-based splitter with abbreviation, initial and decimal protection so
chunks never cut sentences in half at things like "Dr." or "3.5".
"""

from __future__ import annotations

import re

ABBREVIATIONS = {
    "mr", "mrs", "ms", "dr", "prof", "sr", "jr", "st", "vs", "etc",
    "e.g", "i.e", "fig", "no", "vol", "approx", "dept", "est", "inc",
    "ltd", "co", "corp", "univ", "capt", "sgt", "lt", "col", "gen",
}

TERMINALS = set(".!?")
CLOSERS = "\"')]}"
_INITIALS_RE = re.compile(r"(?:^|[\s(])(?:[A-Za-z]\.)+$")



def split_sentences(text: str) -> list[str]:
    """Split `text` into complete sentences, preserving original wording."""
    if not text or not text.strip():
        return []

    sentences: list[str] = []
    current: list[str] = []
    i = 0
    n = len(text)

    while i < n:
        char = text[i]
        current.append(char)

        if char in TERMINALS:
            i += 1
            # Swallow any run of terminals (e.g. "...", "!?") plus closers.
            while i < n and (text[i] in TERMINALS or text[i] in CLOSERS):
                current.append(text[i])
                i += 1
            # Swallow closing quote right after the terminal run.
            while i < n and text[i] in CLOSERS:
                current.append(text[i])
                i += 1
            if _is_sentence_boundary(text, i):
                sentence = "".join(current).strip()
                if sentence:
                    sentences.append(sentence)
                current = []
            continue
        i += 1

    tail = "".join(current).strip()
    if tail:
        if sentences:
            sentences[-1] = f"{sentences[-1]} {tail}"
        else:
            sentences.append(tail)
    return sentences


def _is_sentence_boundary(text: str, pos: int) -> bool:
    """Decide whether the terminal at `pos` really ends a sentence.

    `pos` points just past the terminal run.
    """
    n = len(text)
    if pos >= n:
        return True
    next_char = text[pos]
    if next_char not in (" ", "\n", "\t"):
        # e.g. "3.5" or "example.com" -> not a boundary
        if next_char.isalnum():
            return False
        if next_char in CLOSERS:
            # look past the closer
            j = pos
            while j < n and text[j] in CLOSERS:
                j += 1
            if j < n and text[j].isalnum():
                return False

    # Everything up to and including the terminal run, e.g. "...John F."
    preceding = text[:pos].rstrip()
    stripped = preceding.rstrip(".")

    if _INITIALS_RE.search(preceding):
        return False  # "John F." / "J.F.K."

    token_match = re.search(r"([A-Za-z.]+)$", stripped)
    if token_match and token_match.group(1).lower() in ABBREVIATIONS:
        return False  # "Dr.", "e.g."

    return True
