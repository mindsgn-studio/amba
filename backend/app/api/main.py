"""FastAPI application entrypoint: `uvicorn app.api.main:app`."""

from __future__ import annotations

from contextlib import asynccontextmanager

from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware

from app.api.models.schemas import HealthResponse
from app.config import ensure_directories, settings, setup_logging
from app.database.database import init_db

setup_logging()

API_VERSION = "0.1.0"


@asynccontextmanager
async def lifespan(_: FastAPI):
    ensure_directories()
    init_db()
    yield


def create_app() -> FastAPI:
    app = FastAPI(
        title="Read-Along Audiobook API",
        description="Convert text/PDF/EPUB inputs into audiobooks with "
                    "word-level timing for read-along playback.",
        version=API_VERSION,
        lifespan=lifespan,
    )
    if settings.cors_origins:
        app.add_middleware(
            CORSMiddleware,
            allow_origins=settings.cors_origins,
            allow_credentials=False,
            allow_methods=["*"],
            allow_headers=["*"],
        )

    from app.api.routes import books, files, jobs

    app.include_router(books.router, prefix="/api")
    app.include_router(jobs.router, prefix="/api")
    app.include_router(files.router, prefix="/api")

    @app.get("/health", response_model=HealthResponse, tags=["health"])
    def health() -> HealthResponse:
        return HealthResponse(status="ok", version=API_VERSION)

    return app


app = create_app()
