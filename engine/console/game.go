package console

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/kuredoro/snake_p2p/core"
	//"github.com/sanity-io/litter"
)

type Snake struct {
	Alive bool
	Body  []core.Coord
	Head  core.Coord
	Style tcell.Style
}

type Game struct {
	Ch          chan interface{}
	Snakes      map[int]*Snake
	Food        map[int]core.Coord
	AliveSnakes int
	Over        bool
	WinnerID    int
}

func GameInit(ch chan interface{}) *Game {
	game := &Game{
		Ch:          ch,
		Food:        make(map[int]core.Coord),
		Snakes:      make(map[int]*Snake),
		AliveSnakes: 0,
		Over:        false,
		WinnerID:    -1,
	}
	return game
}

type Boundary struct {
	TopLeft     core.Coord
	BottomRight core.Coord
}

func (boundary Boundary) Contains(coord core.Coord) bool {
	return (coord.X <= boundary.TopLeft.X || coord.X >= boundary.BottomRight.X) ||
		(coord.Y <= boundary.TopLeft.Y || coord.Y >= boundary.BottomRight.Y)
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

func drawBox(s tcell.Screen, boundary Boundary, style tcell.Style) {
	x1, y1 := boundary.TopLeft.X, boundary.TopLeft.Y
	x2, y2 := boundary.BottomRight.X, boundary.BottomRight.Y
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

	// drawText(s, x1+1, y1+1, x2-1, y2-1, style, text)
}

func drawSnake(s tcell.Screen, ID int, snake *Snake, boundary Boundary) error {
	if boundary.Contains(snake.Head) {
		return fmt.Errorf("snakep2p's head coordinates (%d, %d) are out of boundary", snake.Head.X, snake.Head.Y)
	}
	id := []rune(strconv.Itoa(ID))
	s.SetContent(snake.Head.X, snake.Head.Y, id[0], nil, snake.Style)
	for _, point := range snake.Body {
		if boundary.Contains(point) {
			return fmt.Errorf("snakep2p's body coordinates are out of boundary")
		}
		s.SetContent(point.X, point.Y, tcell.RuneBlock, nil, snake.Style)
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
	tcell.ColorReset:  {},
	tcell.ColorWhite:  {},
	tcell.ColorPurple: {},
	tcell.ColorRed:    {},
}

func (game *Game) handleGameEvent(event interface{}) {
	switch event := event.(type) {
	case core.PlayerStarts:
		game.AliveSnakes = len(event.Players)
		for id, start := range event.Players {
			game.Snakes[id] = &Snake{Alive: true, Head: start, Style: genSnakeStyle(defColors)}
		}
	case core.NewFood:
		game.Food[event.FoodID] = event.Pos
	case core.PlayerMove:
		for id, dir := range event.Moves {
			prevHead := game.Snakes[id].Head
			// move snake's head
			switch dir {
			case core.Up:
				game.Snakes[id].Head.Y--
			case core.Left:
				game.Snakes[id].Head.X--
			case core.Right:
				game.Snakes[id].Head.X++
			case core.Down:
				game.Snakes[id].Head.Y++
			default:
				panic("the value of direction is unknown")
			}

			// move snakes body
			if len(game.Snakes[id].Body) == 0 {
				continue
			}
			for i := len(game.Snakes[id].Body) - 1; i > 0; i-- {
				game.Snakes[id].Body[i] = game.Snakes[id].Body[i-1]
			}
			game.Snakes[id].Body[0] = prevHead
		}
	case core.FoodEaten:
		delete(game.Food, event.FoodID)
	case core.PushSegment:
		game.Snakes[event.SnakeID].Body = append(game.Snakes[event.SnakeID].Body, event.Pos)
	case core.PlayerDied:
		game.Snakes[event.SnakeID].Alive = false
	case core.GameOver:
		game.Over = true
		if event.Successful {
			game.WinnerID = event.Winner
		}
	case core.Tick:
	}
}

func (game *Game) RunGame() {
	// Define Game styles
	defStyle := tcell.StyleDefault.Background(tcell.ColorReset).Foreground(tcell.ColorReset)
	boxStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorPurple)
	blackBoxStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack)
	foodStyle := tcell.StyleDefault.Foreground(tcell.ColorRed).Background(tcell.ColorPurple)

	// Define Game field
	boundary := Boundary{core.Coord{X: 1, Y: 1}, core.Coord{X: 81, Y: 41}}

	// Initialize Game Screen
	s, err := tcell.NewScreen()
	if err != nil {
		log.Fatalf("%+v", err)
	}
	if err := s.Init(); err != nil {
		log.Fatalf("%+v", err)
	}
	s.DisableMouse()
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
		after := time.After(20 * time.Millisecond) // update Game Screen every 20 milliseconds
		// Process Game event
	protocolEvents:
		for {
			select {
			case event, ok := <-game.Ch:
				if !ok {
					// s.Fini()
					panic("Channel is closed")
				}

				game.handleGameEvent(event)

				/*
				   fmt.Printf("EVENT %#v\n", event)
				   litter.Dump(game)
				*/
			case <-after:
				break protocolEvents
			}
		}

		if s.HasPendingEvent() {
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

		// Draw Game state
		//s.Clear()
		if game.Over {
			drawBox(s, boundary, boxStyle)
			width, height := 12, 0
			if game.WinnerID != -1 {
				height = 4
			} else {
				height = 2
			}
			x1 := (boundary.BottomRight.X - boundary.TopLeft.X - width) / 2
			y1 := (boundary.BottomRight.Y - boundary.TopLeft.Y - height) / 2
			x2 := (boundary.BottomRight.X - boundary.TopLeft.X + width) / 2
			y2 := (boundary.BottomRight.Y - boundary.TopLeft.Y + height) / 2
			drawBox(s, Boundary{core.Coord{X: x1, Y: y1}, core.Coord{X: x2, Y: y2}}, blackBoxStyle)
			drawText(s, x1+1, y1+1, x2-1, y2-1, blackBoxStyle, "Game Over")
			if game.WinnerID != -1 {
				text := "WinnerID " + strconv.Itoa(game.WinnerID)
				drawText(s, x1+1, y1+3, x2-1, y2-1, blackBoxStyle, text)
			}
			s.Show()
			continue
		}
		drawBox(s, boundary, boxStyle)
		for id, snake := range game.Snakes {
			if !snake.Alive {
				continue
			}
			err := drawSnake(s, id, snake, boundary)
			if err != nil {
				s.Fini()
				log.Fatalf("%+v", err)
				os.Exit(0)
			}
		}
		for _, f := range game.Food {
			err := drawFood(s, f, foodStyle, boundary)
			if err != nil {
				s.Fini()
				fmt.Println(err)
				log.Fatalf("%+v", err)
				os.Exit(0)
			}
		}
		s.Show()
	}
}
