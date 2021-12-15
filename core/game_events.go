package core

type Coord struct {
	X, Y int
}

type PlayerStarts struct {
	Players map[int]Coord  // map from player's ID to its start coordinates
}

type NewFood struct {
	Pos Coord    // coordinates of new food
}

type Direction int

const (
	Up Direction = iota
	Right
	Down
	Left
)
type PlayerMove struct {
	Moves map[int] Direction   // map from player's ID to direction of it's move
}

type PlayerDied struct {
	ID int     // ID of player who died
}

type FoodEaten struct {
	Pos Coord  // coordinates of eaten food
}

type PushSegment struct {
	ID int		// ID of snake
	Pos Coord	// coordinates of a cell to add to a snake
}

type Tick struct{}

type GameOver struct {
	Successful bool    // did game finish without errors or not
	Winner int      // ID of winner player
}
