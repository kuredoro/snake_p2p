package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/gdamore/tcell/v2"
	. "github.com/kuredoro/snake_p2p/core"
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

func drawInitialBox(s tcell.Screen, boundary Boundary, style tcell.Style) {
	x1, y1 := boundary.topLeft.X, boundary.topLeft.Y
	x2, y2 := boundary.bottomRight.X, boundary.bottomRight.Y
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

type Snake struct {
	id    int         // snake id
	body  []Coord     // coordinates of snake's body
	head  Coord       // coordinates of snake's head
	style tcell.Style // snake's style
}

type Boundary struct {
	topLeft     Coord
	bottomRight Coord
}

func (boundary Boundary) Contains(coord Coord) bool {
	return (coord.X <= boundary.topLeft.X || coord.X >= boundary.bottomRight.X) ||
		(coord.Y <= boundary.topLeft.Y || coord.Y >= boundary.bottomRight.Y)
}

func drawSnake(s tcell.Screen, snake Snake, boundary Boundary) error {
	if boundary.Contains(snake.head) {
		return fmt.Errorf("snake's head coordinates are out of boundary")
	}
	s.SetContent(snake.head.X, snake.head.Y, tcell.RuneBullet, nil, snake.style)
	for _, point := range snake.body {
		if boundary.Contains(point) {
			return fmt.Errorf("snake's body coordinates are out of boundary")
		}
		s.SetContent(point.X, point.Y, tcell.RuneBlock, nil, snake.style)
	}
	return nil
}

func drawFood(s tcell.Screen, food Coord, style tcell.Style, boundary Boundary) error {
	if boundary.Contains(food) {
		return fmt.Errorf("food coordinates are out of boundary")
	}
	s.SetContent(food.X, food.Y, '*', nil, style)
	return nil
}

func drawGridCell(s tcell.Screen, cell Coord, style tcell.Style, boundary Boundary) error {
	if boundary.Contains(cell) {
		return fmt.Errorf("cell is out of boundary")
	}
	s.SetContent(cell.X, cell.Y, tcell.RuneBullet, nil, style)
	return nil
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
	foodStyle := tcell.StyleDefault.Foreground(tcell.ColorRed).Background(tcell.ColorPurple)
	defColors[tcell.ColorReset] = struct{}{}
	defColors[tcell.ColorWhite] = struct{}{}
	defColors[tcell.ColorPurple] = struct{}{}
	defColors[tcell.ColorRed] = struct{}{}
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

	// Draw initial grid
	boundary := Boundary{Coord{1, 1}, Coord{81, 41}}
	drawInitialBox(s, boundary, boxStyle)
	// Draw snakes
	snake1 := Snake{id: 0,
		head:  Coord{4, 5},
		body:  []Coord{{5, 5}, {6, 5}},
		style: genSnakeStyle(defColors)}
	snake2 := Snake{id: 0,
		head:  Coord{12, 15},
		body:  []Coord{{12, 14}, {12, 13}},
		style: genSnakeStyle(defColors)}
	err = drawSnake(s, snake1, boundary)
	if err != nil {
		log.Fatalf("%+v", err)
	}
	err = drawSnake(s, snake2, boundary)
	if err != nil {
		log.Fatalf("%+v", err)
	}
	// new food
	food1 := Coord{20, 24}
	food2 := Coord{23, 35}
	err = drawFood(s, food1, foodStyle, boundary)
	if err != nil {
		log.Fatalf("%+v", err)
	}
	err = drawFood(s, food2, foodStyle, boundary)
	if err != nil {
		log.Fatalf("%+v", err)
	}
	s.Show()
	// food is eaten
	time.Sleep(10 * time.Second)
	err = drawGridCell(s, food1, boxStyle, boundary)
	if err != nil {
		log.Fatalf("%+v", err)
	}
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
		}
	}
}
