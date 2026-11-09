package app

import "github.com/egoist/mygo/ui"

// loadingView is the first screen: the app finding its books.
func (a *App) loadingView(c *ui.Context) {
	theme := c.Theme()
	ui.Column(c).Fill().Background(theme.Background).Center().Gap(14).Padding(32).Children(func() {
		ui.Text(c, "Amba").FontSize(42).Bold()
		ui.Text(c, "Listen and read along").FontSize(15).TextColor(theme.TextMuted)
		ui.Spinner(c).Width(28).Height(28)
		ui.Text(c, a.scanStage).FontSize(13).TextColor(theme.TextMuted)
	})
}

// notFoundView is shown when a page's address does not name anything.
func (a *App) notFoundView(c *ui.Context) {
	theme := c.Theme()
	ui.Column(c).Fill().Background(theme.Background).Center().Gap(12).Padding(32).Children(func() {
		ui.Text(c, "Page not found").FontSize(22).Bold()
		if ui.PrimaryButton(c, "Back to library").Clicked() {
			a.goLibrary()
		}
	})
}
