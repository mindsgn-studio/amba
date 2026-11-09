package app

import (
	"context"
	"errors"
	"time"

	"github.com/egoist/mygo/ui"

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
// in both.
type session struct {
	book     library.Book
	content  *readalong.Content
	playback Playback
	duration time.Duration
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
	list          ui.ListState
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

	report(transcribe.Progress{Stage: "Matching the book to the narration…", Fraction: 1})
	content := contentFor(ctx, book, transcript)
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
		book:     book,
		content:  content,
		playback: playback,
		duration: duration,
	}, nil
}

// contentFor prefers the book's own text when it matches the narration, and
// falls back to the transcript when it does not.
func contentFor(ctx context.Context, book library.Book, transcript *transcribe.Transcript) *readalong.Content {
	if book.EPUBPath != "" {
		if err := ctx.Err(); err != nil {
			return nil
		}
		document, err := epub.Open(book.EPUBPath)
		if err == nil {
			chapters, chaptersErr := document.Chapters()
			if chaptersErr == nil {
				if content, alignErr := readalong.FromEPUB(chapters, transcript); alignErr == nil && content.Synced {
					return content
				}
			}
		}
	}
	return readalong.FromTranscript(transcript)
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
