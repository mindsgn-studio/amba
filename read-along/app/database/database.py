"""Engine/session setup for SQLite (WAL mode, FK enforcement)."""

from __future__ import annotations

from sqlalchemy import create_engine, event
from sqlalchemy.engine import Engine
from sqlalchemy.orm import Session, sessionmaker

from app.config import ensure_directories, settings

_connect_args = {}
if settings.database_url.startswith("sqlite"):
    _connect_args["check_same_thread"] = False


def _build_engine() -> Engine:
    if settings.database_url.startswith("sqlite"):
        # Make sure the data directory exists before SQLAlchemy opens the file.
        ensure_directories()
        db_file = settings.db_path
        if db_file is not None:
            db_file.parent.mkdir(parents=True, exist_ok=True)
    return create_engine(
        settings.database_url,
        connect_args=_connect_args,
        future=True,
    )


engine = _build_engine()

if settings.database_url.startswith("sqlite"):

    @event.listens_for(engine, "connect")
    def _set_sqlite_pragma(dbapi_connection, _record):  # noqa: ANN001
        cursor = dbapi_connection.cursor()
        cursor.execute("PRAGMA journal_mode=WAL")
        cursor.execute("PRAGMA foreign_keys=ON")
        cursor.execute("PRAGMA busy_timeout=5000")
        cursor.close()


SessionLocal = sessionmaker(bind=engine, autoflush=False, expire_on_commit=False)


def get_db() -> Session:
    """FastAPI dependency."""
    db = SessionLocal()
    try:
        yield db
    finally:
        db.close()


def init_db() -> None:
    from app.database.models import Base

    ensure_directories()
    Base.metadata.create_all(bind=engine)
