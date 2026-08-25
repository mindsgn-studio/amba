"""Shared pytest configuration.

Points the app at a temporary data dir / database and the `fake` TTS engine
before any app module is imported, so tests run offline and fast.
"""

from __future__ import annotations

import os
import tempfile

_TMP = tempfile.mkdtemp(prefix="readalong-tests-")
os.environ.setdefault("DATA_DIR", os.path.join(_TMP, "data"))
os.environ.setdefault("DATABASE_URL", f"sqlite:///{_TMP}/test.db")
os.environ.setdefault("TTS_PROVIDER", "fake")
os.environ.setdefault("AUDIO_FORMAT", "wav")  # tests don't need ffmpeg
os.environ.setdefault("LOG_LEVEL", "WARNING")

import pytest  # noqa: E402
from fastapi.testclient import TestClient  # noqa: E402


def drain_jobs(max_iterations: int = 200) -> int:
    """Process every queued job (tests share one DB, so drain everything)."""
    from sqlalchemy import func, select

    from app.database.database import SessionLocal
    from app.database.models import Job
    from app.worker.jobs import run_pending_jobs

    processed = 0
    for _ in range(max_iterations):
        with SessionLocal() as db:
            pending = db.scalar(select(func.count()).select_from(Job)
                                .where(Job.status == "queued")) or 0
        if not pending:
            break
        processed += run_pending_jobs(max_jobs=int(pending))
    return processed


@pytest.fixture(scope="session", autouse=True)
def _database():
    from app.database.database import init_db

    init_db()


@pytest.fixture(scope="session")
def client():
    from app.api.main import create_app

    app = create_app()
    with TestClient(app) as test_client:
        yield test_client
