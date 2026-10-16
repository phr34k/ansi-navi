// Package ansible is the shared backend for ansi-navi: it discovers playbooks
// from the filesystem, parses inventory, and builds/runs ansible-playbook argv.
// It is deliberately free of any UI concern so it can be unit-tested on its own.
package ansible

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Entry is one playbook in the palette list.
type Entry struct {
	Name string // display name (first play's name, or relative file path)
	Desc string // one-line description (hosts target and play count)
	Path string // path relative to project root, always forward-slashes
}

// LoadAll scans root for Ansible playbooks and returns them sorted by path.
func LoadAll(root string) []Entry {
	var entries []Entry
	seen := map[string]bool{}

	scanDir(root, root, &entries, seen)

	// Also check a playbooks/ subdirectory if it exists.
	playDir := filepath.Join(root, "playbooks")
	if fi, err := os.Stat(playDir); err == nil && fi.IsDir() {
		scanDir(playDir, root, &entries, seen)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Path < entries[j].Path
	})
	return entries
}

// skipDirs are directories that never contain top-level playbooks.
var skipDirs = map[string]bool{
	"roles": true, "group_vars": true, "host_vars": true,
	"vars": true, "defaults": true, "tasks": true,
	"handlers": true, "templates": true, "files": true,
	"meta": true, "tests": true, ".git": true,
}

func scanDir(dir, root string, entries *[]Entry, seen map[string]bool) {
	infos, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, info := range infos {
		if info.IsDir() {
			if skipDirs[strings.ToLower(info.Name())] {
				continue
			}
		}
		name := info.Name()
		if info.IsDir() || (!strings.HasSuffix(name, ".yml") && !strings.HasSuffix(name, ".yaml")) {
			continue
		}
		full := filepath.Join(dir, name)
		rel, _ := filepath.Rel(root, full)
		rel = filepath.ToSlash(rel)
		if seen[rel] {
			continue
		}
		e, ok := parsePlaybook(full, rel)
		if !ok {
			continue
		}
		seen[rel] = true
		*entries = append(*entries, e)
	}
}

// parsePlaybook tries to parse path as an Ansible playbook.
// Returns ok=false if the file does not look like a playbook.
func parsePlaybook(path, rel string) (Entry, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Entry{}, false
	}

	// A playbook is a YAML list of play maps.
	var plays []map[string]interface{}
	if err := yaml.Unmarshal(data, &plays); err != nil || len(plays) == 0 {
		return Entry{}, false
	}

	first := plays[0]
	if first == nil {
		return Entry{}, false
	}

	// A play must have "hosts", "import_playbook", or "include_playbook".
	hostsRaw, hasHosts := first["hosts"]
	_, hasImport := first["import_playbook"]
	_, hasInclude := first["include_playbook"]
	if !hasHosts && !hasImport && !hasInclude {
		return Entry{}, false
	}

	hosts := anyToString(hostsRaw)
	displayName := rel
	if n, ok := first["name"].(string); ok && n != "" {
		displayName = n
	}

	var descParts []string
	if hosts != "" {
		descParts = append(descParts, "hosts: "+hosts)
	}
	if len(plays) > 1 {
		descParts = append(descParts, fmt.Sprintf("%d plays", len(plays)))
	}
	desc := strings.Join(descParts, " · ")

	return Entry{Name: displayName, Desc: desc, Path: rel}, true
}

func anyToString(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

func oneLine(s string) string {
	return strings.TrimSpace(strings.ReplaceAll(s, "\n", " "))
}
