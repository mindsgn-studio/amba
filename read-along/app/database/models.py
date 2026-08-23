"""SQLAlchemy ORM models matching the plan's SQLite schema."""

from __future__ import annotations

from datetime import datetime, timezone

from sqlalchemy import Float, ForeignKey, Integer, String, Text
from sqlalchemy.orm import DeclarativeBase, Mapped, mapped_column, relationship


def utc_now() -> str:
    return datetime.now(timezone.utc).isoformat(timespec="seconds")


class Base(DeclarativeBase):
    pass


class BookStatus:
    QUEUED = "queued"
    PROCESSING = "processing"
    COMPLETED = "completed"
    FAILED = "failed"


class ChunkStatus:
    PENDING = "pending"
    PROCESSING = "processing"
    COMPLETED = "completed"
    FAILED = "failed"


class JobStatus:
    QUEUED = "queued"
    PROCESSING = "processing"
    COMPLETED = "completed"
    FAILED = "failed"


class Book(Base):
    __tablename__ = "books"

    id: Mapped[str] = mapped_column(String, primary_key=True)
    title: Mapped[str] = mapped_column(String, nullable=False)
    author: Mapped[str | None] = mapped_column(String)
    language: Mapped[str | None] = mapped_column(String)
    source_type: Mapped[str] = mapped_column(String, nullable=False)  # text|pdf|epub
    source_path: Mapped[str | None] = mapped_column(String)
    voice: Mapped[str | None] = mapped_column(String)
    speed: Mapped[float] = mapped_column(Float, default=1.0)
    status: Mapped[str] = mapped_column(String, nullable=False, default=BookStatus.QUEUED)
    created_at: Mapped[str] = mapped_column(Text, nullable=False, default=utc_now)
    updated_at: Mapped[str] = mapped_column(Text, nullable=False, default=utc_now)

    chapters: Mapped[list["Chapter"]] = relationship(  # noqa: F821
        back_populates="book", cascade="all, delete-orphan",
        order_by="Chapter.chapter_index", lazy="selectin",
    )


class Chapter(Base):
    __tablename__ = "chapters"

    id: Mapped[str] = mapped_column(String, primary_key=True)
    book_id: Mapped[str] = mapped_column(ForeignKey("books.id"), nullable=False)
    chapter_index: Mapped[int] = mapped_column(Integer, nullable=False)
    title: Mapped[str | None] = mapped_column(String)
    status: Mapped[str] = mapped_column(String, nullable=False, default="pending")
    text_path: Mapped[str | None] = mapped_column(String)

    book: Mapped[Book] = relationship(back_populates="chapters")
    chunks: Mapped[list["Chunk"]] = relationship(
        back_populates="chapter", cascade="all, delete-orphan",
        order_by="Chunk.chunk_index", lazy="selectin",
    )


class Chunk(Base):
    __tablename__ = "chunks"

    id: Mapped[str] = mapped_column(String, primary_key=True)
    chapter_id: Mapped[str] = mapped_column(ForeignKey("chapters.id"), nullable=False)
    chunk_index: Mapped[int] = mapped_column(Integer, nullable=False)
    text_path: Mapped[str | None] = mapped_column(String)
    audio_path: Mapped[str | None] = mapped_column(String)
    timing_path: Mapped[str | None] = mapped_column(String)
    duration: Mapped[float | None] = mapped_column(Float)
    word_count: Mapped[int | None] = mapped_column(Integer)
    status: Mapped[str] = mapped_column(String, nullable=False, default="pending")

    chapter: Mapped[Chapter] = relationship(back_populates="chunks")


class Job(Base):
    __tablename__ = "jobs"

    id: Mapped[str] = mapped_column(String, primary_key=True)
    type: Mapped[str] = mapped_column(String, nullable=False, default="generate_book")
    book_id: Mapped[str | None] = mapped_column(String)
    status: Mapped[str] = mapped_column(String, nullable=False, default="queued")
    progress: Mapped[int] = mapped_column(Integer, default=0)
    error: Mapped[str | None] = mapped_column(Text)
    created_at: Mapped[str] = mapped_column(Text, nullable=False, default=utc_now)
    started_at: Mapped[str | None] = mapped_column(Text)
    completed_at: Mapped[str | None] = mapped_column(Text)
