package app

import (
	"fmt"

	"github.com/egoist/mygo/ui"

	"github.com/mindsgn-studio/amba/internal/library"
)

// libraryView is the second screen: the books found on disk.
func (a *App) libraryView(c *ui.Context) {
	theme := c.Theme()
	ui.Column(c).Fill().Background(theme.Background).Children(func() {
		a.libraryHeader(c)
		ui.Divider(c)
		switch {
		case a.scanning && len(a.books) == 0:
			a.libraryLoading(c)
		case a.booksErr != nil:
			a.libraryError(c)
		case len(a.books) == 0:
			a.libraryEmpty(c)
		default:
			a.libraryBooks(c)
		}
	})
}

func (a *App) libraryHeader(c *ui.Context) {
	theme := c.Theme()
	ui.Row(c).Padding(24, 18).Gap(12).AlignItems(ui.Center).Children(func() {
		ui.Column(c).Grow(1).Children(func() {
			ui.Text(c, "Amba").FontSize(26).Bold()
			ui.Text(c, "Your audiobooks, with the words on screen").FontSize(13).TextColor(theme.TextMuted)
		})
		if a.scanning {
			ui.Spinner(c)
		}
		if ui.Button(c, "Refresh").Disabled(a.scanning).Clicked() {
			a.refresh(false)
		}
	})
}

func (a *App) libraryLoading(c *ui.Context) {
	theme := c.Theme()
	ui.Column(c).Fill().Center().Gap(12).Children(func() {
		ui.Spinner(c)
		ui.Text(c, a.scanStage).FontSize(13).TextColor(theme.TextMuted)
	})
}

func (a *App) libraryError(c *ui.Context) {
	theme := c.Theme()
	ui.Column(c).Fill().Center().Gap(12).Padding(32).Children(func() {
		ui.Text(c, "Your audiobooks could not be loaded").FontSize(20).Bold()
		ui.Text(c, a.booksErr.Error()).FontSize(13).TextColor(theme.TextMuted).TextAlign(ui.Center).MaxWidth(520)
		ui.Text(c, "Looked in "+a.config.LibraryRoot).FontSize(12).TextColor(theme.TextMuted).TextAlign(ui.Center)
		if ui.PrimaryButton(c, "Try again").Clicked() {
			a.refresh(false)
		}
	})
}

func (a *App) libraryEmpty(c *ui.Context) {
	theme := c.Theme()
	ui.Column(c).Fill().Center().Gap(10).Padding(32).Children(func() {
		ui.Text(c, "Your shelf is empty").FontSize(20).Bold()
		ui.Text(c, "Put audiobooks in "+a.config.LibraryRoot).FontSize(13).TextColor(theme.TextMuted).TextAlign(ui.Center)
		if ui.PrimaryButton(c, "Scan again").Clicked() {
			a.refresh(false)
		}
	})
}

func (a *App) libraryBooks(c *ui.Context) {
	a.libraryList.Selected = &a.libraryChoice
	a.libraryList.Key = func(row int) any { return a.books[row].ID }
	a.libraryList.Label = func(row int) string { return a.books[row].Title }

	ui.Column(c).Grow(1).Padding(24, 12, 24, 24).Children(func() {
		list := ui.List(c, &a.libraryList, len(a.books), func(row int) {
			a.bookRow(c, row)
		}).Grow(1)

		if list.Changed() || list.Submitted() {
			if a.libraryChoice >= 0 && a.libraryChoice < len(a.books) {
				book := a.books[a.libraryChoice]
				a.libraryChoice = -1
				a.openBook(book)
			}
		}
	})
}

func (a *App) bookRow(c *ui.Context, row int) {
	book := a.books[row]
	progress := a.progressCache[book.ID]
	theme := c.Theme()

	item := ui.Row(c).Gap(16).Padding(12).AlignItems(ui.Center).Radius(theme.Radius).Cursor(ui.CursorPointer)
	if item.Hovered() {
		item.Background(theme.SurfaceHover)
	} else {
		item.Background(theme.Surface)
	}

	item.Children(func() {
		if cover := a.covers[book.ID]; cover != nil {
			ui.Image(c, cover).Width(64).Height(96).Fit(ui.Cover).Radius(6)
		} else {
			ui.Column(c).Width(64).Height(96).Center().Background(theme.Background).Border(1, theme.Border).Radius(6).
				Children(func() {
					ui.Text(c, initials(book.Title)).FontSize(22).Bold().TextColor(theme.TextMuted)
				})
		}

		ui.Column(c).Grow(1).Gap(6).Children(func() {
			ui.Text(c, book.Title).FontSize(17).Bold()
			if book.Author != "" {
				ui.Text(c, book.Author).FontSize(13).TextColor(theme.TextMuted)
			}
			if progress.Started() {
				ui.Progress(c, progress.Fraction()).Height(5).WidthPercent(100).MaxWidth(420)
			}
			ui.Text(c, progressText(progress)).FontSize(12).TextColor(theme.TextMuted)
		})

		ui.Text(c, "›").FontSize(22).TextColor(theme.TextMuted)
	})
}

func progressText(progress library.Progress) string {
	switch {
	case !progress.Started():
		return "Not started"
	case progress.Finished():
		return "Finished"
	case progress.Duration > 0:
		return fmt.Sprintf(
			"%d%% · %s left",
			int(progress.Fraction()*100),
			formatDuration(progress.Duration-progress.Position),
		)
	default:
		return "In progress"
	}
}
