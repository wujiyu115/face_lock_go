package main

import (
	"strings"
	"syscall"
	"unsafe"

	"github.com/mitchellh/go-ps"
	log "github.com/sirupsen/logrus"
)

var (
	user32                  = syscall.MustLoadDLL("user32.dll")
	kernel32                = syscall.MustLoadDLL("kernel32.dll")
	getLastInputInfo        = user32.MustFindProc("GetLastInputInfo")
	getTickCount            = kernel32.MustFindProc("GetTickCount")
	mod                     = syscall.NewLazyDLL("user32.dll")
	procGetWindowText       = mod.NewProc("GetWindowTextW")
	procGetWindowTextLength = mod.NewProc("GetWindowTextLengthW")
	lastInputInfo           struct {
		cbSize uint32
		dwTime uint32
	}
)

type (
	HANDLE uintptr
	HWND   HANDLE
)

// checkIfError should be used to naively panics if an error is not nil.
func checkIfError(err error) bool {
	if err == nil {
		return false
	}
	log.Errorf("error: %s", err)
	return true
}

func getIdleTime() float32 {
	lastInputInfo.cbSize = uint32(unsafe.Sizeof(lastInputInfo))
	currentTickCount, _, _ := getTickCount.Call()
	r1, _, err := getLastInputInfo.Call(uintptr(unsafe.Pointer(&lastInputInfo)))
	if r1 == 0 {
		log.Error("error getting last input info: " + err.Error())
	}
	return float32((uint32(currentTickCount) - lastInputInfo.dwTime) / 1000)
}

func lockWorkStation() {
	lockWorkStation := user32.MustFindProc("LockWorkStation")
	lockWorkStation.Call()
}

// Locked checks if the loginui process is running, this is the simplest way
// to detect if a users desktop is locked. Doesn't work with multiple users on the
// same desktop.
// See https://stackoverflow.com/a/61681203
func winLocked() bool {
	processes, err := ps.Processes()
	if err != nil {
		log.Error("error winLocked: " + err.Error())
		return false
	}

	for _, process := range processes {
		if strings.ToLower(process.Executable()) == "logonui.exe" {
			return true
		}
	}

	return false
}

func GetWindowTextLength(hwnd HWND) int {
	ret, _, _ := procGetWindowTextLength.Call(
		uintptr(hwnd))

	return int(ret)
}

func GetWindowText(hwnd HWND) string {
	textLen := GetWindowTextLength(hwnd) + 1

	buf := make([]uint16, textLen)
	procGetWindowText.Call(
		uintptr(hwnd),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(textLen))
	return syscall.UTF16ToString(buf)
}

func getWindow(funcName string) uintptr {
	proc := mod.NewProc(funcName)
	hwnd, _, _ := proc.Call()
	return hwnd
}

func winLocked2() bool {
	if hwnd := getWindow("GetForegroundWindow"); hwnd != 0 {
		text := GetWindowText(HWND(hwnd))
		return text == "Windows 默认锁屏界面"
	}
	return false
}
