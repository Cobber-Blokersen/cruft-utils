//go:build windows

package main

import (
	"os"
	"syscall"
	"unsafe"
)

const enableVirtualTerminalProcessing = 0x0004

var (
	kernel32           = syscall.NewLazyDLL("kernel32.dll")
	procGetConsoleMode = kernel32.NewProc("GetConsoleMode")
	procSetConsoleMode = kernel32.NewProc("SetConsoleMode")
)

func initConsole() {
	handles := []uintptr{os.Stdout.Fd(), os.Stderr.Fd()}
	for _, handle := range handles {
		var mode uint32
		ok, _, _ := procGetConsoleMode.Call(handle, uintptr(unsafe.Pointer(&mode)))
		if ok == 0 {
			continue
		}
		procSetConsoleMode.Call(handle, uintptr(mode|enableVirtualTerminalProcessing))
	}
}
