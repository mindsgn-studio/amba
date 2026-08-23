"""Repository: all database access lives here (API + worker share it)."""

from __future__ import annotations

import logging
import uuid
from datetime import datetime, timezone

from sqlalchemy import delete, func, select, update
from sqlalchemy.orm import Session

from app.database.models import (
    Book,
    BookStatus,
    Chapter,
    Chunk,
    ChunkStatus,
    Job,
    JobStatus,
    utc_now,
)

logger = logging.getLogger(__name__)


def new_id() -> str:
    return uuid.uuid4().hex[:12]


# ---------------------------------------------------------------- books ----

def create_book(db: Session, *, title: str, source_type: str, source_path: str | None,
                author: str | None = None, language: str | None = "en",
                voice: str | None = None, speed: float = 1.0) -> Book:
    book = Book(
        id=new_id(), title=title, author=author, language=language,
        source_type=source_type, source_path=source_path, voice=voice,
        speed=speed, status=BookStatus.QUEUED, created_at=utc_now(), updated_at=utc_now(),
    )
    db.add(book)
    db.commit()
    return book


def get_book(db: Session, book_id: str) -> Book | None:
    return db.get(Book, book_id)


def list_books(db: Session) -> list[Book]:
    stmt = select(Book).order_by(Book.created_at.desc())
    return list(db.scalars(stmt))


def update_book_status(db: Session, book_id: str, status: str) -> None:
    db.execute(
        update(Book).where(Book.id == book_id)
        .values(status=status, updated_at=utc_now())
    )
    db.commit()


def delete_book(db: Session, book_id: str) -> bool:
    book = db.get(Book, book_id)
    if book is None:
        return False
    # Cascades remove chapters and chunks.
    db.delete(book)
    db.execute(delete(Job).where(Job.book_id == book_id))
    db.commit()
    return True


# -------------------------------------------------------------- chapters ----

def create_chapter(db: Session, *, book_id: str, chapter_index: int, title: str | None,
                   text_path: str | None = None) -> Chapter:
    chapter = Chapter(
        id=f"{book_id}_ch{chapter_index + 1:03d}", book_id=book_id,
        chapter_index=chapter_index, title=title, text_path=text_path,
        status="pending",
    )
    db.add(chapter)
    db.commit()
    return chapter


def create_chunk(db: Session, *, chapter_id: str, chunk_index: int,
                 text_path: str | None = None, word_count: int | None = None) -> Chunk:
    chunk = Chunk(
        id=f"{chapter_id}_s{chunk_index + 1:03d}", chapter_id=chapter_id,
        chunk_index=chunk_index, text_path=text_path, word_count=word_count,
        status=ChunkStatus.PENDING,
    )
    db.add(chunk)
    db.commit()
    return chunk


def update_chunk_result(db: Session, chunk_id: str, *, audio_path: str | None = None,
                        timing_path: str | None = None, duration: float | None = None,
                        status: str) -> None:
    values: dict = {"status": status}
    if audio_path is not None:
        values["audio_path"] = audio_path
    if timing_path is not None:
        values["timing_path"] = timing_path
    if duration is not None:
        values["duration"] = duration
    db.execute(update(Chunk).where(Chunk.id == chunk_id).values(**values))
    db.commit()


def chunks_for_book(db: Session, book_id: str) -> list[tuple[Chapter, list[Chunk]]]:
    """Return [(chapter, [chunks...])] ordered by chapter/chunk index."""
    chapters = list(db.scalars(
        select(Chapter).where(Chapter.book_id == book_id).order_by(Chapter.chapter_index)
    ))
    result: list[tuple[Chapter, list[Chunk]]] = []
    for chapter in chapters:
        chunks = list(db.scalars(
            select(Chunk).where(Chunk.chapter_id == chapter.id)
            .order_by(Chunk.chunk_index)
        ))
        result.append((chapter, chunks))
    return result


def chunk_counts(db: Session, book_id: str) -> tuple[int, int]:
    """(total_chunks, completed_chunks) for a book."""
    total = db.scalar(
        select(func.count(Chunk.id))
        .select_from(Chunk)
        .join(Chapter, Chunk.chapter_id == Chapter.id)
        .where(Chapter.book_id == book_id)
    ) or 0
    completed = db.scalar(
        select(func.count(Chunk.id))
        .select_from(Chunk)
        .join(Chapter, Chunk.chapter_id == Chapter.id)
        .where(Chapter.book_id == book_id, Chunk.status == ChunkStatus.COMPLETED)
    ) or 0
    return int(total), int(completed)


# ------------------------------------------------------------------ jobs ----

def create_job(db: Session, *, book_id: str, job_type: str = "generate_book") -> Job:
    job = Job(id=new_id(), type=job_type, book_id=book_id,
              status=JobStatus.QUEUED, progress=0, created_at=utc_now())
    db.add(job)
    db.commit()
    return job


def get_job(db: Session, job_id: str) -> Job | None:
    return db.get(Job, job_id)


def claim_next_job(db: Session) -> Job | None:
    """Atomically claim the oldest queued job (safe against double-claiming).

    Returns the job with status flipped to `processing`, or None.
    """
    job_id = db.scalar(
        select(Job.id).where(Job.status == JobStatus.QUEUED)
        .order_by(Job.created_at).limit(1)
    )
    if job_id is None:
        return None
    claimed = db.execute(
        update(Job)
        .where(Job.id == job_id, Job.status == JobStatus.QUEUED)
        .values(status=JobStatus.PROCESSING, started_at=utc_now())
    )
    db.commit()
    if claimed.rowcount != 1:  # raced with another worker; retry next tick
        return None
    return db.get(Job, job_id)


def update_job_progress(db: Session, job_id: str, progress: int) -> None:
    db.execute(update(Job).where(Job.id == job_id)
               .values(progress=max(0, min(int(progress), 100))))
    db.commit()


def complete_job(db: Session, job_id: str) -> None:
    now = datetime.now(timezone.utc).isoformat(timespec="seconds")
    db.execute(update(Job).where(Job.id == job_id).values(
        status=JobStatus.COMPLETED, progress=100, completed_at=now))
    db.commit()


def fail_job(db: Session, job_id: str, error: str) -> None:
    now = datetime.now(timezone.utc).isoformat(timespec="seconds")
    db.execute(update(Job).where(Job.id == job_id).values(
        status=JobStatus.FAILED, error=error[:2000], completed_at=now))
    db.commit()


def reset_stale_processing_jobs(db: Session) -> int:
    """Restart-safety: jobs stuck in `processing` (crashed worker) requeue."""
    result = db.execute(
        update(Job).where(Job.status == JobStatus.PROCESSING)
        .values(status=JobStatus.QUEUED, started_at=None)
    )
    db.commit()
    count = result.rowcount or 0
    if count:
        logger.warning("Requeued stale processing jobs count=%d", count)
    return count
