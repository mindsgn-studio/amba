"""Job processor: document -> chapters/chunks -> TTS + timings.

Restart-safe: extraction is skipped when chapters already exist and
completed chunks are never regenerated.
"""

from __future__ import annotations

import logging
from pathlib import Path

from sqlalchemy.orm import Session

from app.api.services.book_service import chapter_dir, relative_to_data
from app.config import settings
from app.database import repository as repo
from app.database.models import Book, ChunkStatus, Job
from app.documents.base import Document, UnsupportedDocumentError, parse_document
from app.media.converter import finalize_audio
from app.processing.chunker import chunk_paragraphs
from app.processing.normalizer import normalize_paragraphs
from app.tts.base import SynthesisResult, TTSError, get_provider
from app.timing.words import TimingDocument

logger = logging.getLogger(__name__)


class JobProcessor:
    def __init__(self) -> None:
        self.provider = get_provider(settings.tts_provider)

    # ------------------------------------------------------------- entry --

    def process(self, db: Session, job: Job) -> None:
        book = repo.get_book(db, job.book_id or "")
        if book is None:
            repo.fail_job(db, job.id, f"Book not found: {job.book_id}")
            return

        logger.info("Job started job=%s book=%s", job.id, book.id)
        try:
            repo.update_book_status(db, book.id, "processing")
            total_chunks = self._ensure_structure(db, book)
            if total_chunks == 0:
                raise UnsupportedDocumentError("Document produced no chunks")

            self._generate_audio(db, book, job)
            repo.update_book_status(db, book.id, "completed")
            repo.complete_job(db, job.id)
            logger.info("Job completed job=%s book=%s", job.id, book.id)
        except Exception as exc:  # noqa: BLE001 - any failure fails the job
            message = f"{type(exc).__name__}: {exc}"
            logger.exception("Job failed job=%s error=%s", job.id, message)
            repo.update_book_status(db, book.id, "failed")
            repo.fail_job(db, job.id, message)

    # ------------------------------------------------------- structure ----

    def _ensure_structure(self, db: Session, book: Book) -> int:
        """Create chapters/chunks from the source document (idempotent).

        If the book already has chapters (previous run crashed mid-audio),
        reuse them so completed chunks are preserved.
        """
        existing = repo.chunks_for_book(db, book.id)
        if existing:
            return sum(len(chunks) for _, chunks in existing)

        logger.info("Extracting document source_type=%s book=%s",
                    book.source_type, book.id)
        document = parse_document(
            source_type=book.source_type,
            path=Path(book.source_path) if book.source_path else None,
            title_hint=book.title,
        )
        return self._persist_structure(db, book, document)

    def _persist_structure(self, db: Session, book: Book, document: Document) -> int:
        total = 0
        for chapter_index, chapter in enumerate(document.chapters):
            paragraphs = normalize_paragraphs(chapter.paragraphs)
            if not paragraphs:
                continue
            text_dir = chapter_dir(book.id, chapter_index)
            text_dir.mkdir(parents=True, exist_ok=True)
            chapter_text_path = text_dir / "text.txt"
            chapter_text_path.write_text("\n\n".join(paragraphs), encoding="utf-8")

            db_chapter = repo.create_chapter(
                db, book_id=book.id, chapter_index=chapter_index,
                title=chapter.title,
                text_path=str(chapter_text_path),
            )
            for chunk in chunk_paragraphs(
                paragraphs,
                target_words=settings.chunk_target_words,
                max_words=settings.chunk_max_words,
            ):
                chunk_text_path = text_dir / f"chunk_{chunk.index + 1:03d}.txt"
                chunk_text_path.write_text(chunk.text, encoding="utf-8")
                repo.create_chunk(
                    db, chapter_id=db_chapter.id, chunk_index=chunk.index,
                    text_path=str(chunk_text_path), word_count=chunk.word_count,
                )
                total += 1
        logger.info("Chapters created chunks=%d book=%s", total, book.id)
        return total

    # ------------------------------------------------------------- audio --

    def _generate_audio(self, db: Session, book: Book, job: Job) -> None:
        voice = book.voice or settings.tts_voice
        speed = book.speed or 1.0
        language = book.language or "en"
        tree = repo.chunks_for_book(db, book.id)
        total = sum(len(chunks) for _, chunks in tree)
        done_before = 0

        for chapter, chunks in tree:
            out_dir = chapter_dir(book.id, chapter.chapter_index)
            for chunk in chunks:
                done_before += 1
                progress = int(done_before / max(total, 1) * 100)
                repo.update_job_progress(db, job.id, progress)

                if chunk.status == ChunkStatus.COMPLETED and self._files_exist(chunk):
                    logger.info("Chunk already complete, skipping chunk=%d/%d id=%s",
                                done_before, total, chunk.id)
                    continue

                text_path = Path(chunk.text_path or "")
                text = text_path.read_text(encoding="utf-8") if text_path.exists() else ""
                if not text.strip():
                    repo.update_chunk_result(db, chunk.id, status=ChunkStatus.FAILED)
                    raise TTSError(f"Chunk {chunk.id} has no text")

                stem = f"chunk_{chunk.chunk_index + 1:03d}"
                logger.info("Generating audio chunk=%d/%d words=%d voice=%s",
                            done_before, total, len(text.split()), voice)

                result: SynthesisResult = self.provider.synthesize(
                    text, voice=voice, speed=speed, language=language
                )

                wav_path = out_dir / f"{stem}.wav"
                wav_path.parent.mkdir(parents=True, exist_ok=True)
                _write_wav(wav_path, result.samples, result.sample_rate)
                final_path, _ext = finalize_audio(wav_path, out_dir, stem,
                                                  settings.audio_format)

                timing = TimingDocument(chunk_id=chunk.id, text=text,
                                        words=result.words,
                                        duration=result.duration)
                timing.validate()
                timing_path = out_dir / f"{stem}.json"
                timing.save(timing_path)

                repo.update_chunk_result(
                    db, chunk.id,
                    audio_path=str(final_path),
                    timing_path=str(timing_path),
                    duration=result.duration,
                    status=ChunkStatus.COMPLETED,
                )
                logger.info("Chunk complete duration=%.2fs chunk=%d/%d",
                            result.duration, done_before, total)

        repo.update_job_progress(db, job.id, 100)

    @staticmethod
    def _files_exist(chunk) -> bool:
        return bool(
            chunk.audio_path and Path(chunk.audio_path).is_file()
            and chunk.timing_path and Path(chunk.timing_path).is_file()
        )


def _write_wav(path: Path, samples, sample_rate: int) -> None:
    import soundfile as sf

    path.parent.mkdir(parents=True, exist_ok=True)
    sf.write(str(path), samples, sample_rate, subtype="PCM_16")


def process_claimed_job(db: Session, job: Job) -> None:
    """Module-level convenience used by worker loop and tests."""
    JobProcessor().process(db, job)


__all__ = ["JobProcessor", "process_claimed_job", "relative_to_data"]
