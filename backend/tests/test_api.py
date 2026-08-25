"""API + end-to-end test: POST /api/books -> worker -> GET book -> files."""

from pathlib import Path

import pytest

from app.database import repository as repo
from app.database.database import SessionLocal
from app.timing.words import TimingDocument
from tests.conftest import drain_jobs


def test_health(client):
    response = client.get("/health")
    assert response.status_code == 200
    body = response.json()
    assert body["status"] == "ok"


def test_create_book_from_string_returns_queued_job(client):
    response = client.post("/api/books", json={
        "title": "Example",
        "text": "The quick brown fox jumps over the lazy dog.",
        "language": "en",
        "voice": "af_heart",
    })
    assert response.status_code == 202
    body = response.json()
    assert body["status"] == "queued"
    assert body["book_id"] and body["job_id"]


def test_upload_rejects_unsupported_type(client):
    response = client.post(
        "/api/books/upload",
        files={"file": ("virus.exe", b"MZ...", "application/octet-stream")},
    )
    assert response.status_code == 415


def test_end_to_end_string_to_completed_book(client):
    """The most important test: API -> job -> worker -> audio + timings."""
    create = client.post("/api/books", json={
        "title": "End to End Book",
        "text": " ".join(
            f"Paragraph {i} contains several sentences about the weather today. "
            f"The rain stopped and the sun came out again."
            for i in range(10)
        ),
    })
    assert create.status_code == 202
    book_id = create.json()["book_id"]
    job_id = create.json()["job_id"]

    # Simulate the worker picking the job up.
    assert drain_jobs() >= 1

    job = client.get(f"/api/jobs/{job_id}")
    assert job.status_code == 200
    assert job.json()["status"] == "completed"
    assert job.json()["progress"] == 100

    detail = client.get(f"/api/books/{book_id}")
    assert detail.status_code == 200
    book = detail.json()
    assert book["status"] == "completed"
    assert len(book["chapters"]) >= 1

    chunks = [c for ch in book["chapters"] for c in ch["chunks"]]
    assert len(chunks) > 0

    for chunk in chunks:
        assert chunk["audio_url"].startswith("/api/files/")
        assert chunk["timing_url"].endswith(".json")

        audio = client.get(chunk["audio_url"])
        assert audio.status_code == 200
        assert len(audio.content) > 100  # real audio bytes

        timing_response = client.get(chunk["timing_url"])
        assert timing_response.status_code == 200
        timing_payload = timing_response.json()
        words = timing_payload["words"]
        assert len(words) > 0

        starts = [w["start"] for w in words]
        ends = [w["end"] for w in words]
        assert starts == sorted(starts), "timestamps must be monotonic"
        assert all(s < e for s, e in zip(starts, ends))
        assert timing_payload["duration"] > 0


def test_get_unknown_book_404(client):
    assert client.get("/api/books/doesnotexist").status_code == 404


def test_get_unknown_job_404(client):
    assert client.get("/api/jobs/nope").status_code == 404


def test_files_route_blocks_traversal(client, tmp_path):
    assert client.get("/api/files/../secret.txt").status_code in (403, 404)
    assert client.get("/api/files/%2e%2e/secret.txt").status_code in (403, 404)
    assert client.get("/api/files/books/none/chunk.mp3").status_code == 404


def test_delete_book_removes_records_and_files(client):
    drain_jobs()  # deterministic starting point
    created = client.post("/api/books", json={
        "title": "Doomed", "text": "Short lived book. Goodbye soon.",
    })
    book_id = created.json()["book_id"]
    job_id = created.json()["job_id"]
    drain_jobs()

    with SessionLocal() as db:
        assert repo.get_job(db, job_id).status == "completed"

    deleted = client.delete(f"/api/books/{book_id}")
    assert deleted.status_code == 200
    assert deleted.json() == {"deleted": True, "id": book_id}
    assert client.get(f"/api/books/{book_id}").status_code == 404


@pytest.mark.integration
@pytest.mark.parametrize("builder,ext", [
    ("epub", "epub"),
])
def test_end_to_end_epub_upload(client, builder, ext, tmp_path: Path):
    """Optional integration: EPUB upload through the whole pipeline."""
    from tests.test_epub import _make_epub

    epub_path = tmp_path / f"test-book.{ext}"
    _make_epub(epub_path)

    uploaded = client.post(
        "/api/books/upload",
        files={"file": ("test-book.epub", epub_path.read_bytes(), "application/epub+zip")},
        data={"title": "Uploaded EPUB"},
    )
    if uploaded.status_code != 202:
        pytest.fail(f"upload failed: {uploaded.text}")
    book_id = uploaded.json()["book_id"]
    job_id = uploaded.json()["job_id"]

    assert drain_jobs() >= 1

    with SessionLocal() as db:
        book = repo.get_book(db, book_id)
        assert book.status == "completed"
        assert repo.get_job(db, job_id).status == "completed"
        tree = repo.chunks_for_book(db, book_id)

    total_words = sum(len(c.words) for _, chunks in tree for c in
                      [TimingDocument.load(Path(ch.timing_path)) for ch in chunks])
    assert total_words > 0
