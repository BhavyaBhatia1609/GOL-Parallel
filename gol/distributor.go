package gol

import (
	"fmt"
	"uk.ac.bris.cs/gameoflife/util"
)

type distributorChannels struct {
	events     chan<- Event
	ioCommand  chan<- ioCommand
	ioIdle     <-chan bool
	ioFilename chan<- string
	ioOutput   chan<- uint8
	ioInput    <-chan uint8
}

func readWorld(p Params, c distributorChannels) [][]byte {
	c.ioCommand <- ioInput
	c.ioFilename <- fmt.Sprintf("%dx%d", p.ImageWidth, p.ImageHeight)

	World := make([][]byte, p.ImageHeight)
	for i := range World {
		World[i] = make([]byte, p.ImageWidth)
	}
	//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
	for j := 0; j < p.ImageHeight; j++ {
		for i := 0; i < p.ImageWidth; i++ {
			if <-c.ioInput == 255 {
				World[j][i] = 255
				c.events <- CellFlipped{
					CompletedTurns: 0,
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

func worker() {

}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels) {

	// TODO: Create a 2D slice to store the world.

	World := readWorld(p, c)

	// TODO: Execute all turns of the Game of Life.
	go calculateNextState(p, World)
	turnNum := 0
	for turn := 0; turn < p.Turns; turn++ {
		World = calculateNextState(p, World)
		turnNum++
	}

	// TODO: Report the final state using FinalTurnCompleteEvent.
	c.events <- FinalTurnComplete{CompletedTurns: p.Turns,
		Alive: calculateAliveCells(p, World)}

	writeWorld(p, c, World, turnNum)
	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{p.Turns, Quitting}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}

func calculateNextState(p Params, world [][]byte) [][]byte {
	newWorld := make([][]byte, p.ImageHeight)
	for i := range newWorld {
		newWorld[i] = make([]byte, p.ImageWidth)
	}

	for y := 0; y < p.ImageHeight; y++ {
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

			if world[y][x] == 255 && count < 2 {
				newWorld[y][x] = 0
			} else if world[y][x] == 255 && count > 3 {
				newWorld[y][x] = 0
			} else if world[y][x] == 0 && count == 3 {
				newWorld[y][x] = 255
			} else {
				newWorld[y][x] = world[y][x]
			}
		}
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
