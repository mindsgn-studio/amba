package app

import (
	"fmt"

	"github.com/egoist/mygo/ui"

	"github.com/mindsgn-studio/amba/internal/readalong"
)

// toggleReview shows or hides the attached ebook, opening at the chapter the
// reader is hearing.
func (a *App) toggleReview() {
	reader := &a.reader
	if reader.session == nil || reader.session.review == nil {
		reader.drawerOpen = false
		return
	}
	reader.drawerOpen = !reader.drawerOpen
	if reader.drawerOpen {
		reader.reviewPage = reader.session.review.PageForChapter(reader.chapter)
	}
}

// turnReviewPage moves the drawer by delta pages, staying inside the book.
func (a *App) turnReviewPage(delta int) {
	reader := &a.reader
	if reader.session == nil || reader.session.review == nil {
		return
	}
	pages := len(reader.session.review.Pages)
	if pages == 0 {
		return
	}
	page := reader.reviewPage + delta
	switch {
	case page < 0:
		page = 0
	case page >= pages:
		page = pages - 1
	}
	reader.reviewPage = page
}

// reviewDrawer is the attached ebook beside the reader: the printed words,
// green where the narration reads them and red where it does not.
func (a *App) reviewDrawer(c *ui.Context) {
	reader := &a.reader
	opened := reader.session
	if opened == nil || opened.review == nil || len(opened.review.Pages) == 0 {
		return
	}

	review := opened.review
	pageIndex := reader.reviewPage
	if pageIndex < 0 {
		pageIndex = 0
	}
	if pageIndex >= len(review.Pages) {
		pageIndex = len(review.Pages) - 1
	}
	reader.reviewPage = pageIndex
	page := review.Pages[pageIndex]
	theme := c.Theme()

	ui.Column(c).Width(380).Shrink(0).Background(theme.Surface).Children(func() {
		ui.Row(c).Padding(16, 12).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Column(c).Grow(1).Children(func() {
				ui.Text(c, "Ebook").FontSize(16).Bold()
				ui.Text(c, fmt.Sprintf("%d%% synced · %d of %d words",
					int(review.Percent()*100), review.Synced, review.Total)).
					FontSize(12).TextColor(theme.TextMuted)
			})
			if ui.Button(c, "Hide").Clicked() {
				reader.drawerOpen = false
			}
		})
		ui.Divider(c)
		ui.Row(c).Padding(16, 8).Gap(16).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "● Synced").FontSize(12).TextColor(theme.Success)
			ui.Text(c, "● Unsynced").FontSize(12).TextColor(theme.Danger)
		})
		ui.Divider(c)
		ui.Column(c).Padding(16, 10).Children(func() {
			title := page.Title
			if title == "" {
				title = fmt.Sprintf("Chapter %d", page.Chapter+1)
			}
			ui.Text(c, title).FontSize(14).Bold()
			if !page.Accepted {
				ui.Text(c, "This chapter falls back to the audiobook transcript.").
					FontSize(12).TextColor(theme.TextMuted)
			}
		})
		ui.Divider(c)
		ui.Scroll(c).Grow(1).Children(func() {
			ui.Column(c).Gap(10).Padding(16, 12).Children(func() {
				for _, paragraph := range page.Paragraphs {
					if len(paragraph.Words) == 0 {
						continue
					}
					ui.RichText(c, reviewSpans(paragraph, theme)...).FontSize(15).LineHeight(1.6)
				}
			})
		})
		ui.Divider(c)
		ui.Column(c).Padding(16, 10).Gap(8).Children(func() {
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				if ui.Button(c, "Previous page").Disabled(pageIndex <= 0).Clicked() {
					a.turnReviewPage(-1)
				}
				ui.Text(c, fmt.Sprintf("Page %d of %d", pageIndex+1, len(review.Pages))).
					FontSize(12).TextColor(theme.TextMuted)
				if ui.Button(c, "Next page").Disabled(pageIndex >= len(review.Pages)-1).Clicked() {
					a.turnReviewPage(1)
				}
			})
			if ui.Button(c, "Remove EPUB").Clicked() {
				reader.confirmRemove = true
			}
		})
	})
}

// reviewSpans colors one paragraph of the drawer. Punctuation-only tokens stay
// muted because they carry no narration.
func reviewSpans(paragraph readalong.ReviewParagraph, theme *ui.Theme) []ui.Span {
	spans := make([]ui.Span, len(paragraph.Words))
	for index, word := range paragraph.Words {
		text := word.Text
		if index+1 < len(paragraph.Words) {
			text += " "
		}
		span := ui.Span{Text: text}
		switch {
		case word.Synced:
			span.Color = theme.Success
		case word.Word:
			span.Color = theme.Danger
		default:
			span.Color = theme.TextMuted
		}
		spans[index] = span
	}
	return spans
}

// detachEPUB rebuilds the open book from its transcript, which is already in
// memory, so removing an ebook never transcribes the audiobook again.
func (a *App) detachEPUB() {
	opened := a.reader.session
	if opened == nil {
		a.reader.drawerOpen = false
		a.reader.reviewPage = 0
		return
	}

	opened.book = a.reader.book
	if opened.transcript == nil {
		a.reader.drawerOpen = false
		a.reader.reviewPage = 0
		return
	}
	opened.content = readalong.FromTranscript(opened.transcript)
	opened.review = nil
	a.reader.chapter = opened.content.ChapterAt(opened.playback.Position())
	a.reader.lastParagraph = -1
	a.reader.list = ui.ListState{}
	a.reader.list.ScrollTo(0, ui.Start)
	a.reader.drawerOpen = false
	a.reader.reviewPage = 0
}
