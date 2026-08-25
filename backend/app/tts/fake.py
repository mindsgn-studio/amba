"""Deterministic fake TTS for tests and offline development.

Generates a quiet sine wave with evenly distributed word timings. Duration is
proportional to word count so the pipeline behaves realistically.
"""

from __future__ import annotations

import numpy as np

from app.tts.base import SynthesisResult, TTSProvider
from app.timing.words import Word, distribute_word_timings

SAMPLE_RATE = 24000
SECONDS_PER_WORD = 0.3


class FakeTTS(TTSProvider):
    name = "fake"

    def synthesize(self, text: str, voice: str, speed: float = 1.0,
                   language: str = "en") -> SynthesisResult:
        tokens = [t for t in text.split() if t]
        duration = max(0.5, len(tokens) * SECONDS_PER_WORD / max(speed, 0.01))
        total_samples = int(duration * SAMPLE_RATE)
        t = np.arange(total_samples, dtype=np.float32) / SAMPLE_RATE
        samples = (0.05 * np.sin(2 * np.pi * 220.0 * t)).astype(np.float32)

        words: list[Word] = []
        if tokens:
            words = distribute_word_timings(text, duration)
            # Re-index defensively (distribute already does).
            words = [Word(index=i, text=w.text, start=w.start, end=w.end)
                     for i, w in enumerate(words)]

        return SynthesisResult(samples=samples, sample_rate=SAMPLE_RATE, words=words)


PROVIDER = FakeTTS()
