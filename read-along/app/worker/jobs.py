"""Job claiming helpers (worker side)."""

from __future__ import annotations

import logging
from sqlalchemy.orm import Session

from app.database import repository as repo
from app.database.database import SessionLocal
from app.worker.processor import process_claimed_job

logger = logging.getLogger(__name__)


def recover_stale_jobs() -> int:
    """Requeue jobs left in `processing` by a crashed previous run."""
    with SessionLocal() as db:
        return repo.reset_stale_processing_jobs(db)


def run_pending_jobs(max_jobs: int = 1) -> int:
    """Claim and process up to `max_jobs` queued jobs. Returns count done."""
    processed = 0
    while processed < max_jobs:
        with SessionLocal() as db:
            job = repo.claim_next_job(db)
            if job is None:
                break
            logger.info("Claimed job id=%s type=%s book=%s",
                        job.id, job.type, job.book_id)
            try:
                process_claimed_job(db, job)
            except Exception:  # noqa: BLE001 - processor already records failure
                logger.exception("Unhandled error while processing job=%s", job.id)
            processed += 1
    return processed
