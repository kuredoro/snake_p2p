package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/kuredoro/snake_p2p/core"
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
	id    int         // snakep2p id
	body  []core.Coord     // coordinates of snakep2p's body
	head  core.Coord       // coordinates of snakep2p's head
	style tcell.Style // snakep2p's style
}

type Boundary struct {
	topLeft     core.Coord
	bottomRight core.Coord
}

func (boundary Boundary) Contains(coord core.Coord) bool {
	return (coord.X <= boundary.topLeft.X || coord.X >= boundary.bottomRight.X) ||
		(coord.Y <= boundary.topLeft.Y || coord.Y >= boundary.bottomRight.Y)
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

func handleEvent(event interface{}, s tcell.Screen, snakes *[]Snake, food *[]core.Coord) {
	switch event := event.(type) {
	case core.PlayerStarts:
		//numAliveSnakes = len(event)
		for id, start := range event.Players {
			*snakes = append(*snakes, Snake{id: id, head: start, style: genSnakeStyle(defColors)})
		}
	case core.NewFood:
		*food = append(*food, event.Pos)
	case core.Tick:
	}
}

var defColors = map[tcell.Color]struct{}{
	tcell.ColorReset: {},
	tcell.ColorWhite: {},
	tcell.ColorPurple: {},
	tcell.ColorRed: {},
}

func Run(ch chan interface{}) {
	boxStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorPurple)
	foodStyle := tcell.StyleDefault.Foreground(tcell.ColorRed).Background(tcell.ColorPurple)
	//s.SetStyle(defStyle)
	boundary := Boundary{core.Coord{1, 1}, core.Coord{81, 41}}
	var snakes []Snake
	var food []core.Coord
	//numAliveSnakes := 0
	// Initialize screen
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
	defStyle := tcell.StyleDefault.Background(tcell.ColorReset).Foreground(tcell.ColorReset)
	s.SetStyle(defStyle)
	// Draw initial grid
	//drawInitialBox(s, boundary, boxStyle)
	// Draw snakes
	//snake1 := Snake{id: 0,
	//	head:  core.Coord{4, 5},
	//	body:  []core.Coord{{5, 5}, {6, 5}},
	//	style: genSnakeStyle(defColors)}
	//snake2 := Snake{id: 0,
	//	head:  core.Coord{12, 15},
	//	body:  []core.Coord{{12, 14}, {12, 13}},
	//	style: genSnakeStyle(defColors)}
	//err = drawSnake(s, snake1, boundary)
	//if err != nil {
	//	log.Fatalf("%+v", err)
	//}
	//err = drawSnake(s, snake2, boundary)
	//if err != nil {
	//	log.Fatalf("%+v", err)
	//}
	// new food
	//food1 := core.Coord{20, 24}
	//food2 := core.Coord{23, 35}
	//err = drawFood(s, food1, foodStyle, boundary)
	//if err != nil {
	//	log.Fatalf("%+v", err)
	//}
	//err = drawFood(s, food2, foodStyle, boundary)
	//if err != nil {
	//	log.Fatalf("%+v", err)
	//}
	//s.Show()
	// food is eaten
	//time.Sleep(10 * time.Second)
	//err = drawGridCell(s, food1, boxStyle, boundary)
	//if err != nil {
	//	log.Fatalf("%+v", err)
	//}
	// Event loop
	quit := func() {
		s.Fini()
		os.Exit(0)
	}
	for {
		after := time.After(20 * time.Millisecond)
protocolEvents:
		for {
			select {
			case event, ok := <-ch:
				if !ok {
					panic("Channel is closed")
				}

				handleEvent(event, s, &snakes, &food)
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
			//else if ev.Key() == tcell.KeyCtrlL {
			//	s.Sync()
			//} else if ev.Rune() == 'C' || ev.Rune() == 'c' {
			//	s.Clear()
			//}
			//case *tcell.EventMouse:
			//	X, Y := ev.Position()
			//	button := ev.Buttons()
			//	// Only process button events, not wheel events
			//	button &= tcell.ButtonMask(0xff)
			//
			//	if button != tcell.ButtonNone && ox < 0 {
			//		ox, oy = X, Y
			//	}
			//	switch ev.Buttons() {
			//	case tcell.ButtonNone:
			//		if ox >= 0 {
			//			label := fmt.Sprintf("%d,%d to %d,%d", ox, oy, X, Y)
			//			drawBox(s, ox, oy, X, Y, boxStyle, label)
			//			ox, oy = -1, -1
			//		}
			//	}
		}

		drawInitialBox(s, boundary, boxStyle)
		for _, snake := range snakes {
			err := drawSnake(s, snake, boundary)
			if err != nil {
				log.Fatalf("%+v", err)
			}
		}
		for _, f := range food {
			//fmt.Println(f)
			err := drawFood(s, f, foodStyle, boundary)
			if err != nil {
				println(err)
				log.Fatalf("%+v", err)
			}
		}
		s.Show()
	}
}

func main() {
	rand.Seed(time.Now().UnixNano())
	ch := make(chan interface{}, 100)

	startPlayerPos := make(map[int]core.Coord)
	startPlayerPos[0] = core.Coord{5, 2}
	startPlayerPos[1] = core.Coord{15, 3}
	startPlayerPos[2] = core.Coord{25, 14}
	startPlayerPos[3] = core.Coord{32, 28}
	ch <- core.PlayerStarts{Players: startPlayerPos}
	ch <- core.Tick{}
	//time.Sleep(5 * time.Second)
	ch <- core.NewFood{Pos: core.Coord{X: 5, Y: 18}}
	ch <- core.Tick{}
	//time.Sleep(2 * time.Second)
	ch <- core.NewFood{Pos: core.Coord{X: 55, Y: 12}}
	ch <- core.Tick{}
	//time.Sleep(2 * time.Second)
	ch <- core.NewFood{Pos: core.Coord{X: 15, Y: 10}}
	ch <- core.Tick{}

	Run(ch)
}