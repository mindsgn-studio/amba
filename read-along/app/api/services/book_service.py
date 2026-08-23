"""Book creation service.

The API never generates audio: it stores the source, creates the book row and
a queued job, then returns immediately.
"""

from __future__ import annotations

import logging
import shutil
from pathlib import Path
from sqlalchemy.orm import Session

from app.config import settings
from app.database import repository as repo
from app.database.models import Book, Chunk, Job

logger = logging.getLogger(__name__)


def book_dir(book_id: str) -> Path:
    return settings.books_dir / book_id


def source_dir(book_id: str) -> Path:
    return book_dir(book_id) / "source"


def chapter_dir(book_id: str, chapter_index: int) -> Path:
    return book_dir(book_id) / "chapters" / f"{chapter_index + 1:03d}"


def relative_to_data(path: Path) -> str:
    """Path relative to DATA_DIR using forward slashes (for /api/files URLs)."""
    try:
        rel = path.resolve().relative_to(settings.resolve_data_dir().resolve())
        return rel.as_posix()
    except ValueError:
        return path.name


def create_book_from_text(db: Session, *, title: str, text: str,
                          language: str, voice: str | None,
                          speed: float) -> tuple[Book, Job]:
    """Raw string input: persist it like a .txt source, then queue the job."""
    book = repo.create_book(
        db, title=title, source_type="text", source_path=None,
        language=language or "en", voice=voice or settings.tts_voice,
        speed=speed if speed and speed > 0 else 1.0,
    )
    src = source_dir(book.id)
    src.mkdir(parents=True, exist_ok=True)
    source_file = src / "input.txt"
    source_file.write_text(text, encoding="utf-8")
    book.source_path = str(source_file)
    db.commit()

    job = repo.create_job(db, book_id=book.id)
    logger.info("Created book from string book=%s job=%s chars=%d",
                book.id, job.id, len(text))
    return book, job


def create_book_from_upload(db: Session, *, filename: str, content: bytes,
                            title: str | None, language: str | None,
                            voice: str | None) -> tuple[Book, Job]:
    """File input (.pdf/.epub/.txt): save file, create records, return."""
    from app.documents.base import detect_source_type

    source_type = detect_source_type(filename)
    book_title = (title or "").strip() or Path(filename).stem.replace("_", " ").title() or "Untitled"

    book = repo.create_book(
        db, title=book_title, source_type=source_type, source_path=None,
        language=language or "en", voice=voice or settings.tts_voice, speed=1.0,
    )
    src = source_dir(book.id)
    src.mkdir(parents=True, exist_ok=True)
    safe_name = Path(filename).name  # no traversal via crafted filenames
    dest = src / safe_name
    dest.write_bytes(content)
    book.source_path = str(dest)
    db.commit()

    job = repo.create_job(db, book_id=book.id)
    logger.info("Created book from upload book=%s job=%s file=%s bytes=%d",
                book.id, job.id, safe_name, len(content))
    return book, job


def delete_book_and_files(db: Session, book_id: str) -> bool:
    deleted = repo.delete_book(db, book_id)
    if not deleted:
        return False
    directory = book_dir(book_id)
    if directory.exists():
        shutil.rmtree(directory, ignore_errors=True)
    logger.info("Deleted book %s and its files", book_id)
    return True


def serialize_book_summary(book: Book):
    from app.api.models.schemas import BookSummary

    return BookSummary(
        id=book.id, title=book.title, author=book.author, language=book.language,
        source_type=book.source_type, status=book.status,
        created_at=book.created_at, updated_at=book.updated_at,
    )


def serialize_chunk(book_id: str, chapter_index: int, chunk: Chunk):
    from app.api.models.schemas import ChunkOut

    def url(rel_path: str | None) -> str | None:
        if not rel_path:
            return None
        return "/api/files/" + relative_to_data(Path(rel_path))

    return ChunkOut(
        id=chunk.id,
        chunk_index=chunk.chunk_index,
        duration=chunk.duration,
        audio_url=url(chunk.audio_path),
        timing_url=url(chunk.timing_path),
    )


def serialize_book_detail(book: Book):
    from app.api.models.schemas import BookDetail, ChapterOut

    tree = []
    for chapter in sorted(book.chapters, key=lambda c: c.chapter_index):
        tree.append(ChapterOut(
            id=chapter.id,
            chapter_index=chapter.chapter_index,
            title=chapter.title,
            chunks=[
                serialize_chunk(book.id, chapter.chapter_index, chunk)
                for chunk in sorted(chapter.chunks, key=lambda c: c.chunk_index)
            ],
        ))
    detail = BookDetail(
        **serialize_book_summary(book).model_dump(),
        chapters=tree,
    )
    return detail
