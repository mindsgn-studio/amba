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

func testTranscript() *transcribe.Transcript {
	return &transcribe.Transcript{
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
}

func testContent() *readalong.Content {
	return readalong.FromTranscript(testTranscript())
}

func testReview() *readalong.Review {
	return &readalong.Review{
		Pages: []readalong.ReviewPage{
			{
				Chapter:  0,
				Title:    "Part 1",
				Accepted: true,
				Paragraphs: []readalong.ReviewParagraph{{
					Words: []readalong.ReviewWord{
						{Text: "Call", Synced: true, Word: true},
						{Text: "me", Synced: true, Word: true},
						{Text: "Ishmael.", Synced: false, Word: true},
					},
				}},
			},
			{
				Chapter:  1,
				Title:    "Part 2",
				Accepted: false,
				Paragraphs: []readalong.ReviewParagraph{{
					Words: []readalong.ReviewWord{{Text: "Three", Synced: false, Word: true}},
				}},
			},
		},
		Synced: 2,
		Total:  4,
	}
}

func readyReader(application *App, book library.Book, playback Playback) {
	application.books = []library.Book{book}
	application.reader = newReaderState()
	application.reader.book = book
	application.reader.bookID = book.ID
	application.reader.status = readerReady
	application.reader.session = &session{
		book:       book,
		content:    testContent(),
		transcript: testTranscript(),
		playback:   playback,
		duration:   20 * time.Second,
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

	content, review := contentFor(context.Background(), library.Book{}, transcript)
	if content == nil {
		t.Fatal("contentFor() = nil")
	}
	if content.Synced {
		t.Error("contentFor() trusted transcript-only content")
	}
	if review != nil {
		t.Error("contentFor() built a review without an ebook")
	}

	// A missing EPUB falls back the same way.
	content, review = contentFor(context.Background(), library.Book{EPUBPath: filepath.Join(t.TempDir(), "missing.epub")}, transcript)
	if content == nil || content.Synced {
		t.Errorf("contentFor() with a missing EPUB = %+v, want transcript-only content", content)
	}
	if review != nil {
		t.Error("contentFor() built a review for a missing EPUB")
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

	content, review := contentFor(context.Background(), library.Book{EPUBPath: epubPath}, transcript)
	if content == nil {
		t.Fatal("contentFor() = nil")
	}
	if !content.Synced {
		t.Fatalf("contentFor() did not use the matching EPUB (confidence %v)", content.Confidence)
	}
	if content.Chapters[0].Title != "Chapter One" {
		t.Errorf("chapter title = %q, want %q", content.Chapters[0].Title, "Chapter One")
	}
	if review == nil {
		t.Fatal("contentFor() built no review for the matching EPUB")
	}
	if review.Total != 3 || review.Synced != 3 {
		t.Errorf("review = %d of %d synced, want 3 of 3", review.Synced, review.Total)
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

func TestContentForCachesAlignmentAndInvalidatesOnChanges(t *testing.T) {
	application := newTestApp(t)
	epubPath := writeTestEPUB(t)
	book := library.Book{ID: "book", EPUBPath: epubPath}
	transcript := &transcribe.Transcript{
		Words: []transcribe.Word{
			{Text: "The", Start: 0, End: 400 * time.Millisecond},
			{Text: "cat", Start: 500 * time.Millisecond, End: 900 * time.Millisecond},
			{Text: "sat.", Start: time.Second, End: 1400 * time.Millisecond},
		},
		Duration: 1500 * time.Millisecond,
	}
	report := func(transcribe.Progress) {}

	content, review := application.contentFor(context.Background(), book, transcript, report)
	if content == nil || !content.Synced {
		t.Fatalf("contentFor() = %+v, want synced content", content)
	}
	if review == nil || review.Total == 0 {
		t.Fatal("contentFor() built no review")
	}

	cachePath := application.cache.File("books", book.ID, "alignment.json")
	var first readalong.Alignment
	if err := application.cache.ReadJSON(cachePath, &first); err != nil {
		t.Fatalf("alignment was not cached: %v", err)
	}
	if first.Key == "" || first.Version != readalong.AlignmentVersion {
		t.Errorf("cached alignment = %+v, want a key and version", first)
	}
	if len(first.Matches) == 0 {
		t.Error("cached alignment has no word matches")
	}

	// The same inputs use the cache and keep the same key.
	content, _ = application.contentFor(context.Background(), book, transcript, report)
	if content == nil || !content.Synced {
		t.Fatal("contentFor() failed on the cached path")
	}
	var again readalong.Alignment
	if err := application.cache.ReadJSON(cachePath, &again); err != nil {
		t.Fatal(err)
	}
	if again.Key != first.Key {
		t.Errorf("cache key changed for unchanged inputs: %q then %q", first.Key, again.Key)
	}

	// A changed transcript invalidates the alignment.
	changed := &transcribe.Transcript{
		Words:    append([]transcribe.Word(nil), transcript.Words...),
		Duration: transcript.Duration,
	}
	changed.Words[1].Text = "dog"
	_, _ = application.contentFor(context.Background(), book, changed, report)
	var second readalong.Alignment
	if err := application.cache.ReadJSON(cachePath, &second); err != nil {
		t.Fatal(err)
	}
	if second.Key == first.Key {
		t.Error("changed transcript did not invalidate the alignment cache")
	}
}

func TestAttachEPUBPersistsImport(t *testing.T) {
	application := newTestApp(t)
	book := library.Book{ID: "book", Title: "Book"}
	application.books = []library.Book{book}
	application.reader.bookID = "other"

	epubPath := writeTestEPUB(t)
	if err := application.attachEPUB(book.ID, epubPath); err != nil {
		t.Fatal(err)
	}
	if application.books[0].EPUBPath != epubPath {
		t.Errorf("book EPUBPath = %q, want %q", application.books[0].EPUBPath, epubPath)
	}
	if application.books[0].Title != "Test Book" {
		t.Errorf("book title = %q, want the EPUB title", application.books[0].Title)
	}

	// A fresh scan reattaches the imported EPUB from the cache.
	scanned := []library.Book{{ID: book.ID, Title: "Book"}}
	scanned = application.applyAttachments(scanned)
	if scanned[0].EPUBPath != epubPath {
		t.Errorf("applyAttachments() EPUBPath = %q, want %q", scanned[0].EPUBPath, epubPath)
	}

	// An explicit import wins over an EPUB found beside the audiobook.
	scanned = []library.Book{{ID: book.ID, Title: "Book", EPUBPath: filepath.Join(t.TempDir(), "other.epub")}}
	scanned = application.applyAttachments(scanned)
	if scanned[0].EPUBPath != epubPath {
		t.Errorf("applyAttachments() EPUBPath = %q, want the imported %q", scanned[0].EPUBPath, epubPath)
	}
}

func TestContentForFallsBackToTranscriptWhenEPUBDoesNotMatch(t *testing.T) {
	application := newTestApp(t)
	book := library.Book{ID: "book", EPUBPath: writeTestEPUB(t)}
	transcript := &transcribe.Transcript{
		Words: []transcribe.Word{
			{Text: "alpha", Start: 0, End: 400 * time.Millisecond},
			{Text: "bravo", Start: 500 * time.Millisecond, End: 900 * time.Millisecond},
			{Text: "charlie", Start: time.Second, End: 1400 * time.Millisecond},
			{Text: "delta", Start: 1500 * time.Millisecond, End: 1900 * time.Millisecond},
			{Text: "echo", Start: 2 * time.Second, End: 2400 * time.Millisecond},
		},
		Duration: 2500 * time.Millisecond,
	}

	content, review := application.contentFor(context.Background(), book, transcript, func(transcribe.Progress) {})
	if content == nil {
		t.Fatal("contentFor() returned nil")
	}
	if content.Synced {
		t.Fatalf("unrelated EPUB was marked synced: %v", content.Diagnostics)
	}
	for _, chapter := range content.Chapters {
		if chapter.Source != "transcript" {
			t.Errorf("chapter source = %q, want transcript fallback", chapter.Source)
		}
	}
	if review == nil {
		t.Fatal("contentFor() built no review for the unrelated EPUB")
	}
	if review.Total == 0 || review.Synced != 0 {
		t.Errorf("review = %d of %d synced, want unsynced printed text", review.Synced, review.Total)
	}
}

func TestReaderEbookDrawer(t *testing.T) {
	application := newTestApp(t)
	book := library.Book{ID: "book", Title: "Neuromancer", EPUBPath: "book.epub"}
	playback := &fakePlayback{duration: 20 * time.Second}
	readyReader(application, book, playback)
	application.reader.session.review = testReview()
	application.router.Replace("/read/book")

	tester := ui.NewTester(application.View, 1200, 800)
	tester.Frame()

	if !tester.HasText("Show ebook") {
		t.Fatalf("reader does not offer the ebook drawer: %q", tester.Texts())
	}
	if err := tester.Click("Show ebook"); err != nil {
		t.Fatal(err)
	}
	tester.Frame()

	for _, want := range []string{"Ebook", "● Synced", "● Unsynced", "Part 1", "Page 1 of 2", "Remove EPUB", "50% synced"} {
		if !tester.HasText(want) {
			t.Errorf("drawer does not show %q: %q", want, tester.Texts())
		}
	}

	if err := tester.Click("Next page"); err != nil {
		t.Fatal(err)
	}
	tester.Frame()
	if !tester.HasText("Page 2 of 2") || !tester.HasText("Part 2") {
		t.Errorf("drawer did not turn to page 2: %q", tester.Texts())
	}
	if !tester.HasText("This chapter falls back") {
		t.Error("drawer does not explain a rejected chapter")
	}

	if err := tester.Click("Hide"); err != nil {
		t.Fatal(err)
	}
	tester.Frame()
	if tester.HasText("Remove EPUB") {
		t.Errorf("drawer stayed open after Hide: %q", tester.Texts())
	}
}

func TestReviewSpansColorsSyncState(t *testing.T) {
	theme := ui.LightTheme()
	paragraph := readalong.ReviewParagraph{Words: []readalong.ReviewWord{
		{Text: "Synced", Synced: true, Word: true},
		{Text: "unsynced", Word: true},
		{Text: "—"},
	}}

	spans := reviewSpans(paragraph, theme)
	if len(spans) != 3 {
		t.Fatalf("spans = %d, want 3", len(spans))
	}
	if spans[0].Color != theme.Success || spans[0].Text != "Synced " {
		t.Errorf("synced span = %+v, want green and a trailing space", spans[0])
	}
	if spans[1].Color != theme.Danger || spans[1].Text != "unsynced " {
		t.Errorf("unsynced span = %+v, want red and a trailing space", spans[1])
	}
	if spans[2].Color != theme.TextMuted || spans[2].Text != "—" {
		t.Errorf("punctuation span = %+v, want muted and no trailing space", spans[2])
	}
}

func TestRemoveEPUBRemembersFolderBook(t *testing.T) {
	application := newTestApp(t)
	directory := t.TempDir()
	epubPath := filepath.Join(directory, "book.epub")
	if err := os.WriteFile(epubPath, []byte("epub"), 0o644); err != nil {
		t.Fatal(err)
	}

	book := library.Book{
		ID:        "book",
		Title:     "Test Book",
		Author:    "An Author",
		Directory: directory,
		EPUBPath:  epubPath,
		Cover:     []byte("cover"),
	}
	playback := &fakePlayback{duration: 20 * time.Second}
	readyReader(application, book, playback)
	application.reader.session.review = testReview()
	application.reader.drawerOpen = true

	if err := application.removeEPUB(book.ID); err != nil {
		t.Fatal(err)
	}

	updated := application.books[0]
	if updated.EPUBPath != "" {
		t.Errorf("EPUBPath = %q, want none", updated.EPUBPath)
	}
	if updated.Title != library.TitleFor(directory) {
		t.Errorf("Title = %q, want the folder title %q", updated.Title, library.TitleFor(directory))
	}
	if updated.Author != "" || updated.Cover != nil {
		t.Errorf("removed ebook metadata survived: %+v", updated)
	}

	var record epubRemoval
	if err := application.cache.ReadJSON(application.removalPath(book.ID), &record); err != nil {
		t.Fatalf("removal was not recorded: %v", err)
	}
	if record.Path != epubPath {
		t.Errorf("removal path = %q, want %q", record.Path, epubPath)
	}

	// The open book falls back to the transcript in place, without closing
	// the player or transcribing again.
	if application.reader.session == nil {
		t.Fatal("removeEPUB() closed the session")
	}
	if application.reader.session.review != nil {
		t.Error("removeEPUB() kept the review")
	}
	if application.reader.session.content == nil || application.reader.session.content.Synced {
		t.Error("removeEPUB() did not fall back to the transcript")
	}
	if application.reader.drawerOpen {
		t.Error("removeEPUB() left the drawer open")
	}
	if playback.closed {
		t.Error("removeEPUB() closed the player")
	}

	// A later scan finds the file again; the removal record keeps it out.
	scanned := []library.Book{{
		ID:        book.ID,
		Title:     "Test Book",
		Author:    "An Author",
		Directory: directory,
		EPUBPath:  epubPath,
		Cover:     []byte("cover"),
	}}
	scanned = application.applyAttachments(scanned)
	if scanned[0].EPUBPath != "" {
		t.Errorf("applyAttachments() reattached the removed EPUB: %q", scanned[0].EPUBPath)
	}
	if scanned[0].Title != library.TitleFor(directory) {
		t.Errorf("applyAttachments() Title = %q, want the folder title", scanned[0].Title)
	}
}

func TestRemoveEPUBForgetsImportedBook(t *testing.T) {
	application := newTestApp(t)
	directory := t.TempDir()
	book := library.Book{ID: "book", Title: "Book", Directory: directory}
	application.books = []library.Book{book}
	application.reader.bookID = "other"

	imported := writeTestEPUB(t)
	if err := application.attachEPUB(book.ID, imported); err != nil {
		t.Fatal(err)
	}
	if err := application.removeEPUB(book.ID); err != nil {
		t.Fatal(err)
	}

	if application.books[0].EPUBPath != "" {
		t.Errorf("EPUBPath = %q, want none", application.books[0].EPUBPath)
	}
	var record epubImport
	if err := application.cache.ReadJSON(application.importPath(book.ID), &record); err == nil {
		t.Errorf("import record survived removal: %+v", record)
	}
	var removal epubRemoval
	if err := application.cache.ReadJSON(application.removalPath(book.ID), &removal); err == nil {
		t.Errorf("an imported EPUB left a folder removal record: %+v", removal)
	}

	scanned := application.applyAttachments([]library.Book{{ID: book.ID, Directory: directory}})
	if scanned[0].EPUBPath != "" {
		t.Errorf("applyAttachments() reattached the removed import: %q", scanned[0].EPUBPath)
	}
}

func TestRemoveEPUBFromDrawer(t *testing.T) {
	application := newTestApp(t)
	directory := t.TempDir()
	epubPath := filepath.Join(directory, "book.epub")
	if err := os.WriteFile(epubPath, []byte("epub"), 0o644); err != nil {
		t.Fatal(err)
	}

	book := library.Book{ID: "book", Title: "Book", Directory: directory, EPUBPath: epubPath}
	playback := &fakePlayback{duration: 20 * time.Second}
	readyReader(application, book, playback)
	application.reader.session.review = testReview()
	application.router.Replace("/read/book")

	tester := ui.NewTester(application.View, 1200, 800)
	tester.Frame()
	if err := tester.Click("Show ebook"); err != nil {
		t.Fatal(err)
	}
	tester.Frame()
	if err := tester.Click("Remove EPUB"); err != nil {
		t.Fatal(err)
	}
	tester.Frame()

	if !tester.HasText("Remove the ebook?") {
		t.Fatalf("removal did not ask for confirmation: %q", tester.Texts())
	}
	if err := tester.Click("Cancel"); err != nil {
		t.Fatal(err)
	}
	tester.Frame()
	if application.books[0].EPUBPath == "" {
		t.Fatal("Cancel removed the EPUB")
	}
	if !tester.HasText("Remove EPUB") {
		t.Error("Cancel closed the drawer")
	}

	if err := tester.Click("Remove EPUB"); err != nil {
		t.Fatal(err)
	}
	tester.Frame()
	if err := tester.Click("Remove ebook"); err != nil {
		t.Fatal(err)
	}
	tester.Frame()

	if application.books[0].EPUBPath != "" {
		t.Errorf("EPUBPath = %q, want none", application.books[0].EPUBPath)
	}
	if tester.HasText("Remove EPUB") {
		t.Error("drawer stayed open after removal")
	}
	if !tester.HasText("Import EPUB") {
		t.Error("reader does not offer to import an EPUB after removal")
	}
	if playback.closed {
		t.Error("removing the ebook closed the player")
	}
}
