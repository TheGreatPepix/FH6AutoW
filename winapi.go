package main

import (
	"fmt"
	"runtime"
	"syscall"
	"time"
	"unsafe"
)

const (
	inputKeyboard = 1
	keyEventKeyUp = 0x0002

	vkReturn = 0x0D
	vkUp     = 0x26
	vkS      = 0x53
	vkW      = 0x57
	vkX      = 0x58

	modControl = 0x0002

	wmHotkey = 0x0312
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

type winInput struct {
	inputType uint32
	_pad      uint32
	union     [32]byte
}

func sendKey(vk uint16, keyUp bool) {
	var inp winInput
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

func keyTap(vk uint16, minMs, maxMs int) {
	keyDown(vk)
	time.Sleep(randomDuration(minMs, maxMs))
	keyUpFn(vk)
}

type point struct {
	x int32
	y int32
}

type winMsg struct {
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

	var m winMsg
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
