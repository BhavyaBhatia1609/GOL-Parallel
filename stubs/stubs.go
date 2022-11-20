package stubs

var ProcessGameOfLife = "Game.ProcessGameOfLife"

type Response struct {
	World [][]byte
}

type Request struct {
	World  [][]byte
	Thread int
}
