package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"
)

type macroMode struct {
	name        string
	printConfig func()
	editConfig  func(*bufio.Reader)
	start       func()
	stop        func()
	toggle      func()
}

var goliathMode = &macroMode{
	name:        "Goliath",
	printConfig: printGoliathConfig,
	editConfig:  editGoliathConfig,
	start:       startGoliath,
	stop:        stopGoliath,
	toggle:      toggleGoliath,
}

var crushMode = &macroMode{
	name:        "Crush Combo",
	printConfig: printCrushConfig,
	editConfig:  editCrushConfig,
	start:       startCrush,
	stop:        stopCrush,
	toggle:      toggleCrush,
}

var currentMode *macroMode

func selectMode(reader *bufio.Reader) *macroMode {
	for {
		fmt.Println("Select macro:")
		fmt.Println("  1 - Goliath")
		fmt.Println("  2 - Crush Combo")
		fmt.Print("> ")

		line, err := reader.ReadString('\n')
		if err != nil {
			return nil
		}
		switch strings.TrimSpace(line) {
		case "1":
			return goliathMode
		case "2":
			return crushMode
		case "q", "quit", "exit":
			return nil
		default:
			fmt.Println("Enter 1 or 2.")
		}
	}
}

func main() {
	rand.Seed(time.Now().UnixNano())
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("==== FH6 Macros (terminal) ====")
	fmt.Println()

	currentMode = selectMode(reader)
	if currentMode == nil {
		return
	}

	fmt.Println()
	fmt.Printf("Mode: %s\n", currentMode.name)
	fmt.Println("Hotkey: Ctrl + Up - start/stop")
	fmt.Println("Commands: start | stop | edit | show | mode | quit")
	fmt.Println()

	currentMode.printConfig()
	fmt.Println()

	if askYesNo(reader, "Edit timings before start?", false) {
		currentMode.editConfig(reader)
	}

	go hotkeyListener(func() { currentMode.toggle() })

	for {
		fmt.Print("> ")
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		switch strings.TrimSpace(strings.ToLower(line)) {
		case "start", "s":
			currentMode.start()
		case "stop", "x":
			currentMode.stop()
		case "edit", "e":
			if running {
				fmt.Println("Stop the macro first")
				continue
			}
			currentMode.editConfig(reader)
		case "show":
			currentMode.printConfig()
		case "mode", "m":
			if running {
				fmt.Println("Stop the macro first")
				continue
			}
			newMode := selectMode(reader)
			if newMode != nil {
				currentMode = newMode
				fmt.Println()
				fmt.Printf("Mode: %s\n", currentMode.name)
				currentMode.printConfig()
				fmt.Println()
			}
		case "quit", "q", "exit":
			currentMode.stop()
			fmt.Println("Exiting")
			return
		case "":
		default:
			fmt.Println("Commands: start | stop | edit | show | mode | quit")
		}
	}
}
