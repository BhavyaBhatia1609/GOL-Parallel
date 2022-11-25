package stubs

var ProcessGameOfLife = "Game.ProcessGameOfLife"
var Counter = "Game.counter"

type Response struct {
	World     [][]byte
	TurnsDone int
	Count     int
}

type Request struct {
	World  [][]byte
	Thread int
	Turns  int
	StartY int
	EndY   int
}
