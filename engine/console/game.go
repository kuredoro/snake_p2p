package console

import (
	"fmt"
	"github.com/gdamore/tcell/v2"
	"github.com/kuredoro/snake_p2p/core"
	"log"
	"math/rand"
	"os"
	"time"
)

type Snake struct {
	id    int         		// snakep2p id
	body  []core.Coord     	// coordinates of snakep2p's body
	head  core.Coord      	// coordinates of snakep2p's head
	style tcell.Style 		// snakep2p's style
}

type Game struct {
	Ch             chan interface{} // communication channel
	snakes         []Snake          // snakes' state: alive snakes with ID, head and body coordinates
	food           []core.Coord     // food state: coordinates of food on the field
	numAliveSnakes int              // number of alive snakes in the game
}

type Boundary struct {
	topLeft     core.Coord
	bottomRight core.Coord
}

func (boundary Boundary) Contains(coord core.Coord) bool {
	return (coord.X <= boundary.topLeft.X || coord.X >= boundary.bottomRight.X) ||
		(coord.Y <= boundary.topLeft.Y || coord.Y >= boundary.bottomRight.Y)
}

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

func drawSnake(s tcell.Screen, snake Snake, boundary Boundary) error {
	if boundary.Contains(snake.head) {
		return fmt.Errorf("snakep2p's head coordinates are out of boundary")
	}
	s.SetContent(snake.head.X, snake.head.Y, tcell.RuneBullet, nil, snake.style)
	for _, point := range snake.body {
		if boundary.Contains(point) {
			return fmt.Errorf("snakep2p's body coordinates are out of boundary")
		}
		s.SetContent(point.X, point.Y, tcell.RuneBlock, nil, snake.style)
	}
	return nil
}

func drawFood(s tcell.Screen, food core.Coord, style tcell.Style, boundary Boundary) error {
	if boundary.Contains(food) {
		return fmt.Errorf("food coordinates are out of boundary")
	}
	s.SetContent(food.X, food.Y, '*', nil, style)
	return nil
}

func drawGridCell(s tcell.Screen, cell core.Coord, style tcell.Style, boundary Boundary) error {
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

var defColors = map[tcell.Color]struct{}{
	tcell.ColorReset: {},
	tcell.ColorWhite: {},
	tcell.ColorPurple: {},
	tcell.ColorRed: {},
}

func (game *Game) handleGameEvent(event interface{}) {
	switch event := event.(type) {
	case core.PlayerStarts:
		(*game).numAliveSnakes = len(event.Players)
		for id, start := range event.Players {
			(*game).snakes = append((*game).snakes, Snake{id: id, head: start, style: genSnakeStyle(defColors)})
		}
	case core.NewFood:
		(*game).food = append((*game).food, event.Pos)
	case core.Tick:
	}
}

func (game *Game) RunGame() {
	// Define Game styles
	defStyle := tcell.StyleDefault.Background(tcell.ColorReset).Foreground(tcell.ColorReset)
	boxStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorPurple)
	foodStyle := tcell.StyleDefault.Foreground(tcell.ColorRed).Background(tcell.ColorPurple)

	// Define Game field
	boundary := Boundary{core.Coord{1, 1}, core.Coord{81, 41}}

	// Initialize Game Screen
	s, err := tcell.NewScreen()
	if err != nil {
		log.Fatalf("%+v", err)
	}
	if err := s.Init(); err != nil {
		log.Fatalf("%+v", err)
	}
	s.EnableMouse()
	s.EnablePaste()
	s.Clear()
	s.SetStyle(defStyle)

	// Define function to quit the Game
	quit := func() {
		s.Fini()
		os.Exit(0)
	}

	// Game loop
	for {
		after := time.After(20 * time.Millisecond)  		// update Game Screen every 20 milliseconds
		// Process Game event
protocolEvents:
		for {
			select {
			case event, ok := <- (*game).Ch:
				if !ok {
					panic("Channel is closed")
				}

				(*game).handleGameEvent(event)
			case <-after:
				break protocolEvents
			}
		}

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

		// Draw Game state
		drawInitialBox(s, boundary, boxStyle)
		for _, snake := range (*game).snakes {
			err := drawSnake(s, snake, boundary)
			if err != nil {
				log.Fatalf("%+v", err)
			}
		}
		for _, f := range (*game).food {
			err := drawFood(s, f, foodStyle, boundary)
			if err != nil {
				println(err)
				log.Fatalf("%+v", err)
			}
		}
		s.Show()
	}
}