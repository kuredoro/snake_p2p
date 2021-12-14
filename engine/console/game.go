package console

import (
	"github.com/gdamore/tcell/v2"
	"github.com/kuredoro/snake_p2p/core"
)

type Snake struct {
	id    int         // snakep2p id
	body  []core.Coord     // coordinates of snakep2p's body
	head  core.Coord       // coordinates of snakep2p's head
	style tcell.Style // snakep2p's style
}

type Game struct {
	snakes []Snake
	food []core.Coord
}
