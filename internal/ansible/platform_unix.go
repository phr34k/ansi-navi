//go:build !windows

package ansible

import (
	"fmt"
	"os/exec"
	"strings"
)

// CheckAnsible verifies ansible-playbook is on PATH.
func CheckAnsible() error {
	if _, err := exec.LookPath("ansible-playbook"); err != nil {
		return fmt.Errorf("'ansible-playbook' not found on PATH")
	}
	return nil
}

// ansibleArgv runs ansible-playbook directly on POSIX systems.
func ansibleArgv(args []string) []string {
	return append([]string{"ansible-playbook"}, args...)
}

// quoteArgs renders argv with POSIX shell quoting (equivalent of shlex.quote).
func quoteArgs(argv []string) string {
	parts := make([]string, len(argv))
	for i, a := range argv {
		parts[i] = shellQuote(a)
	}
	return strings.Join(parts, " ")
}

func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	if !strings.ContainsAny(s, " \t\n\"'\\$`&|;<>()*?!#~[]{}") {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
