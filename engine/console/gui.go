package console

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const (
	welcome    = `Welcome to the multiplayer snake game.`
	navigation = `Choose one of the following actions`
)

func Cover() (content tview.Primitive) {
	// Create a frame for the subtitle and navigation infos.
	frame := tview.NewFrame(tview.NewBox()).
		SetBorders(0, 0, 0, 0, 0, 0).
		AddText(welcome, true, tview.AlignCenter, tcell.ColorGreen).
		AddText("", true, tview.AlignCenter, tcell.ColorWhite).
		AddText(navigation, true, tview.AlignCenter, tcell.ColorDarkMagenta)

	createGamebtn := tview.NewButton("Create game").SetSelectedFunc(func() {})
	joinGamebtn := tview.NewButton("Join game").SetSelectedFunc(func() {})

	// Create a Flex layout that centers the logo and subtitle.
	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(tview.NewBox(), 0, 5, false).
		AddItem(frame, 0, 1, false).
		AddItem(tview.NewFlex().
			SetDirection(tview.FlexColumn).
			AddItem(tview.NewBox(), 0, 1, false).
			AddItem(createGamebtn, 20, 1, true).
			AddItem(joinGamebtn, 20, 1, true).
			AddItem(tview.NewBox(), 0, 1, false), 1, 1, true).
		AddItem(tview.NewBox(), 0, 5, false)
	return flex
}
