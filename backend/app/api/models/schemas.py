"""Pydantic request/response schemas for the API."""

from __future__ import annotations

from pydantic import BaseModel, Field


class BookCreateRequest(BaseModel):
    title: str = Field(min_length=1, max_length=300)
    text: str = Field(min_length=1)
    language: str = "en"
    voice: str | None = None
    speed: float = 1.0


class CreatedResponse(BaseModel):
    book_id: str
    job_id: str
    status: str = "queued"


class ChunkOut(BaseModel):
    id: str
    chunk_index: int
    duration: float | None = None
    audio_url: str | None = None
    timing_url: str | None = None


class ChapterOut(BaseModel):
    id: str
    chapter_index: int
    title: str | None = None
    chunks: list[ChunkOut] = []


class BookSummary(BaseModel):
    id: str
    title: str
    author: str | None = None
    language: str | None = None
    source_type: str
    status: str
    created_at: str
    updated_at: str


class BookDetail(BookSummary):
    chapters: list[ChapterOut] = []


class BookListResponse(BaseModel):
    books: list[BookSummary]


class JobResponse(BaseModel):
    id: str
    type: str
    book_id: str | None
    status: str
    progress: int
    error: str | None = None
    created_at: str
    started_at: str | None = None
    completed_at: str | None = None


class DeleteResponse(BaseModel):
    deleted: bool
    id: str


class HealthResponse(BaseModel):
    status: str = "ok"
    version: str
