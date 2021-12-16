package core

type Coord struct {
	X, Y int
}

type PlayerStarts struct {
	Players map[int]Coord  // map from player's SnakeID to its start coordinates
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
type PlayerMove struct {
	Moves map[int] Direction   // map from player's SnakeID to direction of it's move
}

type PlayerDied struct {
	SnakeID int // SnakeID of player who died
}

type FoodEaten struct {
	FoodID int
}

type PushSegment struct {
	SnakeID int
	Pos     Coord // coordinates of a cell to add to a snake
}

type Tick struct{}

type GameOver struct {
	Successful bool    // did game finish without errors or not
	Winner int      // SnakeID of winner player
}
