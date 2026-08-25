"""Worker entrypoint: `python -m app.worker.main`.

Polls SQLite for queued jobs and processes them one at a time. Restart-safe:
stale `processing` jobs are requeued at startup, completed chunks are skipped.
"""

from __future__ import annotations

import logging
import signal
import sys
import time

from app.config import ensure_directories, settings, setup_logging
from app.database.database import init_db

setup_logging()
logger = logging.getLogger("worker")


def main() -> int:
    ensure_directories()
    init_db()

    from app.worker.jobs import recover_stale_jobs, run_pending_jobs

    requeued = recover_stale_jobs()
    logger.info(
        "Worker started provider=%s poll=%.1fs requeued=%d",
        settings.tts_provider, settings.poll_interval_seconds, requeued,
    )

    running = {"flag": True}

    def _stop(_signum, _frame):
        logger.info("Shutdown signal received; finishing current tick")
        running["flag"] = False

    signal.signal(signal.SIGINT, _stop)
    signal.signal(signal.SIGTERM, _stop)

    try:
        while running["flag"]:
            try:
                processed = run_pending_jobs(max_jobs=1)
            except Exception:  # noqa: BLE001 - keep the loop alive on errors
                logger.exception("Tick failed")
                processed = 0
            if not processed:
                time.sleep(settings.poll_interval_seconds)
    finally:
        logger.info("Worker stopped")
    return 0


if __name__ == "__main__":
    sys.exit(main())
