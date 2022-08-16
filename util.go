package main

import (
	"strings"
	"syscall"
	"unsafe"

	"github.com/mitchellh/go-ps"
	log "github.com/sirupsen/logrus"
)

var (
	user32           = syscall.MustLoadDLL("user32.dll")
	kernel32         = syscall.MustLoadDLL("kernel32.dll")
	getLastInputInfo = user32.MustFindProc("GetLastInputInfo")
	getTickCount     = kernel32.MustFindProc("GetTickCount")
	lastInputInfo    struct {
		cbSize uint32
		dwTime uint32
	}
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
