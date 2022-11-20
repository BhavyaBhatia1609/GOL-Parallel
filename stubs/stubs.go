package stubs

type Response struct {
	World [][]byte
}

type Request struct {
	World       [][]byte
	Turn        int
	Thread      int
	imageHeight int
	imageWidth  int
	Start       int
	End         int
}
