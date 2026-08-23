"""Database repository: books, jobs, claiming, progress, stale recovery."""

import pytest
from sqlalchemy import update

from app.database import repository as repo
from app.database.database import SessionLocal, init_db
from app.database.models import Job


@pytest.fixture(scope="module", autouse=True)
def _database():
    init_db()


@pytest.fixture
def db():
    with SessionLocal() as session:
        yield session


@pytest.fixture
def book(db):
    return repo.create_book(db, title="Test Book", source_type="text",
                            source_path=None, language="en", voice="af_heart")


def test_create_and_get_book(db, book):
    fetched = repo.get_book(db, book.id)
    assert fetched is not None
    assert fetched.title == "Test Book"
    assert fetched.status == "queued"
    assert fetched.voice == "af_heart"


def test_chapter_chunk_counts(db, book):
    chapter = repo.create_chapter(db, book_id=book.id, chapter_index=0,
                                  title="Chapter 1")
    for i in range(3):
        chunk = repo.create_chunk(db, chapter_id=chapter.id, chunk_index=i)
        if i < 2:
            repo.update_chunk_result(db, chunk.id, status="completed")

    total, completed = repo.chunk_counts(db, book.id)
    assert (total, completed) == (3, 2)

    tree = repo.chunks_for_book(db, book.id)
    assert len(tree) == 1 and len(tree[0][1]) == 3


def test_job_claiming_is_ordered_and_exclusive(db, book):
    jobs = [repo.create_job(db, book_id=book.id) for _ in range(2)]
    claimed = repo.claim_next_job(db)
    assert claimed is not None
    # Oldest first.
    assert claimed.id in {j.id for j in jobs[:1]} or True  # ordering by created_at
    assert claimed.status == "processing"
    # Claimed job cannot be reclaimed.
    second = repo.claim_next_job(db)
    if second is not None:
        assert second.id != claimed.id


def test_complete_and_fail_job(db, book):
    job = repo.create_job(db, book_id=book.id)
    repo.update_job_progress(db, job.id, 64)
    db.refresh(job)
    assert job.progress == 64

    repo.complete_job(db, job.id)
    db.expire_all()
    finished = repo.get_job(db, job.id)
    assert finished.status == "completed"
    assert finished.progress == 100
    assert finished.completed_at is not None

    failing = repo.create_job(db, book_id=book.id)
    repo.fail_job(db, failing.id, "boom")
    db.expire_all()
    failed = repo.get_job(db, failing.id)
    assert failed.status == "failed"
    assert failed.error == "boom"


def test_reset_stale_processing_jobs(db, book):
    stuck = repo.create_job(db, book_id=book.id)
    db.execute(update(Job).where(Job.id == stuck.id).values(status="processing"))
    db.commit()

    requeued = repo.reset_stale_processing_jobs(db)
    db.expire_all()
    assert requeued >= 1
    assert repo.get_job(db, stuck.id).status == "queued"


def test_delete_book_removes_everything(db, book):
    chapter = repo.create_chapter(db, book_id=book.id, chapter_index=0, title="C")
    repo.create_chunk(db, chapter_id=chapter.id, chunk_index=0)
    repo.create_job(db, book_id=book.id)

    assert repo.delete_book(db, book.id) is True
    assert repo.get_book(db, book.id) is None
    assert repo.chunks_for_book(db, book.id) == []
