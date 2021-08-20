package server

import (
	"golang.org/x/crypto/ssh/terminal"
	"os"
	"syscall"
	"unsafe"
)

func init() {
	// enable VT100 escape sequence on Windows Console.
	// credit: https://github.com/eliukblau/pixterm/blob/master/cmd/pixterm/pixterm_windows.go
	var consoleMode int32
	handle := int(os.Stdout.Fd())

	if terminal.IsTerminal(handle) {
		kernel32 := syscall.NewLazyDLL("kernel32.dll")
		procGetConsoleMode := kernel32.NewProc("GetConsoleMode")
		procSetConsoleMode := kernel32.NewProc("SetConsoleMode")

		procGetConsoleMode.Call(uintptr(handle), uintptr(unsafe.Pointer(&consoleMode)))
		consoleMode |= 0x0004
		procSetConsoleMode.Call(uintptr(handle), uintptr(consoleMode))
	}
}
