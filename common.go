package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	running bool
	mu      sync.Mutex
	stopCh  chan struct{}
)

func randomDuration(minMs, maxMs int) time.Duration {
	if maxMs <= minMs {
		return time.Duration(minMs) * time.Millisecond
	}
	return time.Duration(rand.Intn(maxMs-minMs+1)+minMs) * time.Millisecond
}

func randIntInRange(min, max int) int {
	if max <= min {
		return min
	}
	return rand.Intn(max-min+1) + min
}

func sleepOrStop(d time.Duration) bool {
	select {
	case <-time.After(d):
		return false
	case <-stopCh:
		return true
	}
}

func logStatus(s string) {
	fmt.Printf("[%s] %s\n", time.Now().Format("15:04:05"), s)
}

func readInt(reader *bufio.Reader, prompt string, def int) int {
	fmt.Printf("%s [%d]: ", prompt, def)
	line, err := reader.ReadString('\n')
	if err != nil {
		return def
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return def
	}
	v, err := strconv.Atoi(line)
	if err != nil {
		fmt.Println("  -> invalid value, using default")
		return def
	}
	return v
}

func promptRange(reader *bufio.Reader, label string, defMin, defMax int) (int, int) {
	fmt.Printf("  %s:\n", label)
	min := readInt(reader, "    min", defMin)
	max := readInt(reader, "    max", defMax)
	if min > max {
		fmt.Println("    min > max, swapping")
		min, max = max, min
	}
	return min, max
}

func askYesNo(reader *bufio.Reader, prompt string, def bool) bool {
	hint := "y/N"
	if def {
		hint = "Y/n"
	}
	fmt.Printf("%s [%s]: ", prompt, hint)
	line, err := reader.ReadString('\n')
	if err != nil {
		return def
	}
	s := strings.TrimSpace(strings.ToLower(line))
	if s == "" {
		return def
	}
	return s == "y" || s == "yes"
}
