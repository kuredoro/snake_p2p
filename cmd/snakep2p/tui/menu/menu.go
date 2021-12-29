package menu

import "github.com/gdamore/tcell/v2"

type Menu struct{}

func New(foo interface{}) *Menu {
	return &Menu{}
}

func (m *Menu) Draw(s tcell.Screen) {
	return
}
