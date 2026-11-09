package app

import (
	"archive/zip"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/mindsgn-studio/amba/internal/library"
	"github.com/mindsgn-studio/amba/internal/readalong"
	"github.com/mindsgn-studio/amba/internal/transcribe"
)

// fakePlayback is a player that keeps its place in memory.
type fakePlayback struct {
	position time.Duration
	duration time.Duration
	playing  bool
	speed    float64
	err      error
	closed   bool
}

func (f *fakePlayback) Play() error {
	if f.err != nil {
		return f.err
	}
	f.playing = true
	return nil
}

func (f *fakePlayback) Pause()                      { f.playing = false }
func (f *fakePlayback) Seek(position time.Duration) { f.position = position }
func (f *fakePlayback) Position() time.Duration     { return f.position }
func (f *fakePlayback) Duration() time.Duration     { return f.duration }
func (f *fakePlayback) Playing() bool               { return f.playing }
func (f *fakePlayback) Err() error                  { return f.err }
func (f *fakePlayback) Close()                      { f.closed = true; f.playing = false }

func (f *fakePlayback) Toggle() error {
	if f.playing {
		f.Pause()
		return nil
	}
	return f.Play()
}

func (f *fakePlayback) Speed() float64 { return f.speed }

func (f *fakePlayback) SetSpeed(speed float64) { f.speed = speed }

func newTestApp(t *testing.T) *App {
	t.Helper()

	root := t.TempDir()
	application, err := New(Config{
		LibraryRoot: filepath.Join(root, "audiobooks"),
		CacheRoot:   filepath.Join(root, "cache"),
		ModelDir:    filepath.Join(root, "model"),
		ModelsDir:   filepath.Join(root, "models"),
	})
	if err != nil {
		t.Fatal(err)
	}
	return application
}

func testContent() *readalong.Content {
	transcript := &transcribe.Transcript{
		Files: []transcribe.File{
			{Path: "one.mp3", Start: 0, Duration: 10 * time.Second},
			{Path: "two.mp3", Start: 10 * time.Second, Duration: 10 * time.Second},
		},
		Words: []transcribe.Word{
			{Text: "Call", Start: 0, End: 400 * time.Millisecond},
			{Text: "me", Start: 500 * time.Millisecond, End: 800 * time.Millisecond},
			{Text: "Three", Start: 10 * time.Second, End: 10*time.Second + 400*time.Millisecond},
			{Text: "four", Start: 10*time.Second + 500*time.Millisecond, End: 10*time.Second + 800*time.Millisecond},
		},
		Duration: 20 * time.Second,
	}
	return readalong.FromTranscript(transcript)
}

func readyReader(application *App, book library.Book, playback Playback) {
	application.books = []library.Book{book}
	application.reader = newReaderState()
	application.reader.book = book
	application.reader.bookID = book.ID
	application.reader.status = readerReady
	application.reader.session = &session{
		book:     book,
		content:  testContent(),
		playback: playback,
		duration: 20 * time.Second,
	}
}

func TestLoadingView(t *testing.T) {
	application := newTestApp(t)
	application.scanning = true
	application.scanStage = "Looking for audiobooks…"
	application.router.Replace("/")

	tester := ui.NewTester(application.View, 1040, 780)
	tester.Frame()

	if !tester.HasText("Amba") {
		t.Error("loading screen does not show the app name")
	}
	if !tester.HasText("Looking for audiobooks…") {
		t.Errorf("loading screen does not show its stage: %q", tester.Texts())
	}
}

func TestLibraryView(t *testing.T) {
	application := newTestApp(t)
	application.books = []library.Book{
		{ID: "one", Title: "Neuromancer", Author: "William Gibson"},
		{ID: "two", Title: "Snow Crash"},
	}
	application.progressCache = map[string]library.Progress{
		"one": {Position: 2 * time.Hour, Duration: 6 * time.Hour},
	}
	application.router.Replace("/library")

	tester := ui.NewTester(application.View, 1040, 780)
	tester.Frame()

	if !tester.HasText("Neuromancer") || !tester.HasText("Snow Crash") {
		t.Errorf("library does not list the books: %q", tester.Texts())
	}
	if !tester.HasText("William Gibson") {
		t.Error("library does not show the author")
	}
	if !tester.HasText("Not started") {
		t.Error("library does not show the unstarted book's progress")
	}
	if !tester.HasText("33%") {
		t.Errorf("library does not show the started book's progress: %q", tester.Texts())
	}
}

func TestLibraryEmptyView(t *testing.T) {
	application := newTestApp(t)
	application.router.Replace("/library")

	tester := ui.NewTester(application.View, 1040, 780)
	tester.Frame()

	if !tester.HasText("Your shelf is empty") {
		t.Errorf("empty library does not explain itself: %q", tester.Texts())
	}
}

func TestReaderViewPlays(t *testing.T) {
	application := newTestApp(t)
	book := library.Book{ID: "book", Title: "Neuromancer"}
	playback := &fakePlayback{duration: 20 * time.Second}
	readyReader(application, book, playback)
	application.router.Replace("/read/book")

	tester := ui.NewTester(application.View, 1040, 780)
	tester.Frame()

	if !tester.HasText("Neuromancer") {
		t.Error("reader does not show the book's title")
	}
	if !tester.HasText("Part 1") {
		t.Errorf("reader does not show the chapter: %q", tester.Texts())
	}
	if !tester.HasText("Play") {
		t.Errorf("reader does not offer playback: %q", tester.Texts())
	}
	if !tester.HasText("Follow") {
		t.Error("reader does not offer follow mode")
	}

	if err := tester.Click("Play"); err != nil {
		t.Fatal(err)
	}
	tester.Frame()

	if !playback.playing {
		t.Error("clicking Play did not start playback")
	}
	if !tester.HasText("Pause") {
		t.Error("reader does not offer to pause while playing")
	}
}

func TestReaderNotFound(t *testing.T) {
	application := newTestApp(t)
	application.router.Replace("/read/missing")

	tester := ui.NewTester(application.View, 1040, 780)
	tester.Frame()

	if !tester.HasText("Page not found") {
		t.Errorf("missing book does not show the not-found page: %q", tester.Texts())
	}
}

func TestCloseSessionSavesProgress(t *testing.T) {
	application := newTestApp(t)
	book := library.Book{ID: "book"}
	playback := &fakePlayback{
		position: 90 * time.Second,
		duration: 20 * time.Minute,
		speed:    1.5,
	}
	application.reader.session = &session{book: book, playback: playback, duration: 20 * time.Minute}
	application.reader.bookID = book.ID
	application.reader.status = readerReady

	application.closeSession(true)

	if !playback.closed {
		t.Error("closeSession() left the player open")
	}
	if application.reader.session != nil || application.reader.status != readerIdle {
		t.Error("closeSession() left the reader open")
	}

	progress := application.progress.Load(book.ID)
	if progress.Position != 90*time.Second || progress.Duration != 20*time.Minute || progress.Speed != 1.5 {
		t.Errorf("saved progress = %+v, want position 1m30s, duration 20m and speed 1.5", progress)
	}
	if cached := application.progressCache[book.ID]; cached.Position != 90*time.Second {
		t.Errorf("cached progress = %+v, want the saved position", cached)
	}
}

func TestSeekChapter(t *testing.T) {
	application := newTestApp(t)
	book := library.Book{ID: "book"}
	playback := &fakePlayback{duration: 20 * time.Second}
	readyReader(application, book, playback)
	application.router.Replace("/read/book")

	application.seekChapter(1)

	if application.reader.chapter != 1 {
		t.Errorf("chapter = %d, want 1", application.reader.chapter)
	}
	if playback.position != 10*time.Second {
		t.Errorf("position = %v, want 10s", playback.position)
	}
	if got := application.progress.Load(book.ID).Position; got != 10*time.Second {
		t.Errorf("saved position = %v, want 10s", got)
	}

	// Out-of-range chapters do nothing.
	application.seekChapter(99)
	if application.reader.chapter != 1 || playback.position != 10*time.Second {
		t.Error("seekChapter() moved for an out-of-range chapter")
	}
}

func TestContentForFallsBackToTranscript(t *testing.T) {
	transcript := &transcribe.Transcript{
		Words:    []transcribe.Word{{Text: "word", Start: 0, End: time.Second}},
		Duration: time.Second,
	}

	content := contentFor(context.Background(), library.Book{}, transcript)
	if content == nil {
		t.Fatal("contentFor() = nil")
	}
	if content.Synced {
		t.Error("contentFor() trusted transcript-only content")
	}

	// A missing EPUB falls back the same way.
	content = contentFor(context.Background(), library.Book{EPUBPath: filepath.Join(t.TempDir(), "missing.epub")}, transcript)
	if content == nil || content.Synced {
		t.Errorf("contentFor() with a missing EPUB = %+v, want transcript-only content", content)
	}
}

func TestContentForUsesMatchingEPUB(t *testing.T) {
	epubPath := writeTestEPUB(t)
	transcript := &transcribe.Transcript{
		Words: []transcribe.Word{
			{Text: "The", Start: 0, End: 400 * time.Millisecond},
			{Text: "cat", Start: 500 * time.Millisecond, End: 900 * time.Millisecond},
			{Text: "sat.", Start: time.Second, End: 1400 * time.Millisecond},
		},
		Duration: 1500 * time.Millisecond,
	}

	content := contentFor(context.Background(), library.Book{EPUBPath: epubPath}, transcript)
	if content == nil {
		t.Fatal("contentFor() = nil")
	}
	if !content.Synced {
		t.Fatalf("contentFor() did not use the matching EPUB (confidence %v)", content.Confidence)
	}
	if content.Chapters[0].Title != "Chapter One" {
		t.Errorf("chapter title = %q, want %q", content.Chapters[0].Title, "Chapter One")
	}
}

func TestSpeedIndexFor(t *testing.T) {
	tests := []struct {
		speed float64
		want  int
	}{
		{1, 1},
		{1.1, 1},
		{1.4, 3},
		{0.7, 0},
		{2.2, 5},
	}
	for _, test := range tests {
		if got := speedIndexFor(test.speed); got != test.want {
			t.Errorf("speedIndexFor(%v) = %d, want %d", test.speed, got, test.want)
		}
	}
}

func TestFormat(t *testing.T) {
	clocks := []struct {
		value time.Duration
		want  string
	}{
		{-time.Second, "0:00"},
		{65 * time.Second, "1:05"},
		{time.Hour + 4*time.Minute + 5*time.Second, "1:04:05"},
	}
	for _, test := range clocks {
		if got := formatClock(test.value); got != test.want {
			t.Errorf("formatClock(%v) = %q, want %q", test.value, got, test.want)
		}
	}

	durations := []struct {
		value time.Duration
		want  string
	}{
		{0, "0m"},
		{30 * time.Second, "30s"},
		{45 * time.Minute, "45m"},
		{6*time.Hour + 12*time.Minute, "6h 12m"},
		{6 * time.Hour, "6h"},
	}
	for _, test := range durations {
		if got := formatDuration(test.value); got != test.want {
			t.Errorf("formatDuration(%v) = %q, want %q", test.value, got, test.want)
		}
	}

	if got := initials("Neuromancer"); got != "N" {
		t.Errorf("initials() = %q, want N", got)
	}
	if got := initials("The Left Hand of Darkness"); got != "TL" {
		t.Errorf("initials() = %q, want TL", got)
	}
	if got := initials("..."); got != "?" {
		t.Errorf("initials() = %q, want ?", got)
	}
}

func TestReaderStatusString(t *testing.T) {
	tests := map[readerStatus]string{
		readerIdle:    "idle",
		readerLoading: "loading",
		readerReady:   "ready",
		readerError:   "error",
	}
	for status, want := range tests {
		if got := status.String(); got != want {
			t.Errorf("%d.String() = %q, want %q", int(status), got, want)
		}
	}
}

// writeTestEPUB writes a one-chapter EPUB whose text matches
// TestContentForUsesMatchingEPUB's transcript.
func writeTestEPUB(t *testing.T) string {
	t.Helper()

	filename := filepath.Join(t.TempDir(), "book.epub")
	file, err := os.Create(filename)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	writer := zip.NewWriter(file)
	entries := map[string]string{
		"META-INF/container.xml": `<?xml version="1.0"?>
<container xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles><rootfile full-path="OEBPS/content.opf"/></rootfiles>
</container>`,
		"OEBPS/content.opf": `<?xml version="1.0"?>
<package xmlns="http://www.idpf.org/2007/opf">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/"><dc:title>Test Book</dc:title></metadata>
  <manifest><item id="ch1" href="chapter1.xhtml" media-type="application/xhtml+xml"/></manifest>
  <spine><itemref idref="ch1"/></spine>
</package>`,
		"OEBPS/chapter1.xhtml": `<html><body><h1>Chapter One</h1><p>The cat sat.</p></body></html>`,
	}
	for name, data := range entries {
		part, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := part.Write([]byte(data)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return filename
}
