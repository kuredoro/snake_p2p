package main

import (
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/gdamore/tcell/v2"
)

func drawText(s tcell.Screen, x1, y1, x2, y2 int, style tcell.Style, text string) {
	row := y1
	col := x1
	for _, r := range []rune(text) {
		s.SetContent(col, row, r, nil, style)
		col++
		if col >= x2 {
			row++
			col = x1
		}
		if row > y2 {
			break
		}
	}
}

func drawInitialBox(s tcell.Screen, x1, y1, x2, y2 int, style tcell.Style) {
	if y2 < y1 {
		y1, y2 = y2, y1
	}
	if x2 < x1 {
		x1, x2 = x2, x1
	}

	// Fill background
	for row := y1; row <= y2; row++ {
		for col := x1; col <= x2; col++ {
			s.SetContent(col, row, tcell.RuneBullet, nil, style)
		}
	}

	// Draw borders
	for col := x1; col <= x2; col++ {
		s.SetContent(col, y1, tcell.RuneHLine, nil, style)
		s.SetContent(col, y2, tcell.RuneHLine, nil, style)
	}
	for row := y1 + 1; row < y2; row++ {
		s.SetContent(x1, row, tcell.RuneVLine, nil, style)
		s.SetContent(x2, row, tcell.RuneVLine, nil, style)
	}

	// Only draw corners if necessary
	if y1 != y2 && x1 != x2 {
		s.SetContent(x1, y1, tcell.RuneULCorner, nil, style)
		s.SetContent(x2, y1, tcell.RuneURCorner, nil, style)
		s.SetContent(x1, y2, tcell.RuneLLCorner, nil, style)
		s.SetContent(x2, y2, tcell.RuneLRCorner, nil, style)
	}

	//drawText(s, x1+1, y1+1, x2-1, y2-1, style, text)
}

type Pos struct {
	x, y int
}

type Snake struct {
	id    int         // snake id
	body  []Pos       // coordinates of snake's body
	head  Pos         // coordinates of snake's head
	style tcell.Style // snake's style
}

type Boundary struct {
	x1, y1, x2, y2 int
}

// TODO: add out of boundary error handling
func drawSnake(s tcell.Screen, snake Snake, boundary Boundary) {
	s.SetContent(snake.head.x, snake.head.y, tcell.RuneBullet, nil, snake.style)
	for _, p := range snake.body {
		s.SetContent(p.x, p.y, tcell.RuneBlock, nil, snake.style)
	}
}

func getRandColor(defColors map[tcell.Color]struct{}) tcell.Color {
	color := tcell.PaletteColor(rand.Intn(256))
	_, ok := defColors[color]
	for ok == true {
		color = tcell.PaletteColor(rand.Intn(256))
		_, ok = defColors[color]
	}
	return color
}

func genSnakeStyle(defColors map[tcell.Color]struct{}) tcell.Style {
	style := tcell.StyleDefault.Foreground(getRandColor(defColors)).Background(getRandColor(defColors))
	return style
}

func main() {
	rand.Seed(time.Now().UnixNano())
	defColors := make(map[tcell.Color]struct{})
	defStyle := tcell.StyleDefault.Background(tcell.ColorReset).Foreground(tcell.ColorReset)
	boxStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorPurple)
	defColors[tcell.ColorReset] = struct{}{}
	defColors[tcell.ColorWhite] = struct{}{}
	defColors[tcell.ColorPurple] = struct{}{}
	// Initialize screen
	s, err := tcell.NewScreen()
	if err != nil {
		log.Fatalf("%+v", err)
	}
	if err := s.Init(); err != nil {
		log.Fatalf("%+v", err)
	}
	s.SetStyle(defStyle)
	s.EnableMouse()
	s.EnablePaste()
	s.Clear()

	// Draw initial boxes
	drawInitialBox(s, 1, 1, 81, 41, boxStyle)
	//drawBox(s, 5, 9, 32, 14, boxStyle, "Press C to reset")
	snake := Snake{id: 0,
		head:  Pos{4, 5},
		body:  []Pos{{5, 5}, {6, 5}},
		style: genSnakeStyle(defColors)}
	drawSnake(s, snake, Boundary{1, 1, 81, 41})
	// Event loop
	//ox, oy := -1, -1
	quit := func() {
		s.Fini()
		os.Exit(0)
	}

	for {
		// Update screen
		s.Show()

		// Poll event
		ev := s.PollEvent()

		// Process event
		switch ev := ev.(type) {
		case *tcell.EventResize:
			s.Sync()
		case *tcell.EventKey:
			if ev.Key() == tcell.KeyEscape || ev.Key() == tcell.KeyCtrlC {
				quit()
			}
			//else if ev.Key() == tcell.KeyCtrlL {
			//	s.Sync()
			//} else if ev.Rune() == 'C' || ev.Rune() == 'c' {
			//	s.Clear()
			//}
			//case *tcell.EventMouse:
			//	x, y := ev.Position()
			//	button := ev.Buttons()
			//	// Only process button events, not wheel events
			//	button &= tcell.ButtonMask(0xff)
			//
			//	if button != tcell.ButtonNone && ox < 0 {
			//		ox, oy = x, y
			//	}
			//	switch ev.Buttons() {
			//	case tcell.ButtonNone:
			//		if ox >= 0 {
			//			label := fmt.Sprintf("%d,%d to %d,%d", ox, oy, x, y)
			//			drawBox(s, ox, oy, x, y, boxStyle, label)
			//			ox, oy = -1, -1
			//		}
			//	}
		}
	}
}
