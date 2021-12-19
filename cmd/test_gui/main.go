package main

import (
	"github.com/kuredoro/snake_p2p/engine/console"

	//"github.com/rivo/tview"
)

func main() {
	gameUI := console.GatherUIInit()
	// Shortcuts to navigate the slides.
	//console.App.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
	//	if event.Key() == tcell.KeyCtrlN {
	//		//nextSlide()
	//		return nil
	//	} else if event.Key() == tcell.KeyCtrlP {
	//		//previousSlide()
	//		return nil
	//	} else if event.Key() == tcell.KeyEscape {
	//		console.Cover()
	//		return nil
	//	}
	//	return event
	//})

	// Start the application.
	err := gameUI.Run()
	if err != nil {
		panic("Game Run finished with error")
	}
}
