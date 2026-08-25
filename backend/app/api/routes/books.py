"""Book endpoints."""

from __future__ import annotations

import logging

from fastapi import APIRouter, Depends, File, Form, HTTPException, UploadFile
from sqlalchemy.orm import Session

from app.api.models.schemas import (
    BookCreateRequest,
    BookDetail,
    BookListResponse,
    CreatedResponse,
    DeleteResponse,
)
from app.api.services import book_service
from app.database import repository as repo
from app.database.database import get_db
from app.documents.base import UnsupportedDocumentError

logger = logging.getLogger(__name__)
router = APIRouter(tags=["books"])

MAX_UPLOAD_BYTES = 100 * 1024 * 1024  # 100 MB


@router.post("/books", status_code=202, response_model=CreatedResponse)
def create_book_from_string(body: BookCreateRequest,
                            db: Session = Depends(get_db)) -> CreatedResponse:
    book, job = book_service.create_book_from_text(
        db, title=body.title.strip(), text=body.text,
        language=body.language, voice=body.voice, speed=body.speed,
    )
    return CreatedResponse(book_id=book.id, job_id=job.id, status=job.status)


@router.post("/books/upload", status_code=202, response_model=CreatedResponse)
async def upload_book(file: UploadFile = File(...),
                      title: str | None = Form(None),
                      language: str | None = Form(None),
                      voice: str | None = Form(None),
                      db: Session = Depends(get_db)) -> CreatedResponse:
    filename = file.filename or ""
    try:
        source_type = None
        from app.documents.base import detect_source_type
        source_type = detect_source_type(filename)
    except UnsupportedDocumentError as exc:
        raise HTTPException(status_code=415, detail=str(exc)) from exc

    content = await file.read()
    if len(content) == 0:
        raise HTTPException(status_code=422, detail="Empty file")
    if len(content) > MAX_UPLOAD_BYTES:
        raise HTTPException(status_code=413, detail="File too large (max 100MB)")

    book, job = book_service.create_book_from_upload(
        db, filename=filename, content=content, title=title,
        language=language, voice=voice,
    )
    return CreatedResponse(book_id=book.id, job_id=job.id, status=job.status)


@router.get("/books", response_model=BookListResponse)
def list_books(db: Session = Depends(get_db)) -> BookListResponse:
    return BookListResponse(
        books=[book_service.serialize_book_summary(b) for b in repo.list_books(db)]
    )


@router.get("/books/{book_id}", response_model=BookDetail)
def get_book(book_id: str, db: Session = Depends(get_db)) -> BookDetail:
    book = repo.get_book(db, book_id)
    if book is None:
        raise HTTPException(status_code=404, detail="Book not found")
    return book_service.serialize_book_detail(book)


@router.delete("/books/{book_id}", response_model=DeleteResponse)
def delete_book(book_id: str, db: Session = Depends(get_db)) -> DeleteResponse:
    deleted = book_service.delete_book_and_files(db, book_id)
    if not deleted:
        raise HTTPException(status_code=404, detail="Book not found")
    return DeleteResponse(deleted=True, id=book_id)
