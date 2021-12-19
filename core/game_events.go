package core

import "github.com/libp2p/go-libp2p-core/peer"

type Coord struct {
	X, Y int
}

func EqualCoord(a, b Coord) bool {
	return a.X == b.X && a.Y == b.Y
}

type PlayerStarts struct {
	Players map[peer.ID]Coord // map from player's SnakeID to its start coordinates
}

type NewFood struct {
	FoodID int
	Pos    Coord
}

type Direction int

const (
	Up Direction = iota
	Right
	Down
	Left
)

type PlayerMoves struct {
	Moves map[peer.ID]Direction // map from player's SnakeID to direction of it's move
}

type PlayerDied struct {
	SnakeID peer.ID // SnakeID of player who died
}

type FoodEaten struct {
	FoodID int
}

type PushSegment struct {
	SnakeID peer.ID
	Pos     Coord // coordinates of a cell to add to a snake
}

type Tick struct{}

type GameOver struct {
	Successful bool // did game finish without errors or not
	Winner     int  // SnakeID of winner player
}
