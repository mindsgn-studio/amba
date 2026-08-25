"""Prefetch / warm up the configured TTS engine.

Used by scripts/install.sh to download models ahead of first use. Failure is
non-fatal: the model downloads automatically on the worker's first synthesis.
"""

from __future__ import annotations

import logging
import time


def main() -> int:
    from app.config import settings, setup_logging

    setup_logging()
    log = logging.getLogger("tts.prefetch")

    if settings.tts_provider != "kokoro":
        log.info("Provider %s needs no model download", settings.tts_provider)
        return 0

    from app.tts.base import get_provider

    started = time.time()
    provider = get_provider("kokoro")
    result = provider.synthesize(
        "Kokoro model warmup complete.", voice=settings.tts_voice,
        speed=settings.tts_speed, language="en",
    )
    log.info("Warmup OK duration=%.2fs elapsed=%.1fs",
             result.duration, time.time() - started)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
