from pathlib import Path
from unittest.mock import patch

from app.api.services.book_service import create_book_from_text
from app.database import repository as repo
from app.database.database import SessionLocal, init_db
from app.timing.words import TimingDocument, validate_words
from tests.conftest import drain_jobs


def test_full_pipeline_string_to_audio(tmp_path=None):
    with SessionLocal() as db:
        book, job = create_book_from_text(
            db,
            title="Pipeline Test",
            text=" ".join(
                "Sentence number %d talks about the quick brown fox and the "
                "lazy dog resting under a tree." % i for i in range(40)
            ),
            language="en",
            voice="af_heart",
            speed=1.0,
        )
        processed = drain_jobs()
        assert processed >= 1

        db.expire_all()  # worker ran in other sessions; drop cached state
        finished_book = repo.get_book(db, book.id)
        finished_job = repo.get_job(db, job.id)
        assert finished_book.status == "completed"
        assert finished_job.status == "completed"
        assert finished_job.progress == 100

        tree = repo.chunks_for_book(db, book.id)
        assert len(tree) == 1  # single chapter
        chunks = tree[0][1]
        assert len(chunks) >= 1
        for chunk in chunks:
            assert chunk.status == "completed"
            audio_path = Path(chunk.audio_path)
            timing_path = Path(chunk.timing_path)
            assert audio_path.is_file() and audio_path.stat().st_size > 44
            assert timing_path.is_file()

            timing = TimingDocument.load(timing_path)
            assert len(timing.words) > 0
            timing.validate()
            assert timing.duration > 0
            assert timing.text == Path(chunk.text_path).read_text(encoding="utf-8")


def test_restart_safety_skips_completed_chunks():
    """Completed chunks are never regenerated on a second pass."""
    from app.worker.processor import JobProcessor

    drain_jobs()  # clear any leftovers so our job is deterministic

    with SessionLocal() as db:
        book, job = create_book_from_text(
            db, title="Resume Test",
            text="First sentence here. Second sentence follows. Third one ends it.",
            language="en", voice="af_heart", speed=1.0,
        )
        drain_jobs()
        db.expire_all()
        finished = repo.get_job(db, job.id)
        assert finished.status == "completed"
        tree = repo.chunks_for_book(db, book.id)
        completed_paths = [
            (c.audio_path, c.timing_path) for _, chunks in tree for c in chunks
        ]
        assert all(Path(a).is_file() for a, _ in completed_paths)

        # Simulate a crash: reset statuses but keep files, then re-run.
        repo.update_book_status(db, book.id, "processing")
        stale_job = repo.create_job(db, book_id=book.id)

        with patch.object(JobProcessor, "_generate_audio", autospec=True) as spy:
            def fake_generate(self_inner, db_inner, book_inner, job_inner):
                # Ensure every chunk is still 'completed' (nothing regenerated).
                total = 0
                for _, chunks in repo.chunks_for_book(db_inner, book_inner.id):
                    for c in chunks:
                        assert c.status == "completed"
                        total += 1
                return total
            spy.side_effect = fake_generate
            JobProcessor().process(db, repo.get_job(db, stale_job.id))

        assert repo.get_job(db, stale_job.id).status == "completed"


def test_worker_requeues_stale_processing_jobs_on_startup():
    from app.worker.jobs import recover_stale_jobs

    with SessionLocal() as db:
        book, _job = create_book_from_text(
            db, title="Stale Test", text="Hello there. This is a test.",
            language="en", voice="af_heart", speed=1.0,
        )
        orphan = repo.create_job(db, book_id=book.id)
        from sqlalchemy import update

        from app.database.models import Job

        db.execute(update(Job).where(Job.id == orphan.id).values(status="processing"))
        db.commit()

    assert recover_stale_jobs() >= 1

    with SessionLocal() as db:
        assert repo.get_job(db, orphan.id).status == "queued"
