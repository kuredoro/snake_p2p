package console

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/kuredoro/snake_p2p/protocol/game"
	"github.com/libp2p/go-libp2p-core/peer"
	//"strconv"

	//"log"
	"github.com/gdamore/tcell/v2"
	"github.com/kuredoro/snake_p2p/core"
	"github.com/rs/zerolog/log"
	//"github.com/sanity-io/litter"
)

type Snake struct {
	Alive bool
	Body  []core.Coord
	Head  core.Coord
	Style tcell.Style
}

type GameUI struct {
	gi          *game.GameInstance
	Snakes      map[peer.ID]*Snake
	Food        map[int]core.Coord
	bound       Boundary
	moveNum     int
	AliveSnakes int
	foodLastID  int
	Over        bool
	Successful  bool
	WinnerID    peer.ID
	r           *rand.Rand
}

// add food every N moves
const N = 5

func NewGame(gi *game.GameInstance) *GameUI {
	return &GameUI{
		gi:         gi,
		Food:       make(map[int]core.Coord),
		Snakes:     make(map[peer.ID]*Snake),
		moveNum:    0,
		foodLastID: 0,
		bound:      Boundary{core.Coord{X: 1, Y: 1}, core.Coord{X: 81, Y: 41}},
		Over:       false,
		Successful: false,
	}
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

func drawSnake(s tcell.Screen, ID peer.ID, snake *Snake, boundary Boundary) error {
	if boundary.Contains(snake.Head) {
		return fmt.Errorf("snakep2p's head coordinates (%d, %d) are out of boundary", snake.Head.X, snake.Head.Y)
	}
	// id := []rune(strconv.Itoa(ID))
	s.SetContent(snake.Head.X, snake.Head.Y, tcell.RuneDiamond, nil, snake.Style)
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

var defColors = map[tcell.Color]struct{}{
	tcell.ColorReset:  {},
	tcell.ColorWhite:  {},
	tcell.ColorPurple: {},
	tcell.ColorRed:    {},
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

func (g *GameUI) markDead(newHeadCoord map[peer.ID]core.Coord) {
	// Check snakes for death
	for id1, coord1 := range newHeadCoord {
		// head into head
		for id2, coord2 := range newHeadCoord {
			if id1.Pretty() == id2.Pretty() {
				continue
			}
			if core.EqualCoord(coord1, coord2) {
				g.Snakes[id1].Alive = false
				g.Snakes[id2].Alive = false
				g.AliveSnakes -= 2
			}
		}
		// head into body
		for _, snake := range g.Snakes {
			if core.EqualCoord(snake.Head, coord1) {
				g.Snakes[id1].Alive = false
				g.AliveSnakes--
				break
			}
			for _, b := range snake.Body {
				if core.EqualCoord(b, coord1) {
					g.Snakes[id1].Alive = false
					g.AliveSnakes--
					break
				}
			}
		}
	}
}

func (g *GameUI) goodCoord(coord core.Coord) core.Coord {
	if coord.X == g.bound.TopLeft.X {
		coord = core.Coord{X: g.bound.BottomRight.X - 1, Y: coord.Y}
	}
	if coord.X == g.bound.BottomRight.X {
		coord = core.Coord{X: g.bound.TopLeft.X + 1, Y: coord.Y}
	}
	if coord.Y == g.bound.TopLeft.Y {
		coord = core.Coord{X: coord.X, Y: g.bound.BottomRight.Y - 1}
	}
	if coord.Y == g.bound.BottomRight.Y {
		coord = core.Coord{X: coord.X, Y: g.bound.TopLeft.Y + 1}
	}
	return coord
}

func (g *GameUI) handleOutOfBoundary(newHeadCoord *map[peer.ID]core.Coord) {
	for id, coord := range *newHeadCoord {
		(*newHeadCoord)[id] = g.goodCoord(coord)
	}
}

func (g *GameUI) eatFood(newHeadCoord map[peer.ID]core.Coord) {
	for id, coord := range newHeadCoord {
		for foodID, foodCoord := range g.Food {
			if !core.EqualCoord(coord, foodCoord) {
				continue
			}
			// add segment to snake
			g.Snakes[id].Body = append(g.Snakes[id].Body, core.Coord{})
			// remove food from field
			delete(g.Food, foodID)
			log.Info().Msgf("Food on (%#d, %#d) eaten by %#s", coord.X, coord.Y, id.Pretty())
		}
	}
}

func (g *GameUI) moveSnakes(newHeadCoord map[peer.ID]core.Coord) {
	for id, coord := range newHeadCoord {
		// move head
		prevHead := g.Snakes[id].Head
		g.Snakes[id].Head = coord
		log.Info().Msgf("New coordinates (%#d, %#d) for snake %#s", g.Snakes[id].Head.X, g.Snakes[id].Head.Y, id.Pretty())
		// move snakes body
		if len(g.Snakes[id].Body) == 0 {
			continue
		}
		for i := len(g.Snakes[id].Body) - 1; i > 0; i-- {
			g.Snakes[id].Body[i] = g.Snakes[id].Body[i-1]
		}
		g.Snakes[id].Body[0] = prevHead
	}
}

func (g *GameUI) newFood() {
	if g.moveNum%N != 0 {
		return
	}

	x1, y1 := g.bound.TopLeft.X, g.bound.TopLeft.Y
	x2, y2 := g.bound.BottomRight.X, g.bound.BottomRight.Y
	c := len(g.Snakes) + (g.moveNum / N) + 2
	var cell int
	if y2-y1-c <= 0 || x2-x1-c <= 0 {
		cell = g.r.Intn(10)
	} else {
		cell = g.r.Intn((x2 - x1 - c) * (y2 - y1 - c))
	}
	for row := x1 + 1; row < x2; row++ {
		for col := y1 + 1; col < y2; col++ {
			flag := true
			for _, snake := range g.Snakes {
				if snake.Head.X == row && snake.Head.Y == col {
					flag = false
					break
				}
				for _, b := range snake.Body {
					if b.X == row && b.Y == col {
						flag = false
						break
					}
				}
			}
			for _, f := range g.Food {
				if f.X == row && f.Y == col {
					flag = false
					break
				}
			}
			if flag {
				cell--
			}
			if cell == 0 {
				g.Food[g.foodLastID] = core.Coord{X: row, Y: col}
				g.foodLastID++
				log.Info().Msgf("New food should be created on (%#d, %#d)", row, col)
				return
			}
		}
	}
}

func (g *GameUI) isOver() bool {
	if g.AliveSnakes == 1 {
		g.Over = true
		g.Successful = true
		for id, snake := range g.Snakes {
			if snake.Alive {
				g.WinnerID = id
				break
			}
		}
	} else if g.AliveSnakes < 1 {
		g.Over = true
		g.Successful = false
	}
	return g.Over
}

func (g *GameUI) handleMoves(moves core.PlayerMoves) {
	newHeadCoord := make(map[peer.ID]core.Coord)
	for id, dir := range moves.Moves {
		switch dir {
		case core.Up:
			newHeadCoord[id] = core.Coord{X: g.Snakes[id].Head.X, Y: g.Snakes[id].Head.Y - 1}
		case core.Left:
			newHeadCoord[id] = core.Coord{X: g.Snakes[id].Head.X - 1, Y: g.Snakes[id].Head.Y}
		case core.Right:
			newHeadCoord[id] = core.Coord{X: g.Snakes[id].Head.X + 1, Y: g.Snakes[id].Head.Y}
		case core.Down:
			newHeadCoord[id] = core.Coord{X: g.Snakes[id].Head.X, Y: g.Snakes[id].Head.Y + 1}
		default:
			panic("the value of direction is unknown")
		}
	}
	g.handleOutOfBoundary(&newHeadCoord)
	g.markDead(newHeadCoord)
	if g.isOver() {
		return
	}
	g.eatFood(newHeadCoord)
	g.moveSnakes(newHeadCoord)
	g.newFood()
	g.moveNum++
	log.Info().Msgf("Next move %#d", g.moveNum)
}

func (g *GameUI) checkMove(move core.Coord) bool {
	id := g.gi.SelfID()
	if len(g.Snakes[id].Body) > 0 {
		if g.Snakes[id].Body[0] == move {
			return false
		}
	}
	return true
}

func (g *GameUI) handleKeyEvent(ev *tcell.EventKey) {
	id := g.gi.SelfID()
	if ev.Key() == tcell.KeyUp {
		snake := g.Snakes[id]
		move := g.goodCoord(core.Coord{X: snake.Head.X, Y: snake.Head.Y - 1})
		if !g.checkMove(move) {
			return
		}
		err := g.gi.SendMove(core.Up)
		if err != nil {
			log.Err(err).Int("move", int(core.Up)).Msg("Key pressed")
		}

		log.Info().Int("move", int(core.Up)).Msg("Key pressed")

	}
	if ev.Key() == tcell.KeyDown {
		move := g.goodCoord(core.Coord{X: g.Snakes[id].Head.X, Y: g.Snakes[id].Head.Y + 1})
		if !g.checkMove(move) {
			return
		}
		err := g.gi.SendMove(core.Down)
		if err != nil {
			log.Err(err).Int("move", int(core.Down)).Msg("Key pressed")
		}

		log.Info().Int("move", int(core.Down)).Msg("Key pressed")

	}
	if ev.Key() == tcell.KeyRight {
		move := g.goodCoord(core.Coord{X: g.Snakes[id].Head.X + 1, Y: g.Snakes[id].Head.Y})
		if !g.checkMove(move) {
			return
		}
		err := g.gi.SendMove(core.Right)
		if err != nil {
			log.Err(err).Int("move", int(core.Right)).Msg("Key pressed")
		}
		log.Info().Int("move", int(core.Right)).Msg("Key pressed")

	}
	if ev.Key() == tcell.KeyLeft {
		move := g.goodCoord(core.Coord{X: g.Snakes[id].Head.X - 1, Y: g.Snakes[id].Head.Y})
		if !g.checkMove(move) {
			return
		}
		err := g.gi.SendMove(core.Left)
		if err != nil {
			log.Err(err).Int("move", int(core.Left)).Msg("Key pressed")
		}
		log.Info().Int("move", int(core.Left)).Msg("Key pressed")

	}
}

func (g *GameUI) RunGame(seed int64) {
	rand.Seed(seed)
	g.r = rand.New(rand.NewSource(seed))
	// Generate snakes
	for _, id := range g.gi.PlayersIDs() {
		var start core.Coord
		for {
			start = core.Coord{
				X: g.r.Intn(g.bound.BottomRight.X-g.bound.TopLeft.X-1) + 1,
				Y: g.r.Intn(g.bound.BottomRight.Y-g.bound.TopLeft.Y-1) + 1,
			}
			flag := true
			for _, snake := range g.Snakes {
				if core.EqualCoord(start, snake.Head) {
					flag = false
				}
			}
			if flag {
				break
			}
		}
		// log.Debug().Msg("\nSnakeIDs: " + id.Pretty())
		g.Snakes[id] = &Snake{Alive: true, Head: start, Style: genSnakeStyle(defColors)}
	}
	g.AliveSnakes = len(g.Snakes)
	// Define GameUI styles
	defStyle := tcell.StyleDefault.Background(tcell.ColorReset).Foreground(tcell.ColorReset)
	boxStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorPurple)
	blackBoxStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack)
	foodStyle := tcell.StyleDefault.Foreground(tcell.ColorRed).Background(tcell.ColorPurple)

	// Initialize GameUI Screen
	s, err := tcell.NewScreen()
	if err != nil {
		log.Err(err)
	}
	if err := s.Init(); err != nil {
		log.Err(err)
	}
	s.DisableMouse()
	s.EnablePaste()
	s.Clear()
	s.SetStyle(defStyle)

	// Define function to quit the GameUI
	quit := func() {
		s.Fini()
		os.Exit(0)
	}

	// GameUI loop
	for {
		// Draw GameUI state
		if g.Over {
			drawBox(s, g.bound, boxStyle)
			width, height := 0, 0
			if g.Successful {
				height = 4
				width = len(g.WinnerID.Pretty()) + 10
			} else {
				height = 2
				width = 12
			}
			x1 := (g.bound.BottomRight.X - g.bound.TopLeft.X - width) / 2
			y1 := (g.bound.BottomRight.Y - g.bound.TopLeft.Y - height) / 2
			x2 := (g.bound.BottomRight.X - g.bound.TopLeft.X + width) / 2
			y2 := (g.bound.BottomRight.Y - g.bound.TopLeft.Y + height) / 2
			drawBox(s, Boundary{core.Coord{X: x1, Y: y1}, core.Coord{X: x2, Y: y2}}, blackBoxStyle)
			drawText(s, x1+1, y1+1, x2-1, y2-1, blackBoxStyle, "GameUI Over")
			if g.Successful {
				text := "WinnerID " + g.WinnerID.Pretty()
				drawText(s, x1+1, y1+3, x2-1, y2-1, blackBoxStyle, text)
			}
		} else {
			drawBox(s, g.bound, boxStyle)
			for id, snake := range g.Snakes {
				if !snake.Alive {
					continue
				}
				err := drawSnake(s, id, snake, g.bound)
				if err != nil {
					s.Fini()
					log.Err(err)
					os.Exit(0)
				}
				// log.Info().Msg("Drew snake")
			}
			for _, f := range g.Food {
				err := drawFood(s, f, foodStyle, g.bound)
				if err != nil {
					s.Fini()
					fmt.Println(err)
					log.Err(err)
					os.Exit(0)
				}
				// log.Info().Msg("Drew food")
			}
		}
		s.Show()

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
				if g.Over && ev.Key() == tcell.KeyEnter {
					quit()
				}
				g.handleKeyEvent(ev)

				// TODO: spare me...
			getIncomming:
				for {
					select {
					case moves, ok := <-g.gi.IncommingMoves():
						// Process GameUI event
						if !ok {
							// s.Fini()
							panic("Channel is closed")
						}
						log.Info().Msgf("Incoming message %#v", moves.Moves)
						g.handleMoves(moves)
						break getIncomming
						}
					}
				}
			}
		}
	}
}
