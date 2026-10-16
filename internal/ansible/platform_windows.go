package ansible

import (
	"fmt"
	"os/exec"
	"strings"
)

// CheckAnsible verifies wsl is available. Ansible is not natively available on
// Windows; ansi-navi runs it through the Windows Subsystem for Linux (WSL).
func CheckAnsible() error {
	if _, err := exec.LookPath("wsl"); err != nil {
		return fmt.Errorf("'wsl' not found on PATH — Ansible requires WSL on Windows.\n" +
			"Install WSL (https://aka.ms/wsl) and ensure ansible-playbook is available inside it.")
	}
	return nil
}

// ansibleArgv wraps argv to run ansible-playbook through WSL.
func ansibleArgv(args []string) []string {
	return append([]string{"wsl", "ansible-playbook"}, args...)
}

// quoteArgs renders argv as a Windows command line using list2cmdline quoting.
func quoteArgs(argv []string) string {
	var b strings.Builder
	for i, arg := range argv {
		if i > 0 {
			b.WriteByte(' ')
		}
		needQuote := arg == "" || strings.ContainsAny(arg, " \t")
		if needQuote {
			b.WriteByte('"')
		}
		var bs int
		for _, c := range arg {
			switch c {
			case '\\':
				bs++
			case '"':
				b.WriteString(strings.Repeat("\\", bs*2))
				bs = 0
				b.WriteString("\\\"")
			default:
				if bs > 0 {
					b.WriteString(strings.Repeat("\\", bs))
					bs = 0
				}
				b.WriteRune(c)
			}
		}
		if bs > 0 {
			n := bs
			if needQuote {
				n *= 2
			}
			b.WriteString(strings.Repeat("\\", n))
		}
		if needQuote {
			b.WriteByte('"')
		}
	}
	return b.String()
}
