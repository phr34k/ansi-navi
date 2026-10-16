package ui

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const donateURL = "https://github.com/sponsors/phr34k"

const donateEvery = 15

const (
	footerHints = " ↑↓ move · Enter configure & run · Ctrl+T shell · Esc clear / quit "
	donateLabel = "Ctrl+D donate"
)

var counterKey = []byte{
	0xa1, 0x2b, 0x8e, 0x63, 0xd4, 0x07, 0x1c, 0xe5,
	0x56, 0xfb, 0x48, 0x3d, 0x92, 0x7a, 0xcf, 0x5f,
	0xe1, 0x84, 0xb6, 0x2d, 0xa7, 0x30, 0x69, 0x95,
	0x4c, 0xf8, 0xd2, 0x1e, 0x58, 0x7b, 0x0a, 0x33,
}

func signCounter(n int) string {
	mac := hmac.New(sha256.New, counterKey)
	fmt.Fprintf(mac, "%d", n)
	return hex.EncodeToString(mac.Sum(nil))
}

func verifyCounter(n int, sig string) bool {
	got, err := hex.DecodeString(strings.TrimSpace(sig))
	if err != nil {
		return false
	}
	want, _ := hex.DecodeString(signCounter(n))
	return hmac.Equal(got, want)
}

func donateHyperlink(label string) string {
	return "\x1b]8;;" + donateURL + "\x1b\\" + label + "\x1b]8;;\x1b\\"
}

func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}

func startupCountFile() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "ansi-navi", "startups")
}

func incrementStartupCount() int {
	path := startupCountFile()
	if path == "" {
		return 0
	}

	var n int
	if data, err := os.ReadFile(path); err == nil {
		parts := strings.SplitN(strings.TrimSpace(string(data)), ":", 2)
		if len(parts) == 2 {
			fmt.Sscanf(parts[0], "%d", &n)
			if !verifyCounter(n, parts[1]) {
				n = donateEvery - 1
			}
		}
	}

	n++
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return n
	}
	os.WriteFile(path, []byte(fmt.Sprintf("%d:%s", n, signCounter(n))), 0o644) //nolint:errcheck
	return n
}

func shouldShowDonateDialog(count int) bool {
	return count > 0 && count%donateEvery == 0
}
