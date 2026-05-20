package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"sort"
	"time"
)

type crushConfig struct {
	DriveMinSec int
	DriveMaxSec int

	MicroPauseMaxCount     int // 0..N per drive
	MicroPauseReleaseMinMs int
	MicroPauseReleaseMaxMs int

	STapMaxCount  int // 1..N taps
	STapWaitMinS  int
	STapWaitMaxS  int
	STapHoldMinMs int
	STapHoldMaxMs int

	HumanDelayMinMs int
	HumanDelayMaxMs int

	KeyTapMinMs int
	KeyTapMaxMs int

	FirstEnterMinSec int
	FirstEnterMaxSec int

	SecondEnterMinSec int
	SecondEnterMaxSec int

	EndDelayMinMs int
	EndDelayMaxMs int
}

var cCfg = crushConfig{
	DriveMinSec: 65,
	DriveMaxSec: 70,

	MicroPauseMaxCount:     2,
	MicroPauseReleaseMinMs: 80,
	MicroPauseReleaseMaxMs: 250,

	STapMaxCount:  2,
	STapWaitMinS:  3,
	STapWaitMaxS:  8,
	STapHoldMinMs: 80,
	STapHoldMaxMs: 180,

	HumanDelayMinMs: 150,
	HumanDelayMaxMs: 800,

	KeyTapMinMs: 40,
	KeyTapMaxMs: 80,

	FirstEnterMinSec: 2,
	FirstEnterMaxSec: 4,

	SecondEnterMinSec: 8,
	SecondEnterMaxSec: 11,

	EndDelayMinMs: 300,
	EndDelayMaxMs: 1500,
}

func printCrushConfig() {
	fmt.Println("---- Crush Combo timings ----")
	fmt.Printf("1)  W hold:                   %d..%d sec\n", cCfg.DriveMinSec, cCfg.DriveMaxSec)
	fmt.Printf("2)  W micro-releases (max):   %d\n", cCfg.MicroPauseMaxCount)
	fmt.Printf("3)  W micro-pause duration:   %d..%d ms\n", cCfg.MicroPauseReleaseMinMs, cCfg.MicroPauseReleaseMaxMs)
	fmt.Printf("4)  S taps (max):             %d\n", cCfg.STapMaxCount)
	fmt.Printf("5)  Wait before each S tap:   %d..%d sec\n", cCfg.STapWaitMinS, cCfg.STapWaitMaxS)
	fmt.Printf("6)  S tap duration:           %d..%d ms\n", cCfg.STapHoldMinMs, cCfg.STapHoldMaxMs)
	fmt.Printf("7)  Delay before X:           %d..%d ms\n", cCfg.HumanDelayMinMs, cCfg.HumanDelayMaxMs)
	fmt.Printf("8)  KeyTap hold:              %d..%d ms\n", cCfg.KeyTapMinMs, cCfg.KeyTapMaxMs)
	fmt.Printf("9)  Wait until first Enter:   %d..%d sec\n", cCfg.FirstEnterMinSec, cCfg.FirstEnterMaxSec)
	fmt.Printf("10) Wait until second Enter:  %d..%d sec\n", cCfg.SecondEnterMinSec, cCfg.SecondEnterMaxSec)
	fmt.Printf("11) Delay before next cycle:  %d..%d ms\n", cCfg.EndDelayMinMs, cCfg.EndDelayMaxMs)
}

func editCrushConfig(reader *bufio.Reader) {
	fmt.Println()
	fmt.Println("== Edit Crush Combo timings (Enter - keep default) ==")

	cCfg.DriveMinSec, cCfg.DriveMaxSec = promptRange(reader, "W hold (sec)", cCfg.DriveMinSec, cCfg.DriveMaxSec)

	cCfg.MicroPauseMaxCount = readInt(reader, "W micro-releases: max count per drive (0=off)", cCfg.MicroPauseMaxCount)

	if cCfg.MicroPauseMaxCount > 0 {
		cCfg.MicroPauseReleaseMinMs, cCfg.MicroPauseReleaseMaxMs = promptRange(
			reader, "W micro-pause duration (ms)", cCfg.MicroPauseReleaseMinMs, cCfg.MicroPauseReleaseMaxMs)
	}

	cCfg.STapMaxCount = readInt(reader, "S taps: max count per drive (0=off)", cCfg.STapMaxCount)

	if cCfg.STapMaxCount > 0 {
		cCfg.STapWaitMinS, cCfg.STapWaitMaxS = promptRange(
			reader, "Wait before each S tap (sec)", cCfg.STapWaitMinS, cCfg.STapWaitMaxS)
		cCfg.STapHoldMinMs, cCfg.STapHoldMaxMs = promptRange(
			reader, "S tap duration (ms)", cCfg.STapHoldMinMs, cCfg.STapHoldMaxMs)
	}

	cCfg.HumanDelayMinMs, cCfg.HumanDelayMaxMs = promptRange(
		reader, "Delay before X (ms)", cCfg.HumanDelayMinMs, cCfg.HumanDelayMaxMs)

	cCfg.KeyTapMinMs, cCfg.KeyTapMaxMs = promptRange(
		reader, "KeyTap hold (ms)", cCfg.KeyTapMinMs, cCfg.KeyTapMaxMs)

	cCfg.FirstEnterMinSec, cCfg.FirstEnterMaxSec = promptRange(
		reader, "Wait until first Enter (sec)", cCfg.FirstEnterMinSec, cCfg.FirstEnterMaxSec)

	cCfg.SecondEnterMinSec, cCfg.SecondEnterMaxSec = promptRange(
		reader, "Wait until second Enter (sec)", cCfg.SecondEnterMinSec, cCfg.SecondEnterMaxSec)

	cCfg.EndDelayMinMs, cCfg.EndDelayMaxMs = promptRange(
		reader, "Delay before next cycle (ms)", cCfg.EndDelayMinMs, cCfg.EndDelayMaxMs)

	fmt.Println()
	fmt.Println("Done, final settings:")
	printCrushConfig()
	fmt.Println()
}

func startCrush() {
	mu.Lock()
	if running {
		mu.Unlock()
		logStatus("Macro is already running")
		return
	}
	running = true
	stopCh = make(chan struct{})
	mu.Unlock()

	logStatus("Crush Combo started")

	go func() {
		defer func() {
			keyUpFn(vkW)
			keyUpFn(vkS)
			logStatus("Macro stopped, W released")
		}()

		cycle := 1

		for {
			// ---- DRIVE PHASE ----

			driveTime := randIntInRange(cCfg.DriveMinSec, cCfg.DriveMaxSec)
			logStatus(fmt.Sprintf("[Cycle %d] Holding W for %d sec", cycle, driveTime))

			driveStart := time.Now()
			deadline := driveStart.Add(time.Duration(driveTime) * time.Second)

			keyDown(vkW)

			// Events during W hold: micro-pauses and S taps.
			// Offsets are generated within [3, driveTime-3] sec from start.
			type driveEvent struct {
				offset time.Duration
				kind   string
			}

			var events []driveEvent

			microPauseCount := 0
			if cCfg.MicroPauseMaxCount > 0 {
				microPauseCount = rand.Intn(cCfg.MicroPauseMaxCount + 1)
			}

			for i := 0; i < microPauseCount; i++ {
				lo, hi := 3, driveTime-3
				if hi <= lo {
					break
				}
				off := rand.Intn(hi-lo+1) + lo
				events = append(events, driveEvent{
					offset: time.Duration(off) * time.Second,
					kind:   "micro_w",
				})
			}

			sTapCount := 0
			if cCfg.STapMaxCount > 0 {
				sTapCount = rand.Intn(cCfg.STapMaxCount) + 1
			}

			for i := 0; i < sTapCount; i++ {
				lo, hi := cCfg.STapWaitMinS, cCfg.STapWaitMaxS
				if hi > driveTime-3 {
					hi = driveTime - 3
				}
				if hi < lo {
					break
				}
				off := rand.Intn(hi-lo+1) + lo
				events = append(events, driveEvent{
					offset: time.Duration(off) * time.Second,
					kind:   "s_tap",
				})
			}

			sort.Slice(events, func(i, j int) bool {
				return events[i].offset < events[j].offset
			})

			for _, ev := range events {
				wait := time.Until(driveStart.Add(ev.offset))
				if wait > 0 {
					if sleepOrStop(wait) {
						return
					}
				}

				switch ev.kind {
				case "micro_w":
					logStatus("Micro-release W")
					keyUpFn(vkW)
					if sleepOrStop(randomDuration(cCfg.MicroPauseReleaseMinMs, cCfg.MicroPauseReleaseMaxMs)) {
						return
					}
					keyDown(vkW)
				case "s_tap":
					logStatus("Short S tap")
					keyDown(vkS)
					if sleepOrStop(randomDuration(cCfg.STapHoldMinMs, cCfg.STapHoldMaxMs)) {
						return
					}
					keyUpFn(vkS)
				}
			}

			remaining := time.Until(deadline)
			if remaining > 0 {
				if sleepOrStop(remaining) {
					return
				}
			}

			keyUpFn(vkW)
			logStatus(fmt.Sprintf("W released (actually held %.1f sec)", time.Since(driveStart).Seconds()))

			// ---- HUMAN DELAY BEFORE X ----

			humanDelay := randomDuration(cCfg.HumanDelayMinMs, cCfg.HumanDelayMaxMs)
			logStatus(fmt.Sprintf("Waiting %.0f ms before X", humanDelay.Seconds()*1000))
			if sleepOrStop(humanDelay) {
				return
			}
			logStatus("Pressing X")
			keyTap(vkX, cCfg.KeyTapMinMs, cCfg.KeyTapMaxMs)

			// ---- FIRST ENTER ----

			waitFirstEnter := randIntInRange(cCfg.FirstEnterMinSec, cCfg.FirstEnterMaxSec)
			logStatus(fmt.Sprintf("Waiting %d sec until first Enter", waitFirstEnter))
			if sleepOrStop(time.Duration(waitFirstEnter) * time.Second) {
				return
			}
			logStatus("Pressing first Enter")
			keyTap(vkReturn, cCfg.KeyTapMinMs, cCfg.KeyTapMaxMs)

			// ---- SECOND ENTER ----

			waitSecondEnter := randIntInRange(cCfg.SecondEnterMinSec, cCfg.SecondEnterMaxSec)
			logStatus(fmt.Sprintf("Waiting %d sec until second Enter", waitSecondEnter))
			if sleepOrStop(time.Duration(waitSecondEnter) * time.Second) {
				return
			}
			logStatus("Pressing second Enter")
			keyTap(vkReturn, cCfg.KeyTapMinMs, cCfg.KeyTapMaxMs)

			// ---- END DELAY ----

			endDelay := randomDuration(cCfg.EndDelayMinMs, cCfg.EndDelayMaxMs)
			logStatus(fmt.Sprintf("Pause %.0f ms before next cycle", endDelay.Seconds()*1000))
			if sleepOrStop(endDelay) {
				return
			}

			cycle++
		}
	}()
}

func stopCrush() {
	mu.Lock()
	defer mu.Unlock()
	if !running {
		return
	}
	running = false
	close(stopCh)
	keyUpFn(vkW)
	keyUpFn(vkS)
	logStatus("Stopping Crush Combo")
}

func toggleCrush() {
	mu.Lock()
	isRunning := running
	mu.Unlock()
	if isRunning {
		stopCrush()
	} else {
		startCrush()
	}
}
