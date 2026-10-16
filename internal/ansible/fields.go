package ansible

import "strings"

// Field is one editable row in the run form. Buttons are appended by the UI.
type Field struct {
	Name       string
	Role       string   // "limit" | "option" | "verbosity" | "extra" | "button"
	Widget     string   // "text" | "enum" | "button"
	Value      string
	Required   bool
	Desc       string
	Choices    []string // enum options (first entry is "" for unset)
	CI         int      // index into Choices for current enum selection
	Flag       string   // CLI flag, e.g. "--limit"
	TakesValue bool     // flag takes a value vs. boolean flag
}

type playbookOption struct {
	name       string
	flag       string
	takesValue bool
	desc       string
}

var options = []playbookOption{
	{"inventory", "--inventory", true, "Inventory file or comma-separated hosts. (required)"},
	{"limit", "--limit", true, "Further limit selected hosts or groups."},
	{"tags", "--tags", true, "Only run plays and tasks tagged with these values."},
	{"skip-tags", "--skip-tags", true, "Only run plays and tasks not tagged with these values."},
	{"extra-vars", "--extra-vars", true, "Set additional variables as key=value or JSON string."},
	{"check", "--check", false, "Dry run — don't make any changes."},
	{"diff", "--diff", false, "Show diffs for changed (templated) files."},
	{"become", "--become", false, "Run operations with privilege escalation."},
	{"become-user", "--become-user", true, "Run operations as this user (default: root)."},
	{"forks", "--forks", true, "Number of parallel processes (default: 5)."},
	{"start-at-task", "--start-at-task", true, "Start the playbook at the task matching this name."},
}

func makeField(o playbookOption) Field {
	if o.takesValue {
		role := "option"
		if o.name == "limit" {
			role = "limit"
		}
		return Field{
			Name: o.name, Role: role, Widget: "text",
			Flag: o.flag, TakesValue: true, Desc: o.desc,
			Required: o.name == "inventory",
		}
	}
	return Field{
		Name: o.name, Role: "option", Widget: "enum",
		Flag: o.flag, TakesValue: false, Desc: o.desc,
		Choices: []string{"", "true", "false"},
	}
}

// BuildFields produces the editable field list for any playbook.
// Ansible playbooks have no typed parameters — only CLI options are offered.
func BuildFields() []Field {
	var fields []Field
	for _, o := range options {
		fields = append(fields, makeField(o))
	}
	fields = append(fields, Field{
		Name: "verbose", Role: "verbosity", Widget: "enum",
		Desc:    "Verbosity level (empty = off).",
		Choices: []string{"", "-v", "-vv", "-vvv", "-vvvv"},
	})
	fields = append(fields, Field{
		Name: "extra args", Role: "extra", Widget: "text",
		Desc: "Any other raw ansible-playbook flags not listed above.",
	})
	return fields
}

// BuildArgv composes the ansible-playbook argument list.
// path is the playbook file (relative to project root, forward-slashes).
func BuildArgv(path string, fields []Field) []string {
	var argv []string
	var extra string
	for _, f := range fields {
		if f.Widget == "button" {
			continue
		}
		if f.Role == "extra" {
			extra = f.Value
			continue
		}
		v := strings.TrimSpace(f.Value)
		if f.Role == "verbosity" {
			if v != "" {
				argv = append(argv, v)
			}
			continue
		}
		if f.Role == "option" || f.Role == "limit" {
			if f.TakesValue {
				if v != "" {
					argv = append(argv, f.Flag, v)
				}
			} else if v == "true" {
				argv = append(argv, f.Flag)
			}
		}
	}
	if strings.TrimSpace(extra) != "" {
		argv = append(argv, splitArgs(extra)...)
	}
	argv = append(argv, path)
	return argv
}

// MissingRequired returns names of required fields left empty.
func MissingRequired(fields []Field) []string {
	var miss []string
	for _, f := range fields {
		if f.Widget != "button" && f.Required && strings.TrimSpace(f.Value) == "" {
			miss = append(miss, f.Name)
		}
	}
	return miss
}

// Preview renders argv as a copy-pasteable command line.
func Preview(argv []string) string {
	return "ansible-playbook " + quoteArgs(argv)
}

func splitArgs(s string) []string {
	fields, ok := shlexSplit(s)
	if !ok {
		return strings.Fields(s)
	}
	return fields
}

func shlexSplit(s string) ([]string, bool) {
	var out []string
	var cur strings.Builder
	inToken := false
	var quote rune
	escaped := false
	for _, r := range s {
		switch {
		case escaped:
			cur.WriteRune(r)
			escaped = false
			inToken = true
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				cur.WriteRune(r)
			}
		case r == '\\':
			escaped = true
			inToken = true
		case r == '\'' || r == '"':
			quote = r
			inToken = true
		case r == ' ' || r == '\t' || r == '\n':
			if inToken {
				out = append(out, cur.String())
				cur.Reset()
				inToken = false
			}
		default:
			cur.WriteRune(r)
			inToken = true
		}
	}
	if quote != 0 || escaped {
		return nil, false
	}
	if inToken {
		out = append(out, cur.String())
	}
	return out, true
}
