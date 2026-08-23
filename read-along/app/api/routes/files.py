"""File serving: audio (.mp3/.wav) and timing JSON under DATA_DIR.

For the MVP the API serves generated files directly. Paths returned by
`GET /api/books/{id}` (audio_url/timing_url) map 1:1 onto this route.
"""

from __future__ import annotations

import mimetypes
from pathlib import Path

from fastapi import APIRouter, HTTPException
from fastapi.responses import FileResponse

from app.config import settings

router = APIRouter(tags=["files"])

MEDIA_TYPES = {
    ".mp3": "audio/mpeg",
    ".wav": "audio/wav",
    ".json": "application/json",
    ".txt": "text/plain",
}


@router.get("/files/{file_path:path}")
def serve_file(file_path: str) -> FileResponse:
    data_root = settings.resolve_data_dir().resolve()
    requested = Path(file_path)

    if requested.is_absolute() or ".." in requested.parts:
        raise HTTPException(status_code=403, detail="Forbidden path")

    full_path = (data_root / requested).resolve()
    if data_root not in full_path.parents:
        raise HTTPException(status_code=403, detail="Forbidden path")
    if not full_path.is_file():
        raise HTTPException(status_code=404, detail="File not found")

    media_type = MEDIA_TYPES.get(
        full_path.suffix.lower(),
        mimetypes.guess_type(full_path.name)[0] or "application/octet-stream",
    )
    return FileResponse(full_path, media_type=media_type)
