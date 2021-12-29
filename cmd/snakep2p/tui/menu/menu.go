package menu

import "github.com/gdamore/tcell/v2"

var (
	backgroundStyleDefault = tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorWhite)
	buttonStyleDefault     = tcell.StyleDefault.Background(tcell.ColorLightCyan).Foreground(tcell.ColorBlack)
)

type MenuItem struct {
	Hotkey string
	Name   string
	Action func()
}

type Menu struct{}

func New(foo interface{}) *Menu {
	return &Menu{}
}

func (m *Menu) Draw(s tcell.Screen) {
	return
}

func (m *Menu) GetBackgroundStyle() tcell.Style {
	return backgroundStyleDefault
}

func (m *Menu) GetButtonStyle() tcell.Style {
	return buttonStyleDefault
}
