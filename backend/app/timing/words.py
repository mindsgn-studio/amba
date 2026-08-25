"""Word-level timing model and helpers.

This is the heart of the read-along feature: each chunk stores its exact
source text plus per-word start/end seconds, so the mobile app only needs
`start <= current_time < end` to highlight the right word.
"""

from __future__ import annotations

import json
from dataclasses import dataclass, field
from pathlib import Path


@dataclass
class Word:
    index: int
    text: str
    start: float
    end: float

    def to_dict(self) -> dict:
        return {"index": self.index, "text": self.text,
                "start": round(self.start, 4), "end": round(self.end, 4)}

    @classmethod
    def from_dict(cls, data: dict) -> "Word":
        return cls(index=int(data["index"]), text=str(data["text"]),
                   start=float(data["start"]), end=float(data["end"]))


def validate_words(words: list[Word]) -> None:
    """Enforce: ordered, monotonic, start < end, sequential indexes."""
    previous_end = -1.0
    for i, word in enumerate(words):
        if word.end <= word.start:
            raise ValueError(f"word {i}: end ({word.end}) must be > start ({word.start})")
        if word.start < previous_end - 1e-6:
            raise ValueError(
                f"word {i}: start ({word.start}) overlaps previous end ({previous_end})"
            )
        if word.index != i:
            raise ValueError(f"word {i}: index mismatch ({word.index})")
        previous_end = word.end


@dataclass
class TimingDocument:
    chunk_id: str
    text: str
    words: list[Word] = field(default_factory=list)
    duration: float = 0.0

    def validate(self) -> None:
        validate_words(self.words)

    def to_dict(self) -> dict:
        return {
            "chunk_id": self.chunk_id,
            "text": self.text,
            "duration": round(self.duration, 4),
            "words": [w.to_dict() for w in self.words],
        }

    def save(self, path: Path) -> None:
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(json.dumps(self.to_dict(), ensure_ascii=False, indent=2),
                        encoding="utf-8")

    @classmethod
    def load(cls, path: Path) -> "TimingDocument":
        data = json.loads(path.read_text(encoding="utf-8"))
        return cls(
            chunk_id=data.get("chunk_id", ""),
            text=data.get("text", ""),
            words=[Word.from_dict(w) for w in data.get("words", [])],
            duration=float(data.get("duration", 0.0)),
        )


def distribute_word_timings(text: str, duration: float, chunk_id: str = "") -> list[Word]:
    """Fallback aligner when an engine has no native timestamps.

    Distributes `duration` across words proportionally to their length.
    Only used when the TTS engine cannot provide real timings.
    """
    tokens = [t for t in text.split() if t]
    if not tokens or duration <= 0:
        return []
    weights = [max(len(t), 1) + 1 for t in tokens]  # +1 approximates inter-word gap
    total = float(sum(weights))
    words: list[Word] = []
    cursor = 0.0
    for index, (token, weight) in enumerate(zip(tokens, weights)):
        span = duration * (weight / total)
        end = duration if index == len(tokens) - 1 else min(cursor + span * 0.96, duration)
        words.append(Word(index=index, text=token,
                          start=round(cursor, 4), end=round(max(end, cursor + 0.01), 4)))
        cursor += span
    validate_words(words)
    return words


def find_word_index_at(words: list[Word], current_time: float, hint: int = 0) -> int:
    """Efficient forward-only lookup.

    The caller keeps a persistent index (`hint`) between ticks; we advance it
    instead of scanning from zero every frame (plan section 32).
    """
    if not words:
        return -1
    i = max(0, min(hint, len(words) - 1))
    while i + 1 < len(words) and words[i].end < current_time:
        i += 1
    while i > 0 and words[i].start > current_time:
        i -= 1
    return i
