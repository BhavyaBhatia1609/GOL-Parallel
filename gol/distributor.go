package gol

import (
	"fmt"
	"os"
	"time"
	"uk.ac.bris.cs/gameoflife/util"
)

type distributorChannels struct {
	events     chan<- Event
	ioCommand  chan<- ioCommand
	ioIdle     <-chan bool
	ioFilename chan<- string
	ioOutput   chan<- uint8
	ioInput    <-chan uint8
	ioKeyPress <-chan rune
}

func readWorld(p Params, c distributorChannels) [][]byte {
	c.ioCommand <- ioInput
	c.ioFilename <- fmt.Sprintf("%dx%d", p.ImageWidth, p.ImageHeight)

	World := make([][]byte, p.ImageHeight)
	for i := range World {
		World[i] = make([]byte, p.ImageWidth)
	}

	for j := 0; j < p.ImageHeight; j++ {
		for i := 0; i < p.ImageWidth; i++ {
			if <-c.ioInput == 255 {
				World[j][i] = 255
				c.events <- CellFlipped{
					CompletedTurns: p.Turns,
					Cell:           util.Cell{X: i, Y: j},
				}
			}
		}
	}
	return World
}

func writeWorld(p Params, c distributorChannels, world [][]byte, turnNum int) {
	c.ioCommand <- ioOutput
	c.ioFilename <- fmt.Sprintf("%dx%dx%d", p.ImageWidth, p.ImageHeight, turnNum)

	for j := 0; j < p.ImageHeight; j++ {
		for i := 0; i < p.ImageWidth; i++ {
			c.ioOutput <- world[j][i]
		}
	}
}

func worker(p Params, world [][]byte, c distributorChannels, turn int, startY int, endY int, out chan<- [][]byte) {
	world1 := calculateNextState(p, world, c, turn, startY, endY)
	out <- world1
}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels) {
	World := readWorld(p, c) //Reads the world and puts it in a 2D slice

	ticker := time.NewTicker(2 * time.Second)

	for turn := 0; turn < p.Turns; {
		select {
		case <-ticker.C:
			c.events <- AliveCellsCount{CompletedTurns: turn,
				CellsCount: counterCells(p, World)}
		case key := <-c.ioKeyPress:
			switch key {
			case 'p':
				c.events <- StateChange{turn, Paused}
				fmt.Println("Current turn:", turn)
				for {
					if <-c.ioKeyPress == 'p' {
						c.events <- StateChange{turn, Executing}
						fmt.Println("Continuing")
						break
					}
				}
			case 'q':
				writeWorld(p, c, World, turn) //writes the world in the current turn that the user quit on
				c.events <- StateChange{turn, Quitting}
				os.Exit(0)
			case 's':
				writeWorld(p, c, World, turn) //writes the world in the current turn that the user is on
			}
		default:
			WorkerOut := make([]chan [][]byte, p.Threads) // A 2D matrix of channels to put in the slices of the world
			for i := range WorkerOut {
				WorkerOut[i] = make(chan [][]byte)
			}

			sliceHeight := p.ImageHeight / p.Threads
			remaining := p.ImageHeight % p.Threads

			if p.Threads > 1 {
				for thread := 0; thread < p.Threads; thread++ {
					if (remaining > 0) && ((thread + 1) == p.Threads) {
						go worker(p, World, c, turn, thread*sliceHeight, ((thread+1)*sliceHeight)+remaining, WorkerOut[thread])
					} else {
						go worker(p, World, c, turn, thread*sliceHeight, (thread+1)*sliceHeight, WorkerOut[thread])
					}
				}

				newWorld := make([][]byte, 0) // A new world slice to append what was taken from the worker out channel
				for i := 0; i < p.Threads; i++ {
					part := <-WorkerOut[i]
					newWorld = append(newWorld, part...)
				}

				World = newWorld
				c.events <- TurnComplete{turn}
				turn++
			} else {
				World = calculateNextState(p, World, c, turn, 0, p.ImageHeight)
				c.events <- TurnComplete{turn}
				turn++
			}
		}
	}

	c.events <- FinalTurnComplete{CompletedTurns: p.Turns,
		Alive: calculateAliveCells(p, World)}

	writeWorld(p, c, World, p.Turns)
	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{p.Turns, Quitting}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}

func calculateNextState(p Params, world [][]byte, c distributorChannels, turn int, start int, end int) [][]byte {
	newWorld := make([][]byte, end-start)
	for i := range newWorld {
		newWorld[i] = make([]byte, p.ImageWidth)
	}
	k := 0 // The position where the y would be in a particular slice from the worker since we slice them into start and end
	for y := start; y < end; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			count := 0
			for j := y - 1; j <= y+1; j++ {
				for i := x - 1; i <= x+1; i++ {
					if j == y && i == x {
						continue
					}
					w, z := i, j

					if z >= p.ImageHeight {
						z = 0
					}
					if w >= p.ImageWidth {
						w = 0
					}
					if z < 0 {
						z = p.ImageHeight - 1
					}
					if w < 0 {
						w = p.ImageWidth - 1
					}
					if world[z][w] == 255 {
						count++
					}
				}
			}

			if world[y][x] == 255 {
				if count < 2 {
					newWorld[k][x] = 0
					c.events <- CellFlipped{turn, util.Cell{X: x, Y: y}}
				} else if count == 2 || count == 3 {
					newWorld[k][x] = 255
				} else {
					newWorld[k][x] = 0
					c.events <- CellFlipped{turn, util.Cell{X: x, Y: y}}
				}
			} else {
				if count == 3 {
					newWorld[k][x] = 255
					c.events <- CellFlipped{turn, util.Cell{X: x, Y: y}}
				}
			}
		}
		k++
	}
	return newWorld
}

func calculateAliveCells(p Params, world [][]byte) []util.Cell {
	slice := []util.Cell{}
	for j := 0; j < p.ImageHeight; j++ {
		for i := 0; i < p.ImageWidth; i++ {
			if world[j][i] == 255 {
				slice = append(slice, util.Cell{X: i, Y: j})
			}
		}
	}
	return slice
}

func counterCells(p Params, world [][]byte) int {
	count := 0
	for j := 0; j < p.ImageHeight; j++ {
		for i := 0; i < p.ImageWidth; i++ {
			if world[j][i] == 255 {
				count++
			}
		}
	}
	return count
}
