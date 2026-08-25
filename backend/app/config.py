"""Central configuration.

All paths and tunables come from environment variables (optionally a `.env`
file in the project root). Nothing is hard-coded.
"""

from __future__ import annotations

import logging
import sys
from functools import lru_cache
from pathlib import Path

from pydantic_settings import BaseSettings, SettingsConfigDict

PROJECT_ROOT = Path(__file__).resolve().parent.parent


class Settings(BaseSettings):
    model_config = SettingsConfigDict(
        env_file=(PROJECT_ROOT / ".env"),
        env_file_encoding="utf-8",
        extra="ignore",
        case_sensitive=False,
    )

    app_env: str = "development"

    database_url: str = f"sqlite:///{PROJECT_ROOT / 'data' / 'app.db'}"

    data_dir: Path = PROJECT_ROOT / "data"

    tts_provider: str = "kokoro"
    tts_voice: str = "af_heart"
    tts_speed: float = 1.0
    audio_format: str = "mp3"  # mp3 | wav

    chunk_target_words: int = 800
    chunk_max_words: int = 1500

    api_host: str = "0.0.0.0"
    api_port: int = 8000
    cors_origins: list[str] = ["*"]

    log_level: str = "INFO"
    poll_interval_seconds: float = 1.0

    def resolve_data_dir(self) -> Path:
        p = Path(self.data_dir)
        return p if p.is_absolute() else (PROJECT_ROOT / p)

    @property
    def books_dir(self) -> Path:
        return self.resolve_data_dir() / "books"

    @property
    def db_path(self) -> Path | None:
        """Filesystem path of the SQLite file, or None for non-sqlite URLs."""
        prefix = "sqlite:///"
        if self.database_url.startswith(prefix):
            raw = self.database_url[len(prefix):]
            p = Path(raw)
            return p if p.is_absolute() else (PROJECT_ROOT / p)
        return None


@lru_cache
def get_settings() -> Settings:
    return Settings()


settings = get_settings()


def setup_logging() -> None:
    level = getattr(logging, settings.log_level.upper(), logging.INFO)
    logging.basicConfig(
        level=level,
        stream=sys.stdout,
        format="%(asctime)s %(levelname)s [%(name)s] %(message)s",
    )
    # Quiet down noisy third-party loggers unless debugging.
    if level > logging.DEBUG:
        for noisy in ("urllib3", "httpx", "httpcore", "ebooklib", "numba", "matplotlib"):
            logging.getLogger(noisy).setLevel(logging.WARNING)


def ensure_directories() -> None:
    for d in (
        settings.resolve_data_dir(),
        settings.books_dir,
        PROJECT_ROOT / "data",
    ):
        d.mkdir(parents=True, exist_ok=True)
