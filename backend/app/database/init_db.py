"""Create data directories and initialise the SQLite schema."""

from __future__ import annotations

import sys


def main() -> int:
    from app.config import ensure_directories, settings, setup_logging
    from app.database.database import engine, init_db

    setup_logging()
    ensure_directories()
    init_db()
    print(f"Database initialised: {settings.db_path or settings.database_url}")
    print(f"Data directory:       {settings.resolve_data_dir()}")
    engine.dispose()
    return 0


if __name__ == "__main__":
    sys.exit(main())
