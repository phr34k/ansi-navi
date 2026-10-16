//go:build !windows

package ui

import (
	"bufio"
	"io"
	"os"
	"os/signal"
)

// waitForExitKey waits until the user presses Enter or Ctrl+C at the pause
// prompt, then returns so ansi-navi can resume its TUI.
func waitForExitKey(r io.Reader) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	defer signal.Stop(sigCh)

	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = bufio.NewReader(r).ReadString('\n')
	}()
	select {
	case <-done:
	case <-sigCh:
	}
}
