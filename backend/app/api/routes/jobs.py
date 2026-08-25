"""Job status endpoints."""

from __future__ import annotations

from fastapi import APIRouter, Depends, HTTPException
from sqlalchemy.orm import Session

from app.api.models.schemas import JobResponse
from app.database import repository as repo
from app.database.database import get_db

router = APIRouter(tags=["jobs"])


@router.get("/jobs/{job_id}", response_model=JobResponse)
def get_job(job_id: str, db: Session = Depends(get_db)) -> JobResponse:
    job = repo.get_job(db, job_id)
    if job is None:
        raise HTTPException(status_code=404, detail="Job not found")
    return JobResponse(
        id=job.id, type=job.type, book_id=job.book_id, status=job.status,
        progress=job.progress, error=job.error, created_at=job.created_at,
        started_at=job.started_at, completed_at=job.completed_at,
    )
