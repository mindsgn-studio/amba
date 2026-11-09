// Amba is a read-along player: it shows an audiobook's words as they are
// narrated, and keeps the listener's place between sessions.
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/mindsgn-studio/amba/internal/app"
)

func main() {
	application, err := app.New(app.DefaultConfig())
	if err != nil {
		log.Fatal(err)
	}

	mygo.App.WhenReady(func() {
		window := mygo.NewWindow(mygo.WindowOptions{
			Title:     "Amba",
			Width:     1040,
			Height:    780,
			MinWidth:  760,
			MinHeight: 560,
			Content:   ui.View(application.View),
		})
		application.Attach(window)
		window.Show()
		application.Start()
	})

	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
