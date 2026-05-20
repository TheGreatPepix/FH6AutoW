package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

// ---------------- WinAPI ----------------

const (
	inputKeyboard = 1
	keyEventKeyUp = 0x0002

	vkUp = 0x26
	vkW  = 0x57

	modControl = 0x0002
	wmHotkey   = 0x0312
)

var (
	user32             = syscall.NewLazyDLL("user32.dll")
	procSendInput      = user32.NewProc("SendInput")
	procRegisterHotKey = user32.NewProc("RegisterHotKey")
	procGetMessageW    = user32.NewProc("GetMessageW")
)

type keyboardInput struct {
	wVk         uint16
	wScan       uint16
	dwFlags     uint32
	time        uint32
	dwExtraInfo uintptr
}

type input struct {
	inputType uint32
	_pad      uint32
	union     [32]byte
}

func sendKey(vk uint16, keyUp bool) {
	var inp input
	inp.inputType = inputKeyboard

	ki := (*keyboardInput)(unsafe.Pointer(&inp.union[0]))
	ki.wVk = vk
	if keyUp {
		ki.dwFlags = keyEventKeyUp
	}

	procSendInput.Call(1, uintptr(unsafe.Pointer(&inp)), unsafe.Sizeof(inp))
}

func keyDown(vk uint16) { sendKey(vk, false) }
func keyUpFn(vk uint16) { sendKey(vk, true) }

type point struct {
	x int32
	y int32
}

type msg struct {
	hwnd    uintptr
	message uint32
	_pad1   uint32
	wParam  uintptr
	lParam  uintptr
	time    uint32
	_pad2   uint32
	pt      point
}

func hotkeyListener(onHotkey func()) {
	runtime.LockOSThread()

	ret, _, err := procRegisterHotKey.Call(0, 1, modControl, vkUp)
	if ret == 0 {
		fmt.Printf("RegisterHotKey failed: %v\n", err)
		return
	}

	var m msg
	for {
		r, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 {
			return
		}
		if m.message == wmHotkey {
			onHotkey()
		}
	}
}

// ---------------- Config ----------------

type config struct {
	IntervalMinSec int
	IntervalMaxSec int

	ReleaseMinMs int
	ReleaseMaxMs int
}

var cfg = config{
	IntervalMinSec: 60,
	IntervalMaxSec: 120,
	ReleaseMinMs:   80,
	ReleaseMaxMs:   250,
}

func printConfig() {
	fmt.Println("---- Timing ----")
	fmt.Printf("Delay before next W release: %d..%d sec\n", cfg.IntervalMinSec, cfg.IntervalMaxSec)
	fmt.Printf("W release duration:          %d..%d ms\n", cfg.ReleaseMinMs, cfg.ReleaseMaxMs)
}

// ---------------- Macro ----------------

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

func startMacro() {
	mu.Lock()
	if running {
		mu.Unlock()
		logStatus("Macro already running")
		return
	}
	running = true
	stopCh = make(chan struct{})
	mu.Unlock()

	logStatus("Macro started: holding W")

	go func() {
		defer func() {
			keyUpFn(vkW)
			logStatus("W released, macro stopped")
		}()

		keyDown(vkW)

		for {
			wait := randIntInRange(cfg.IntervalMinSec, cfg.IntervalMaxSec)
			logStatus(fmt.Sprintf("Waiting %d sec before next W release", wait))

			if sleepOrStop(time.Duration(wait) * time.Second) {
				return
			}

			rel := randomDuration(cfg.ReleaseMinMs, cfg.ReleaseMaxMs)
			logStatus(fmt.Sprintf("Releasing W for %.0f ms", rel.Seconds()*1000))

			keyUpFn(vkW)

			if sleepOrStop(rel) {
				return
			}

			keyDown(vkW)
		}
	}()
}

func stopMacro() {
	mu.Lock()
	defer mu.Unlock()
	if !running {
		return
	}
	running = false
	close(stopCh)
	keyUpFn(vkW)
	logStatus("Stopping macro")
}

func toggleMacro() {
	mu.Lock()
	isRunning := running
	mu.Unlock()
	if isRunning {
		stopMacro()
	} else {
		startMacro()
	}
}

// ---------------- CLI ----------------

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
		fmt.Println("  -> Invalid input, using default:", def)
		return def
	}
	return v
}

func promptRange(reader *bufio.Reader, label string, defMin, defMax int) (int, int) {
	fmt.Printf("  %s:\n", label)
	min := readInt(reader, "    min", defMin)
	max := readInt(reader, "    max", defMax)
	if min > max {
		fmt.Println("    min > max, swapping values")
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

func editConfig(reader *bufio.Reader) {
	fmt.Println()
	fmt.Println("== Timing settings (Press Enter to keep defaults) ==")

	cfg.IntervalMinSec, cfg.IntervalMaxSec = promptRange(
		reader, "Delay between W releases (sec)", cfg.IntervalMinSec, cfg.IntervalMaxSec)

	cfg.ReleaseMinMs, cfg.ReleaseMaxMs = promptRange(
		reader, "W release duration (ms)", cfg.ReleaseMinMs, cfg.ReleaseMaxMs)

	fmt.Println()
	fmt.Println("Done. Final settings:")
	printConfig()
	fmt.Println()
}

func main() {
	rand.Seed(time.Now().UnixNano())
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("==== Hold-W Macro (terminal) ====")
	fmt.Println("Logic: constantly holding W, occasionally releasing it briefly")
	fmt.Println("Hotkey: Ctrl + ↑ — start/stop")
	fmt.Println("Commands: start | stop | edit | show | quit")
	fmt.Println()

	printConfig()
	fmt.Println()

	if askYesNo(reader, "Edit timings before start?", false) {
		editConfig(reader)
	}

	go hotkeyListener(toggleMacro)

	for {
		fmt.Print("> ")
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		switch strings.TrimSpace(strings.ToLower(line)) {
		case "start", "s":
			startMacro()
		case "stop", "x":
			stopMacro()
		case "edit", "e":
			if running {
				fmt.Println("Stop the macro first")
				continue
			}
			editConfig(reader)
		case "show":
			printConfig()
		case "quit", "q", "exit":
			stopMacro()
			fmt.Println("Exiting")
			return
		case "":
		default:
			fmt.Println("Commands: start | stop | edit | show | quit")
		}
	}
}
