//go:build windows

package ui

import (
	"io"
	"os"

	"golang.org/x/sys/windows"
)

const (
	enableProcessedInput = uint32(0x0001)
	enableLineInput      = uint32(0x0002)
	enableEchoInput      = uint32(0x0004)
)

// waitForExitKey waits until the user presses Enter or Ctrl+C at the pause
// prompt, then returns so ansi-navi can resume its TUI.
func waitForExitKey(r io.Reader) {
	f, ok := r.(*os.File)
	if !ok {
		waitForNewline(r)
		return
	}

	handle := windows.Handle(f.Fd())
	var oldMode uint32
	if err := windows.GetConsoleMode(handle, &oldMode); err != nil {
		waitForNewline(r)
		return
	}

	rawMode := oldMode &^ (enableProcessedInput | enableLineInput | enableEchoInput)
	if err := windows.SetConsoleMode(handle, rawMode); err != nil {
		waitForNewline(r)
		return
	}
	defer windows.SetConsoleMode(handle, oldMode) //nolint:errcheck

	buf := make([]byte, 1)
	for {
		n, err := f.Read(buf)
		if err != nil || n == 0 {
			return
		}
		if buf[0] == '\r' || buf[0] == '\n' || buf[0] == 3 {
			return
		}
	}
}

func waitForNewline(r io.Reader) {
	buf := make([]byte, 1)
	for {
		n, err := r.Read(buf)
		if err != nil || n == 0 {
			return
		}
		if buf[0] == '\r' || buf[0] == '\n' {
			return
		}
	}
}
