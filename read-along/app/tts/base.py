"""TTS provider abstraction.

The worker only talks to `TTSProvider`; Kokoro is one interchangeable
implementation. New engines (Piper, Chatterbox, cloud) plug in by adding a
module and registering it in `_PROVIDER_MODULES`.
"""

from __future__ import annotations

import abc
from dataclasses import dataclass, field

import numpy as np

from app.timing.words import Word


class TTSError(RuntimeError):
    """Raised when speech synthesis fails."""


@dataclass
class SynthesisResult:
    samples: np.ndarray  # float32 mono
    sample_rate: int
    words: list[Word] = field(default_factory=list)

    @property
    def duration(self) -> float:
        if len(self.samples) == 0:
            return 0.0
        return round(len(self.samples) / self.sample_rate, 4)


class TTSProvider(abc.ABC):
    """Interface every engine must implement.

    Implementations must return word timings relative to the start of the
    synthesized audio; if the engine cannot produce them natively, fall back
    to `distribute_word_timings`.
    """

    name: str = "base"

    @abc.abstractmethod
    def synthesize(self, text: str, voice: str, speed: float = 1.0,
                   language: str = "en") -> SynthesisResult:
        ...


# Provider name -> module providing a singleton `PROVIDER`.
_PROVIDER_MODULES: dict[str, str] = {
    "kokoro": "app.tts.kokoro",
    "fake": "app.tts.fake",
}

_instances: dict[str, TTSProvider] = {}


def get_provider(name: str | None = None) -> TTSProvider:
    """Return the provider instance for `name`, importing its module lazily."""
    from app.config import settings

    key = (name or settings.tts_provider).lower()
    if key in _instances:
        return _instances[key]
    module_path = _PROVIDER_MODULES.get(key)
    if module_path is None:
        raise TTSError(
            f"Unknown TTS provider {key!r}. Available: {sorted(_PROVIDER_MODULES)}"
        )
    import importlib

    module = importlib.import_module(module_path)
    provider: TTSProvider = module.PROVIDER  # type: ignore[attr-defined]
    _instances[key] = provider
    return provider
