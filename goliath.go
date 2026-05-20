package main

import (
	"bufio"
	"fmt"
	"time"
)

type goliathConfig struct {
	IntervalMinSec int
	IntervalMaxSec int
	ReleaseMinMs   int
	ReleaseMaxMs   int
}

var gCfg = goliathConfig{
	IntervalMinSec: 60,
	IntervalMaxSec: 120,
	ReleaseMinMs:   80,
	ReleaseMaxMs:   250,
}

func printGoliathConfig() {
	fmt.Println("---- Goliath timings ----")
	fmt.Printf("Delay before next W release: %d..%d sec\n", gCfg.IntervalMinSec, gCfg.IntervalMaxSec)
	fmt.Printf("W release duration:          %d..%d ms\n", gCfg.ReleaseMinMs, gCfg.ReleaseMaxMs)
}

func editGoliathConfig(reader *bufio.Reader) {
	fmt.Println()
	fmt.Println("== Edit Goliath timings (Enter - keep default) ==")

	gCfg.IntervalMinSec, gCfg.IntervalMaxSec = promptRange(
		reader, "Delay between W releases (sec)", gCfg.IntervalMinSec, gCfg.IntervalMaxSec)

	gCfg.ReleaseMinMs, gCfg.ReleaseMaxMs = promptRange(
		reader, "W release duration (ms)", gCfg.ReleaseMinMs, gCfg.ReleaseMaxMs)

	fmt.Println()
	fmt.Println("Done, final settings:")
	printGoliathConfig()
	fmt.Println()
}

func startGoliath() {
	mu.Lock()
	if running {
		mu.Unlock()
		logStatus("Macro is already running")
		return
	}
	running = true
	stopCh = make(chan struct{})
	mu.Unlock()

	logStatus("Goliath started: holding W")

	go func() {
		defer func() {
			keyUpFn(vkW)
			logStatus("W released, macro stopped")
		}()

		keyDown(vkW)

		for {
			wait := randIntInRange(gCfg.IntervalMinSec, gCfg.IntervalMaxSec)
			logStatus(fmt.Sprintf("Waiting %d sec before next W release", wait))

			if sleepOrStop(time.Duration(wait) * time.Second) {
				return
			}

			rel := randomDuration(gCfg.ReleaseMinMs, gCfg.ReleaseMaxMs)
			logStatus(fmt.Sprintf("Releasing W for %.0f ms", rel.Seconds()*1000))

			keyUpFn(vkW)

			if sleepOrStop(rel) {
				return
			}

			keyDown(vkW)
		}
	}()
}

func stopGoliath() {
	mu.Lock()
	defer mu.Unlock()
	if !running {
		return
	}
	running = false
	close(stopCh)
	keyUpFn(vkW)
	logStatus("Stopping Goliath")
}

func toggleGoliath() {
	mu.Lock()
	isRunning := running
	mu.Unlock()
	if isRunning {
		stopGoliath()
	} else {
		startGoliath()
	}
}
