package snake_p2p

type Coord struct {
	X, Y int
}
type PlayerStarts map[int]Coord // map from player's ID to its start coordinates

type NewFood Coord // coordinates of new food

type Direction int

const (
	Up Direction = iota
	Right
	Down
	Left
)

type PlayerMove map[int]Direction // map from player's ID to direction of it's move

type PlayerDied int // ID of player who died

type FoodEaten struct {
	Id    int   // ID of player who ate food
	Point Coord // coordinates of eaten food
}

type Tick struct{}

type GameOver struct {
	successful bool // did game finish without errors or not
	winner     int  // ID of winner player
}
