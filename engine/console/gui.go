package console

import (
	"github.com/rivo/tview"
)

func TviewTest() {
	box := tview.NewBox().SetBorder(true).SetTitle("Hello, let's play snake game :)")
	if err := tview.NewApplication().SetRoot(box, true).Run(); err != nil {
		panic(err)
	}
}
