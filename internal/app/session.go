package app

import (
	"context"
	"errors"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/mindsgn-studio/amba/internal/align"
	"github.com/mindsgn-studio/amba/internal/epub"
	"github.com/mindsgn-studio/amba/internal/library"
	"github.com/mindsgn-studio/amba/internal/player"
	"github.com/mindsgn-studio/amba/internal/readalong"
	"github.com/mindsgn-studio/amba/internal/transcribe"
)

// Playback is what the reader needs from an audiobook player. The app uses
// *player.Playlist; tests use a stand-in.
type Playback interface {
	Play() error
	Pause()
	Toggle() error
	Seek(position time.Duration)
	Position() time.Duration
	Duration() time.Duration
	Playing() bool
	Speed() float64
	SetSpeed(speed float64)
	Err() error
	Close()
}

// speeds are the playback speeds the reader offers.
var speeds = []float64{0.75, 1, 1.25, 1.5, 1.75, 2}

var speedLabels = []string{"0.75×", "1×", "1.25×", "1.5×", "1.75×", "2×"}

const defaultSpeedIndex = 1

type readerStatus int

const (
	readerIdle readerStatus = iota
	readerLoading
	readerReady
	readerError
)

// session is one open book: its text, its narration and the listener's place
// in both. The transcript is kept so an attached ebook can be removed without
// transcribing the audiobook again, and the review is the attached ebook laid
// out for the reader's drawer.
type session struct {
	book       library.Book
	content    *readalong.Content
	transcript *transcribe.Transcript
	review     *readalong.Review
	playback   Playback
	duration   time.Duration
}

// readerState is everything the reader screen shows.
type readerState struct {
	book     library.Book
	bookID   string
	status   readerStatus
	stage    string
	fraction float64
	err      error
	session  *session
	cancel   context.CancelFunc

	chapter       int
	follow        bool
	speedIndex    int
	seekValue     float64
	scrubbing     bool
	lastParagraph int
	lastSave      time.Time
	playbackErr   error
	importErr     error
	removeErr     error
	list          ui.ListState

	drawerOpen    bool
	reviewPage    int
	confirmRemove bool
}

func newReaderState() readerState {
	return readerState{
		status:        readerIdle,
		fraction:      -1,
		follow:        true,
		speedIndex:    defaultSpeedIndex,
		lastParagraph: -1,
	}
}

// openBook leaves the library and prepares a book in the background.
func (a *App) openBook(book library.Book) {
	a.closeSession(true)
	a.reader = newReaderState()
	a.reader.book = book
	a.reader.bookID = book.ID
	a.reader.status = readerLoading
	a.reader.stage = "Preparing the audiobook…"
	a.reader.lastSave = time.Now()

	ctx, cancel := context.WithCancel(context.Background())
	a.reader.cancel = cancel

	a.router.Push("/read/" + book.ID)
	go a.prepare(ctx, book)
}

// prepare transcribes, aligns and opens the book, publishing each stage.
func (a *App) prepare(ctx context.Context, book library.Book) {
	report := func(progress transcribe.Progress) {
		a.Update(func() {
			if ctx.Err() != nil || a.reader.bookID != book.ID || a.reader.status != readerLoading {
				return
			}
			a.reader.stage = progress.Stage
			a.reader.fraction = progress.Fraction
		})
	}

	opened, err := a.buildSession(ctx, book, report)

	a.Update(func() {
		if ctx.Err() != nil || a.reader.bookID != book.ID || a.reader.status != readerLoading {
			if opened != nil {
				opened.playback.Close()
			}
			return
		}
		if err != nil {
			a.reader.status = readerError
			a.reader.err = err
			return
		}

		a.reader.session = opened
		a.reader.status = readerReady
		a.reader.chapter = opened.content.ChapterAt(opened.playback.Position())
		a.reader.speedIndex = speedIndexFor(opened.playback.Speed())
		a.reader.lastSave = time.Now()
		a.reader.list = ui.ListState{}
	})
}

// buildSession prepares a book synchronously. It is separate from the UI so
// the pieces can be tested and reused.
func (a *App) buildSession(
	ctx context.Context,
	book library.Book,
	report func(transcribe.Progress),
) (*session, error) {
	transcript, err := a.transcriber.Transcript(ctx, book.ID, book.AudioFiles, report)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	content, review := a.contentFor(ctx, book, transcript, report)
	if content == nil {
		return nil, errors.New("could not read the audiobook")
	}

	if len(transcript.Files) == 0 {
		return nil, errors.New("the audiobook has no playable files")
	}
	tracks := make([]player.Track, len(transcript.Files))
	for index, file := range transcript.Files {
		tracks[index] = player.Track{Path: file.Path, Duration: file.Duration}
	}

	playback := player.New(tracks)
	saved := a.progress.Load(book.ID)
	if saved.Position > 0 {
		playback.Seek(saved.Position)
	}
	speed := saved.Speed
	if speed <= 0 {
		speed = 1
	}
	playback.SetSpeed(speed)

	duration := transcript.Duration
	if duration <= 0 {
		duration = playback.Duration()
	}

	return &session{
		book:       book,
		content:    content,
		transcript: transcript,
		review:     review,
		playback:   playback,
		duration:   duration,
	}, nil
}

// contentFor prefers the book's own text when it matches the narration, and
// falls back to the transcript when it does not. The review is the attached
// ebook laid out for inspection, and is returned even when the reader had to
// fall back to the transcript, so the drawer can still show the printed book.
// It does not use the alignment cache; the app uses App.contentFor, which does.
func contentFor(ctx context.Context, book library.Book, transcript *transcribe.Transcript) (*readalong.Content, *readalong.Review) {
	if book.EPUBPath != "" {
		if err := ctx.Err(); err != nil {
			return nil, nil
		}
		document, err := epub.Open(book.EPUBPath)
		if err == nil {
			chapters, chaptersErr := document.Chapters()
			if chaptersErr == nil {
				content, alignErr := readalong.FromEPUB(chapters, transcript)
				if alignErr == nil {
					review := readalong.BuildReview(chapters, content.Alignment)
					if content.Synced {
						return content, review
					}
					return readalong.FromTranscript(transcript), review
				}
				return readalong.FromTranscript(transcript), readalong.BuildReview(chapters, nil)
			}
		}
	}
	return readalong.FromTranscript(transcript), nil
}

// contentFor loads and aligns the EPUB, reusing a cached alignment when the
// transcript, ebook text and configuration are unchanged. It never starts a
// new transcription. The review is built from the book's own text, so it
// survives a fallback to the transcript.
func (a *App) contentFor(
	ctx context.Context,
	book library.Book,
	transcript *transcribe.Transcript,
	report func(transcribe.Progress),
) (*readalong.Content, *readalong.Review) {
	if book.EPUBPath == "" {
		return readalong.FromTranscript(transcript), nil
	}
	if err := ctx.Err(); err != nil {
		return nil, nil
	}

	report(transcribe.Progress{Stage: "Reading the EPUB…", Fraction: 0})
	document, err := epub.Open(book.EPUBPath)
	if err != nil {
		return readalong.FromTranscript(transcript), nil
	}
	chapters, err := document.Chapters()
	if err != nil {
		return readalong.FromTranscript(transcript), nil
	}

	config := align.DefaultConfig()
	key := readalong.AlignmentKey(chapters, transcript, config)
	cachePath := a.cache.File("books", book.ID, "alignment.json")

	var cached readalong.Alignment
	if err := a.cache.ReadJSON(cachePath, &cached); err == nil &&
		cached.Key == key && cached.Version == readalong.AlignmentVersion {
		if content, buildErr := readalong.FromEPUBWithAlignment(chapters, transcript, &cached); buildErr == nil {
			review := readalong.BuildReview(chapters, &cached)
			if !content.Synced {
				return readalong.FromTranscript(transcript), review
			}
			return content, review
		}
	}

	report(transcribe.Progress{Stage: "Matching the book to the narration…", Fraction: 0})
	alignment, err := readalong.AlignBook(ctx, chapters, transcript, config, func(fraction float64) {
		report(transcribe.Progress{Stage: "Matching the book to the narration…", Fraction: fraction})
	})
	if err != nil {
		if ctx.Err() != nil {
			return nil, nil
		}
		return readalong.FromTranscript(transcript), readalong.BuildReview(chapters, nil)
	}

	// A cache failure must not stop the book from opening.
	_ = a.cache.WriteJSON(cachePath, alignment)

	content, err := readalong.BuildContent(chapters, transcript, alignment)
	review := readalong.BuildReview(chapters, alignment)
	if err != nil || !content.Synced {
		return readalong.FromTranscript(transcript), review
	}
	return content, review
}

// closeSession stops the reader and optionally saves its place.
func (a *App) closeSession(save bool) {
	if a.reader.cancel != nil {
		a.reader.cancel()
		a.reader.cancel = nil
	}
	if a.reader.session == nil {
		a.reader.status = readerIdle
		return
	}

	if save {
		a.saveProgress(a.reader.session)
	}
	a.reader.session.playback.Close()
	a.reader.session = nil
	a.reader.status = readerIdle
}

// saveProgress records where the listener is in a book.
func (a *App) saveProgress(opened *session) {
	progress := library.Progress{
		Position: opened.playback.Position(),
		Duration: opened.duration,
		Speed:    opened.playback.Speed(),
		Updated:  time.Now(),
	}
	if err := a.progress.Save(opened.book.ID, progress); err == nil {
		a.progressCache[opened.book.ID] = progress
	}
}

func speedIndexFor(speed float64) int {
	best := defaultSpeedIndex
	bestDistance := -1.0
	for index, candidate := range speeds {
		distance := candidate - speed
		if distance < 0 {
			distance = -distance
		}
		if bestDistance < 0 || distance < bestDistance {
			best = index
			bestDistance = distance
		}
	}
	return best
}

func (a *App) retryBook() {
	if a.reader.book.ID == "" {
		a.goLibrary()
		return
	}
	a.openBook(a.reader.book)
}

func (a *App) goLibrary() {
	a.closeSession(true)
	a.reader = newReaderState()
	a.router.Replace("/library")
}

func (a *App) seekChapter(chapter int) {
	opened := a.reader.session
	if opened == nil {
		return
	}
	if chapter < 0 || chapter >= len(opened.content.Chapters) {
		return
	}

	a.reader.chapter = chapter
	a.reader.lastParagraph = -1
	a.reader.list.ScrollTo(0, ui.Start)
	opened.playback.Seek(opened.content.ChapterStart(chapter))
	a.saveProgress(opened)
}

// String names a reader status for tests and diagnostics.
func (status readerStatus) String() string {
	switch status {
	case readerLoading:
		return "loading"
	case readerReady:
		return "ready"
	case readerError:
		return "error"
	default:
		return "idle"
	}
}
