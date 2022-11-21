package gol

import (
	"flag"
	"fmt"
	"net/rpc"
	"os"
	"time"
	"uk.ac.bris.cs/gameoflife/stubs"
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

var server *string

func init() {
	//serverIP := "localhost:8030"
	serverIP := "44.212.21.187:8030"
	server = flag.String("server", serverIP, "IP:port string to connect to as server")
}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels) {
	World := readWorld(p, c) //Reads the world and puts it in a 2D slice

	ticker := time.NewTicker(2 * time.Second)

	flag.Parse()
	client, _ := rpc.Dial("tcp", *server)
	defer client.Close()

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
			newWorld := makeCall(client, p, World)

			for j := 0; j < p.ImageHeight; j++ {
				for i := 0; i < p.ImageWidth; i++ {
					if newWorld[j][i] != World[j][i] {
						c.events <- CellFlipped{
							CompletedTurns: turn,
							Cell:           util.Cell{X: i, Y: j},
						}
					}
				}
			}

			World = newWorld
			c.events <- TurnComplete{turn}
			turn++
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

func makeCall(client *rpc.Client, p Params, world [][]byte) [][]byte {
	request := stubs.Request{World: world, Thread: p.Threads}
	response := new(stubs.Response)
	client.Call(stubs.ProcessGameOfLife, request, response)
	return response.World
}
