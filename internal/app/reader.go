package app

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/mindsgn-studio/amba/internal/readalong"
)

// readerView is the third screen: the book, the narration and the word being
// read.
func (a *App) readerView(c *ui.Context, id string) {
	if a.reader.bookID != id {
		a.closeSession(true)
		book, ok := a.bookByID(id)
		if !ok {
			a.notFoundView(c)
			return
		}
		a.openBook(book)
		c.Invalidate()
		return
	}

	theme := c.Theme()
	ui.Column(c).Fill().Background(theme.Background).Children(func() {
		a.readerHeader(c)
		ui.Divider(c)
		switch a.reader.status {
		case readerReady:
			ui.Row(c).Grow(1).Children(func() {
				if a.reader.drawerOpen && a.reader.session != nil && a.reader.session.review != nil {
					a.reviewDrawer(c)
					ui.Divider(c)
				}
				ui.Column(c).Grow(1).Children(func() {
					a.readerBody(c)
					a.readerFooter(c)
				})
			})
		case readerError:
			a.readerErrorView(c)
		default:
			a.readerLoadingView(c)
		}
	})

	ui.Modal(c, &a.reader.confirmRemove, func() {
		a.confirmRemoveView(c)
	})
}

func (a *App) readerHeader(c *ui.Context) {
	theme := c.Theme()
	reader := &a.reader

	ui.Row(c).Padding(20, 14).Gap(12).AlignItems(ui.Center).Children(func() {
		if ui.Button(c, "‹ Library").Clicked() {
			a.goLibrary()
			return
		}

		ui.Column(c).Grow(1).Children(func() {
			ui.Text(c, reader.book.Title).FontSize(16).Bold()
			if reader.status == readerReady && reader.session != nil {
				ui.Text(c, a.readerSubtitle()).FontSize(12).TextColor(theme.TextMuted)
			}
		})

		if reader.status == readerReady && reader.session != nil {
			if reader.session.book.EPUBPath == "" {
				if ui.Button(c, "Import EPUB").Clicked() {
					a.importEPUB(reader.book.ID)
				}
			}
			if reader.session.review != nil {
				label := "Show ebook"
				if reader.drawerOpen {
					label = "Hide ebook"
				}
				if ui.Button(c, label).Clicked() {
					a.toggleReview()
				}
			}
			if ui.Button(c, "Previous").Disabled(reader.chapter <= 0).Clicked() {
				a.seekChapter(reader.chapter - 1)
			}
			last := len(reader.session.content.Chapters) - 1
			if ui.Button(c, "Next").Disabled(reader.chapter >= last).Clicked() {
				a.seekChapter(reader.chapter + 1)
			}
		}
	})
}

func (a *App) readerSubtitle() string {
	reader := &a.reader
	if reader.session == nil || len(reader.session.content.Chapters) == 0 {
		return ""
	}
	content := reader.session.content
	chapter := content.Chapters[reader.chapter]
	title := chapter.Title
	if title == "" {
		title = "Chapter"
	}
	mode := "Transcript"
	switch {
	case chapter.Source == "epub" && chapter.Accepted:
		mode = fmt.Sprintf("Book text · %.0f%% aligned", chapter.Confidence*100)
	case chapter.Rejected && content.Synced:
		mode = "Transcript fallback · EPUB sync unavailable"
	case content.Synced:
		mode = fmt.Sprintf("Book text · %.0f%% aligned", content.Confidence*100)
	}
	return title + " · " + mode
}

func (a *App) readerLoadingView(c *ui.Context) {
	theme := c.Theme()
	reader := &a.reader

	ui.Column(c).Fill().Center().Gap(14).Padding(32).Children(func() {
		ui.Spinner(c)
		ui.Text(c, reader.stage).FontSize(14).TextColor(theme.TextMuted).TextAlign(ui.Center).MaxWidth(520)
		if reader.fraction >= 0 {
			ui.Progress(c, reader.fraction).Width(360)
		}
		ui.Text(c, "The first time, the audiobook is transcribed. This can take a while.").
			FontSize(12).TextColor(theme.TextMuted).TextAlign(ui.Center).MaxWidth(420)
		if ui.Button(c, "Cancel").Clicked() {
			a.goLibrary()
		}
	})
}

func (a *App) readerErrorView(c *ui.Context) {
	theme := c.Theme()
	reader := &a.reader

	ui.Column(c).Fill().Center().Gap(12).Padding(32).Children(func() {
		ui.Text(c, "Could not open this audiobook").FontSize(20).Bold()
		if reader.err != nil {
			ui.Text(c, reader.err.Error()).FontSize(13).TextColor(theme.TextMuted).TextAlign(ui.Center).MaxWidth(560)
		}
		ui.Row(c).Gap(8).Children(func() {
			if ui.PrimaryButton(c, "Try again").Clicked() {
				a.retryBook()
			}
			if ui.Button(c, "Library").Clicked() {
				a.goLibrary()
			}
		})
	})
}

func (a *App) readerBody(c *ui.Context) {
	reader := &a.reader
	opened := reader.session
	if opened == nil {
		return
	}
	content := opened.content
	if len(content.Chapters) == 0 {
		return
	}

	position := opened.playback.Position()
	chapter := content.ChapterAt(position)
	if chapter != reader.chapter {
		reader.chapter = chapter
		reader.lastParagraph = -1
		reader.list.ScrollTo(0, ui.Start)
	}
	if reader.chapter < 0 || reader.chapter >= len(content.Chapters) {
		reader.chapter = 0
	}

	active := content.ActiveToken(position)
	var activeRef *readalong.FlatToken
	if active >= 0 && active < len(content.Flat) {
		activeRef = &content.Flat[active]
		if reader.follow && activeRef.Chapter == reader.chapter && activeRef.Paragraph != reader.lastParagraph {
			reader.list.ScrollTo(activeRef.Paragraph, ui.Center)
			reader.lastParagraph = activeRef.Paragraph
		}
	}

	reader.list.Selected = nil
	reader.list.Key = func(row int) any { return reader.chapter*1_000_000 + row }
	reader.list.Label = func(row int) string {
		paragraphs := content.Chapters[reader.chapter].Paragraphs
		if row < 0 || row >= len(paragraphs) {
			return ""
		}
		var text strings.Builder
		for _, token := range paragraphs[row].Tokens {
			if text.Len() > 0 {
				text.WriteByte(' ')
			}
			text.WriteString(token.Text)
			if text.Len() > 90 {
				break
			}
		}
		return text.String()
	}

	paragraphs := content.Chapters[reader.chapter].Paragraphs
	theme := c.Theme()
	ui.Column(c).Grow(1).AlignItems(ui.Center).Children(func() {
		chapter := content.Chapters[reader.chapter]
		if chapter.Rejected {
			ui.Text(c, "EPUB synchronisation is unavailable for this chapter. Showing the audiobook transcript.").
				FontSize(12).TextColor(theme.TextMuted).TextAlign(ui.Center).MaxWidth(620).Padding(24, 10)
		}
		if reader.importErr != nil {
			ui.Text(c, reader.importErr.Error()).
				FontSize(12).TextColor(theme.Danger).TextAlign(ui.Center).MaxWidth(620).Padding(24, 0)
		}
		if reader.removeErr != nil {
			ui.Text(c, reader.removeErr.Error()).
				FontSize(12).TextColor(theme.Danger).TextAlign(ui.Center).MaxWidth(620).Padding(24, 0)
		}
		ui.List(c, &reader.list, len(paragraphs), func(row int) {
			a.paragraphRow(c, row, activeRef)
		}).Grow(1).WidthPercent(100).MaxWidth(760).Padding(24, 0)
	})
}

func (a *App) paragraphRow(c *ui.Context, row int, active *readalong.FlatToken) {
	reader := &a.reader
	if reader.session == nil {
		return
	}
	paragraphs := reader.session.content.Chapters[reader.chapter].Paragraphs
	if row < 0 || row >= len(paragraphs) {
		return
	}

	paragraph := paragraphs[row]
	theme := c.Theme()
	activeToken := -1
	if active != nil && active.Chapter == reader.chapter && active.Paragraph == row {
		activeToken = active.Token
	}

	spans := make([]ui.Span, len(paragraph.Tokens))
	for index, token := range paragraph.Tokens {
		text := token.Text
		if index+1 < len(paragraph.Tokens) {
			text += " "
		}
		span := ui.Span{Text: text}
		if index == activeToken {
			span.Background = ui.RGBA(theme.Accent.R, theme.Accent.G, theme.Accent.B, 0.26)
			span.Weight = 700
		}
		spans[index] = span
	}

	ui.Column(c).Padding(0, 9).Children(func() {
		ui.RichText(c, spans...).FontSize(18).LineHeight(1.65)
	})
}

func (a *App) readerFooter(c *ui.Context) {
	reader := &a.reader
	opened := reader.session
	if opened == nil {
		return
	}
	theme := c.Theme()

	position := opened.playback.Position()
	duration := opened.duration
	if duration <= 0 {
		duration = opened.playback.Duration()
	}
	playing := opened.playback.Playing()

	if playing {
		c.Invalidate()
		if time.Since(reader.lastSave) >= 5*time.Second {
			a.saveProgress(opened)
			reader.lastSave = time.Now()
		}
	}
	if err := opened.playback.Err(); err != nil {
		reader.playbackErr = err
	}

	ui.Column(c).Padding(20, 12).Gap(10).Children(func() {
		ui.Divider(c)
		ui.Row(c).Gap(16).AlignItems(ui.Center).Children(func() {
			label := "Play"
			if playing {
				label = "Pause"
			}
			if ui.PrimaryButton(c, label).Clicked() {
				if err := opened.playback.Toggle(); err != nil {
					reader.playbackErr = err
				} else {
					reader.playbackErr = nil
				}
				a.saveProgress(opened)
			}

			ui.Text(c, formatClock(position)+" / "+formatClock(duration)).
				FontSize(12).TextColor(theme.TextMuted)

			if !reader.scrubbing {
				reader.seekValue = position.Seconds()
			}
			slider := ui.Slider(c, &reader.seekValue, 0, math.Max(duration.Seconds(), 1)).Grow(1)
			if slider.Changed() {
				reader.scrubbing = true
			}
			if reader.scrubbing && !slider.Dragging() {
				opened.playback.Seek(time.Duration(reader.seekValue * float64(time.Second)))
				a.saveProgress(opened)
				reader.scrubbing = false
				reader.lastParagraph = -1
			}

			segmented := ui.Segmented(c, &reader.speedIndex, speedLabels...).Label("Speed")
			if segmented.Changed() {
				opened.playback.SetSpeed(speeds[reader.speedIndex])
				a.saveProgress(opened)
			}

			follow := ui.Switch(c, &reader.follow).Label("Follow")
			if follow.Changed() && reader.follow {
				reader.lastParagraph = -1
			}
		})

		if reader.playbackErr != nil {
			ui.Text(c, reader.playbackErr.Error()).FontSize(12).TextColor(theme.Danger)
		}
	})
}

// confirmRemoveView asks before detaching the attached ebook. The audiobook
// and the listener's place in it are untouched.
func (a *App) confirmRemoveView(c *ui.Context) {
	theme := c.Theme()
	ui.Column(c).Width(420).Gap(12).Children(func() {
		ui.Text(c, "Remove the ebook?").FontSize(18).Bold()
		ui.Text(c, "The audiobook and your place in it stay. You can import the EPUB again later.").
			FontSize(13).TextColor(theme.TextMuted)
		ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
			if ui.Button(c, "Cancel").Clicked() {
				a.reader.confirmRemove = false
			}
			if ui.PrimaryButton(c, "Remove ebook").Clicked() {
				a.reader.confirmRemove = false
				if err := a.removeEPUB(a.reader.bookID); err != nil {
					a.reader.removeErr = err
				}
			}
		})
	})
}
