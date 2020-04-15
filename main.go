package main

import (
	"bufio"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/AlecAivazis/survey/v2"
)

func clearScreen() {
	if runtime.GOOS == "windows" {
		cmd := exec.Command("cmd", "/c", "cls")
		cmd.Stdout = os.Stdout
		cmd.Run()
		return
	}

	fmt.Print("\033[H\033[2J")
}

type gameBoard struct {
	cells         []bool
	generation    int
	width, height int
}

func (board *gameBoard) inBounds(x int, y int) bool {
	return (x >= 0 &&
		x < board.width &&
		y >= 0 &&
		y < board.height)
}

var errOutOfBounds = fmt.Errorf("out of bounds")

func (board *gameBoard) get(x, y int) (bool, error) {
	if !board.inBounds(x, y) {
		return false, errOutOfBounds
	}

	return board.cells[y*board.width+x], nil
}

func (board *gameBoard) set(x, y int, value bool) error {
	if !board.inBounds(x, y) {
		return errOutOfBounds
	}

	board.cells[y*board.width+x] = value
	return nil
}

func (board *gameBoard) equals(other *gameBoard) bool {
	if board.width != other.width || board.height != other.height {
		return false
	}

	if len(board.cells) != len(other.cells) {
		return false
	}

	total := board.width * board.height
	for i := 0; i < total; i++ {
		if board.cells[i] != other.cells[i] {
			return false
		}
	}

	return true
}

func (board *gameBoard) randomize(percent int) {
	total := board.width * board.height
	numAlive := percent * total / 100

	for i := 0; i < total; i++ {
		board.cells[i] = i < numAlive
	}

	r := rand.New(rand.NewSource(time.Now().Unix()))
	shuffled := board.cells
	for i := total; i > 0; i-- {
		index := i - 1
		randIndex := r.Intn(i)
		shuffled[index], shuffled[randIndex] = shuffled[randIndex], shuffled[index]
	}
	board.cells = shuffled
}

var neighborsArr = []int{-1, 0, 1}

func (board *gameBoard) neighbors(x, y int) int {
	count := 0

	for _, i := range neighborsArr {
		for _, j := range neighborsArr {
			ix := i + x
			iy := j + y

			if alive, err := board.get(ix, iy); err == nil && alive && !(i == 0 && j == 0) {
				count++
			}
		}
	}

	return count
}

func (board *gameBoard) step() bool {
	oldBoard := newGameBoard(board.width, board.height)
	copy(oldBoard.cells, board.cells)

	for y := 0; y < board.height; y++ {
		for x := 0; x < board.width; x++ {
			alive, err := oldBoard.get(x, y)
			if err != nil {
				continue
			}

			neighbors := oldBoard.neighbors(x, y)

			if !alive {
				if neighbors == 3 {
					board.set(x, y, true)
				}

				continue
			}

			if neighbors < 2 {
				board.set(x, y, false)
				continue
			}

			if neighbors > 3 {
				board.set(x, y, false)
				continue
			}
		}
	}

	board.generation++

	return !board.equals(oldBoard)
}

func (board *gameBoard) print() {
	fmt.Print("╔")
	for x := 1; x <= board.width; x++ {
		fmt.Print("══")
	}
	fmt.Println("╗")

	for y := 0; y < board.height; y++ {
		fmt.Print("║")
		for x := 0; x < board.width; x++ {
			if alive, err := board.get(x, y); err == nil && alive {
				fmt.Print("██")
			} else {
				fmt.Print("  ")
			}
		}
		fmt.Println("║")
	}

	fmt.Print("╚")
	for x := 1; x <= board.width; x++ {
		fmt.Print("══")
	}
	fmt.Println("╝")
}

func newGameBoard(w, h int) *gameBoard {
	cells := make([]bool, w*h)

	return &gameBoard{
		cells:      cells,
		generation: 0,
		width:      w,
		height:     h,
	}
}

func setupCloseHandler() {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("\r- Ctrl+C pressed in Terminal")
		os.Exit(0)
	}()
}

type exampleConfig struct {
	path  string
	label string
}

func main() {
	clearScreen()
	fmt.Println("Welcome to the Game of Life!")
	fmt.Println()

	var exampleConfigs []exampleConfig
	filepath.Walk("./configs", func(path string, info os.FileInfo, err error) error {
		name := filepath.Base(path)
		ext := filepath.Ext(name)

		if ext == ".cells" {
			exampleConfigs = append(exampleConfigs, exampleConfig{
				path:  path,
				label: strings.TrimSuffix(name, ext),
			})
		}
		return nil
	})

	startingGridOptions := []string{"random"}
	for _, exampleConfig := range exampleConfigs {
		startingGridOptions = append(startingGridOptions, exampleConfig.label)
	}

	startingGrid := ""
	startingGridPrompt := &survey.Select{
		Message: "Choose a starting grid:",
		Options: startingGridOptions,
	}
	survey.AskOne(startingGridPrompt, &startingGrid)

	setupCloseHandler()

	board := newGameBoard(60, 40)

	switch startingGrid {
	case "random":
		board.randomize(10)
		break
	default:
		var selectedOption exampleConfig
		for _, exampleConfig := range exampleConfigs {
			if exampleConfig.label == startingGrid {
				selectedOption = exampleConfig
				break
			}
		}

		exampleFile, err := os.Open(selectedOption.path)
		if err != nil {
			log.Fatal("could not open the selected configuration")
		}
		reader := bufio.NewReader(exampleFile)

		y := 0
		for line, _, err := reader.ReadLine(); err == nil; line, _, err = reader.ReadLine() {
			if len(line) == 0 || line[0] == '!' {
				continue
			}

			for x, c := range line {
				board.set(x, y, c == 'O')
			}
			y++
		}

		exampleFile.Close()
		break
	}

	draw := func() {
		clearScreen()
		board.print()
		fmt.Println("Generation", board.generation)
		fmt.Println("Press Ctrl+C to exit")
		time.Sleep(time.Second / 20)
	}

	draw()
	for board.step() {
		draw()
	}
}
