"""Kokoro-82M local TTS engine.

Uses the official `kokoro` package (hexgrad, Apache-2.0). English pipelines
yield word-level tokens (`result.tokens`) which we convert into Word timings,
offsetting each segment by the accumulated audio duration so concatenated
segments stay synchronized. Non-English pipelines fall back to proportional
distribution.
"""

from __future__ import annotations

import logging

import numpy as np

from app.tts.base import TTSError, TTSProvider, SynthesisResult
from app.timing.words import Word, distribute_word_timings

logger = logging.getLogger(__name__)

# Kokoro lang_code -> pipeline code (voice prefix must match the language).
LANG_CODE_MAP = {
    "en": "a", "en-us": "a", "en-gb": "b",
    "es": "e", "fr": "f", "hi": "h", "it": "i",
    "ja": "j", "pt": "p", "zh": "z",
}

SAMPLE_RATE = 24000


class KokoroTTS(TTSProvider):
    name = "kokoro"

    def __init__(self) -> None:
        self._pipelines: dict[str, object] = {}

    def _pipeline(self, language: str):
        code = LANG_CODE_MAP.get(language.lower(), "a")
        if code in self._pipelines:
            return self._pipelines[code]
        try:
            from kokoro import KPipeline  # heavy import: torch + model
        except ImportError as exc:
            raise TTSError(
                "Kokoro is not installed. Run scripts/install.sh or "
                "`pip install kokoro soundfile`."
            ) from exc
        logger.info("Loading Kokoro pipeline lang=%s", code)
        try:
            import torch

            device = "cuda" if torch.cuda.is_available() else "cpu"
        except ImportError:  # pragma: no cover - torch ships with kokoro
            device = "cpu"
        pipeline = KPipeline(lang_code=code, repo_id="hexgrad/Kokoro-82M", device=device)
        self._pipelines[code] = pipeline
        return pipeline

    def synthesize(self, text: str, voice: str, speed: float = 1.0,
                   language: str = "en") -> SynthesisResult:
        if not text.strip():
            raise TTSError("Cannot synthesize empty text")
        pipeline = self._pipeline(language)
        segments: list[np.ndarray] = []
        words: list[Word] = []
        offset = 0.0

        try:
            for result in pipeline(text, voice=voice, speed=speed):
                audio = result.audio
                if audio is None:
                    continue
                samples = audio.detach().cpu().numpy().astype(np.float32)
                segment_duration = len(samples) / SAMPLE_RATE
                tokens = getattr(result, "tokens", None) or []
                for token in tokens:
                    token_text = (token.text or "").strip()
                    start = getattr(token, "start_ts", None)
                    end = getattr(token, "end_ts", None)
                    if not token_text or start is None or end is None or end <= start:
                        continue
                    words.append(Word(
                        index=len(words),
                        text=token_text,
                        start=round(offset + float(start), 4),
                        end=round(offset + float(end), 4),
                    ))
                segments.append(samples)
                offset += segment_duration
        except Exception as exc:  # noqa: BLE001 - normalize engine failures
            raise TTSError(f"Kokoro synthesis failed: {exc}") from exc

        if not segments:
            raise TTSError("Kokoro produced no audio for the given text")

        samples = np.concatenate(segments)
        duration = len(samples) / SAMPLE_RATE
        if not words:
            logger.warning("No native timestamps; distributing proportionally")
            words = distribute_word_timings(text, duration)

        return SynthesisResult(samples=samples, sample_rate=SAMPLE_RATE, words=words)


PROVIDER = KokoroTTS()
