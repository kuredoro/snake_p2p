package main

import (
	"github.com/kuredoro/snake_p2p/engine/console"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// The application.
var app = tview.NewApplication()

func main() {
	// Shortcuts to navigate the slides.
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyCtrlN {
			//nextSlide()
			return nil
		} else if event.Key() == tcell.KeyCtrlP {
			//previousSlide()
			return nil
		} else if event.Key() == tcell.KeyEscape {
			console.Cover()
			return nil
		}
		return event
	})

	// Start the application.
	if err := app.SetRoot(console.Cover(), true).EnableMouse(true).Run(); err != nil {
		panic(err)
	}
}
