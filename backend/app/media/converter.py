"""Audio conversion helpers (WAV -> MP3 via FFmpeg).

Timing is generated from the WAV *before* conversion; conversion is
time-preserving, so timings remain valid for the MP3.
"""

from __future__ import annotations

import logging
import shutil
import subprocess
from pathlib import Path

logger = logging.getLogger(__name__)


class MediaConversionError(RuntimeError):
    pass


def ffmpeg_available() -> bool:
    return shutil.which("ffmpeg") is not None


def _run_ffmpeg(args: list[str]) -> None:
    process = subprocess.run(
        ["ffmpeg", "-hide_banner", "-loglevel", "error", "-y", *args],
        capture_output=True, text=True, timeout=300,
    )
    if process.returncode != 0:
        raise MediaConversionError(
            f"ffmpeg failed ({process.returncode}): {process.stderr.strip()[-400:]}"
        )


def convert_wav_to_mp3(wav_path: Path, mp3_path: Path) -> Path:
    if not ffmpeg_available():
        raise MediaConversionError("ffmpeg not found on PATH")
    mp3_path.parent.mkdir(parents=True, exist_ok=True)
    _run_ffmpeg(["-i", str(wav_path), "-codec:a", "libmp3lame", "-qscale:a", "4",
                 str(mp3_path)])
    if not mp3_path.exists() or mp3_path.stat().st_size == 0:
        raise MediaConversionError(f"ffmpeg produced no output at {mp3_path}")
    return mp3_path


def finalize_audio(wav_path: Path, out_dir: Path, stem: str,
                   audio_format: str = "mp3") -> tuple[Path, str]:
    """Convert/copy `wav_path` into `out_dir` as the requested format.

    Returns (final_path, extension_without_dot). Falls back to WAV when MP3
    is requested but ffmpeg is unavailable.
    """
    out_dir.mkdir(parents=True, exist_ok=True)
    fmt = audio_format.lower()
    if fmt == "mp3" and ffmpeg_available():
        mp3_path = out_dir / f"{stem}.mp3"
        convert_wav_to_mp3(wav_path, mp3_path)
        wav_path.unlink(missing_ok=True)
        return mp3_path, "mp3"
    if fmt == "mp3":
        logger.warning("ffmpeg unavailable; storing WAV instead of MP3")
    wav_final = out_dir / f"{stem}.wav"
    shutil.move(str(wav_path), wav_final)
    return wav_final, "wav"


def probe_duration_seconds(path: Path) -> float | None:
    """Best-effort duration lookup via ffprobe; None when unavailable."""
    ffprobe = shutil.which("ffprobe")
    if ffprobe is None or not path.exists():
        return None
    try:
        out = subprocess.run(
            [ffprobe, "-v", "error", "-show_entries", "format=duration",
             "-of", "default=noprint_wrappers=1:nokey=1", str(path)],
            capture_output=True, text=True, timeout=60, check=True,
        ).stdout.strip()
        return round(float(out), 4)
    except Exception:  # noqa: BLE001 - probing is best-effort only
        return None
