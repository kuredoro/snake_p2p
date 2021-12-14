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
	startPlayerPos[0] = core.Coord{X: 5, Y: 2}
	startPlayerPos[1] = core.Coord{X: 15, Y: 3}
	startPlayerPos[2] = core.Coord{X: 25, Y: 14}
	startPlayerPos[3] = core.Coord{X: 32, Y: 28}
	game.Ch <- core.PlayerStarts{Players: startPlayerPos}
	game.Ch <- core.Tick{}
	game.Ch <- core.NewFood{Pos: core.Coord{X: 5, Y: 18}}
	game.Ch <- core.Tick{}
	game.Ch <- core.NewFood{Pos: core.Coord{X: 55, Y: 12}}
	game.Ch <- core.Tick{}
	game.Ch <- core.NewFood{Pos: core.Coord{X: 15, Y: 10}}
	game.Ch <- core.Tick{}
	game.RunGame()
}