package menu

import (
	"fmt"

	"code.rocketnine.space/tslocum/cview"
	"github.com/gdamore/tcell/v2"
)

var (
	backgroundStyleDefault = tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorWhite)
	buttonStyleDefault     = tcell.StyleDefault.Background(tcell.ColorLightCyan).Foreground(tcell.ColorBlack)
)

type labelColorable interface {
	SetLabelColor(tcell.Color)
}

type textColorable interface {
	SetTextColor(tcell.Color)
}

type backgroundColorable interface {
	SetBackgroundColor(tcell.Color)
}

type textBackgroundColorable interface {
	textColorable
	backgroundColorable
}

type labelBackgroundColorable interface {
	labelColorable
	backgroundColorable
}

func setTextBackgroundStyle(p textBackgroundColorable, s tcell.Style) {
	fg, bg, _ := s.Decompose()
	p.SetTextColor(fg)
	p.SetBackgroundColor(bg)
}

func setLabelBackgroundStyle(p labelBackgroundColorable, s tcell.Style) {
	fg, bg, _ := s.Decompose()
	p.SetLabelColor(fg)
	p.SetBackgroundColor(bg)
}

type MenuItem struct {
	Hotkey string
	Name   string
	Action func()
}

type Menu struct {
	hotkeyLabel *cview.TextView
	button      *cview.TextView
}

func New(items []MenuItem) *Menu {
	if len(items) == 0 {
		return &Menu{}
	}

	hotkeyLabel := cview.NewTextView()
	setTextBackgroundStyle(hotkeyLabel, backgroundStyleDefault)
	hotkeyLabel.SetText(items[0].Hotkey)
	hotkeyLabel.SetRect(0, 0, len(items[0].Hotkey), 1)

	button := cview.NewTextView()
	button.SetText(items[0].Name)
	setTextBackgroundStyle(button, buttonStyleDefault)

	return &Menu{
		hotkeyLabel: hotkeyLabel,
		button:      button,
	}
}

func (m *Menu) SetRect(x, y, w, h int) {
	if m.hotkeyLabel == nil || m.button == nil {
		return
	}

	start := len(m.hotkeyLabel.GetText(true))
	fmt.Println(start)
	m.button.SetRect(start, 0, w-start, 1)
}

func (m *Menu) Draw(s tcell.Screen) {
	if m.hotkeyLabel == nil || m.button == nil {
		return
	}

	m.hotkeyLabel.Draw(s)
	m.button.Draw(s)
}

func (m *Menu) GetBackgroundStyle() tcell.Style {
	return backgroundStyleDefault
}

func (m *Menu) GetButtonStyle() tcell.Style {
	return buttonStyleDefault
}
