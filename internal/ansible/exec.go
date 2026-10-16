package ansible

import (
	"os"
	"os/exec"
	"strings"
)

// projectDir, when set via SetProject, is forwarded as context for resolving
// relative paths. An empty string means use the working directory.
var projectDir string

// SetProject records the Ansible project directory. An empty string means
// "use the working directory" (the default behavior).
func SetProject(dir string) { projectDir = dir }

// AnsiblePlaybookCmd builds an *exec.Cmd for ansible-playbook <argv...>,
// wrapped for the platform (on Windows, runs through WSL). Std streams are
// left nil so tea.ExecProcess can attach the terminal.
//
// argv and the inherited environment are scrubbed of NUL bytes to prevent
// silent failures on Windows (CreateProcess rejects NUL in any argument).
func AnsiblePlaybookCmd(root string, argv []string) *exec.Cmd {
	wrapped := stripNULs(ansibleArgv(argv))
	cmd := exec.Command(wrapped[0], wrapped[1:]...)
	cmd.Dir = root
	cmd.Env = CleanEnviron()
	return cmd
}

func stripNULs(argv []string) []string {
	out := make([]string, len(argv))
	for i, a := range argv {
		out[i] = strings.ReplaceAll(a, "\x00", "")
	}
	return out
}

// CleanEnviron returns os.Environ() with any NUL-containing entries dropped.
func CleanEnviron() []string {
	env := os.Environ()
	out := make([]string, 0, len(env))
	for _, e := range env {
		if !strings.ContainsRune(e, 0) {
			out = append(out, e)
		}
	}
	return out
}

// CheckPath verifies ansible-playbook (or wsl on Windows) is reachable.
// Implemented in platform_*.go as CheckAnsible.
var _ = exec.LookPath // keep import used
