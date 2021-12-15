package main

import (
	"github.com/kuredoro/snake_p2p/core"
	"github.com/kuredoro/snake_p2p/engine/console"
	"math/rand"
	"time"
)

func main() {
	rand.Seed(time.Now().UnixNano())
	// Create Game
	game := new(console.Game)
	game.Ch = make(chan interface{}, 100)
	// Add events to channel
	startPlayerPos := make(map[int]core.Coord)
	startPlayerPos[0] = core.Coord{X: 4, Y: 2}
	startPlayerPos[1] = core.Coord{X: 7, Y: 5}
	startPlayerPos[2] = core.Coord{X: 10, Y: 2}
	startPlayerPos[3] = core.Coord{X: 10, Y: 6}
	game.Ch <- core.PlayerStarts{Players: startPlayerPos}
	game.Ch <- core.Tick{}
	game.Ch <- core.NewFood{Pos: core.Coord{X: 5, Y: 2}}
	game.Ch <- core.Tick{}
	game.Ch <- core.FoodEaten{Pos: core.Coord{X: 5, Y: 2}}
	game.Ch <- core.PlayerMove{Moves: map[int]core.Direction{0: core.Right,
															 1: core.Left,
															 2: core.Down,
															 3: core.Up}}
	game.Ch <- core.PushSegment{ID: 0, Pos: core.Coord{X: 4, Y: 2}}
	game.Ch <- core.NewFood{Pos: core.Coord{X: 5, Y: 3}}
	game.Ch <- core.Tick{}
	game.Ch <- core.NewFood{Pos: core.Coord{X: 10, Y: 4}}
	game.Ch <- core.Tick{}
	game.Ch <- core.PlayerMove{Moves: map[int]core.Direction{0: core.Down,
															 1: core.Down,
															 2: core.Down,
															 3: core.Right}}
	game.Ch <- core.FoodEaten{Pos: core.Coord{X: 5, Y: 3}}
	game.Ch <- core.FoodEaten{Pos: core.Coord{X: 10, Y: 4}}
	game.Ch <- core.PushSegment{ID: 0, Pos: core.Coord{X: 4, Y: 2}}
	game.Ch <- core.PushSegment{ID: 2, Pos: core.Coord{X: 10, Y: 3}}
	game.Ch <- core.PlayerMove{Moves: map[int]core.Direction{0: core.Down,
															 1: core.Right,
															 2: core.Down,
															 3: core.Left}}
	game.Ch <- core.PlayerDied{ID: 2}
	game.Ch <- core.PlayerDied{ID: 3}
	game.Ch <- core.GameOver{Successful: true, Winner: 0}
	game.RunGame()
}