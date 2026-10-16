// Command ansi-navi is a keyboard-driven TUI launcher for Ansible playbooks.
// Run it from an Ansible project root, or install system-wide and point it
// at a project with --project:
//
//	ansi-navi                        # use the current directory
//	ansi-navi --project /etc/ansible # run against a project from anywhere
package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/phr34k/ansi-navi/internal/ansible"
	"github.com/phr34k/ansi-navi/internal/ui"
)

func main() {
	var project, limit string
	flag.StringVar(&project, "project", "",
		"Ansible project directory; used to locate playbooks and inventory "+
			"(defaults to the current directory)")
	flag.StringVar(&limit, "limit", "",
		"Comma-separated hosts or groups to pre-populate the limit field "+
			"when the run wizard opens (e.g. --limit web1,web2 or --limit all)")
	flag.Parse()

	if err := ansible.CheckAnsible(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	root := project
	if root == "" {
		root = "."
	}
	ansible.SetProject(project)

	p := tea.NewProgram(ui.New(root, limit), tea.WithAltScreen(), tea.WithMouseAllMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "ansi-navi:", err)
		os.Exit(1)
	}
}
