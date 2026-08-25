"""TTS provider abstraction and fake engine behavior."""

import numpy as np
import pytest

from app.tts.base import TTSError, get_provider
from app.timing.words import validate_words


def test_fake_provider_produces_audio_and_timings():
    provider = get_provider("fake")
    assert provider.name == "fake"
    result = provider.synthesize("Hello world from the fake engine.", voice="af_heart")
    assert result.sample_rate > 0
    assert result.samples.dtype == np.float32
    assert result.duration > 0.5
    assert len(result.words) == 6
    validate_words(result.words)
    assert result.words[-1].end == pytest.approx(result.duration, abs=0.05)


def test_unknown_provider_raises():
    with pytest.raises(TTSError):
        get_provider("does-not-exist")


def test_kokoro_module_imports_without_model_load():
    """Import must not download/load the model (lazy pipeline init)."""
    import app.tts.kokoro as kokoro_module

    assert kokoro_module.PROVIDER.name == "kokoro"
    assert "en" in kokoro_module.LANG_CODE_MAP


@pytest.mark.integration
def test_real_kokoro_synthesis():
    """Optional: requires `kokoro` installed + model downloaded.

    Run explicitly with:  pytest -m integration -k kokoro
    """
    pytest.importorskip("kokoro", reason="kokoro package not installed")
    from app.tts.kokoro import PROVIDER

    result = PROVIDER.synthesize(
        "The quick brown fox jumps over the lazy dog.",
        voice="af_heart",
    )
    assert result.duration > 1.0
    words = result.words
    assert [w.text.lower() for w in words][:3] == ["the", "quick", "brown"]
    starts = [w.start for w in words]
    ends = [w.end for w in words]
    assert starts == sorted(starts), "word timings must be ordered"
    assert all(e > s for s, e in zip(starts, ends))
