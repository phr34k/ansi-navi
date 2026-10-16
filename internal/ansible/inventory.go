package ansible

import (
	"bufio"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// candidateInventories lists filenames to probe in order when auto-detecting.
var candidateInventories = []string{
	"inventory.yml", "inventory.yaml", "inventory.ini",
	"hosts.yml", "hosts.yaml", "hosts", "inventory",
}

// DetectInventory returns the first inventory file found in root, or "".
func DetectInventory(root string) string {
	for _, name := range candidateInventories {
		if _, err := os.Stat(filepath.Join(root, name)); err == nil {
			return name
		}
	}
	return ""
}

// LoadHosts returns host names and group names from an inventory file.
// It tries YAML first, then INI format.
func LoadHosts(path string) (hosts, groups []string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil
	}

	var inv map[string]interface{}
	if err := yaml.Unmarshal(data, &inv); err == nil && len(inv) > 0 {
		h, g := hostsFromYAML(inv)
		if len(h) > 0 || len(g) > 0 {
			return h, g
		}
	}

	return hostsFromINI(string(data))
}

// hostsFromYAML extracts hosts and groups from an Ansible YAML inventory.
//
// Standard structure:
//
//	all:
//	  hosts:
//	    web1.example.com:
//	  children:
//	    webservers:
//	      hosts:
//	        web2.example.com:
func hostsFromYAML(inv map[string]interface{}) (hosts, groups []string) {
	seenH := map[string]bool{}
	seenG := map[string]bool{}

	var walkGroup func(name string, m map[string]interface{})
	walkGroup = func(name string, m map[string]interface{}) {
		if name != "" && name != "all" && !seenG[name] {
			seenG[name] = true
			groups = append(groups, name)
		}
		if hh, ok := m["hosts"]; ok {
			if hmap, ok := hh.(map[string]interface{}); ok {
				for h := range hmap {
					if !seenH[h] {
						seenH[h] = true
						hosts = append(hosts, h)
					}
				}
			}
		}
		if cc, ok := m["children"]; ok {
			if cmap, ok := cc.(map[string]interface{}); ok {
				for gname, gval := range cmap {
					if gm, ok := gval.(map[string]interface{}); ok {
						walkGroup(gname, gm)
					} else if !seenG[gname] {
						seenG[gname] = true
						groups = append(groups, gname)
					}
				}
			}
		}
	}

	for topKey, topVal := range inv {
		if gm, ok := topVal.(map[string]interface{}); ok {
			walkGroup(topKey, gm)
		}
	}

	sort.Strings(hosts)
	sort.Strings(groups)
	return hosts, groups
}

// hostsFromINI extracts hosts and groups from an Ansible INI inventory.
//
// Example:
//
//	[webservers]
//	web1.example.com
//	web2.example.com ansible_user=deploy
func hostsFromINI(data string) (hosts, groups []string) {
	seenH := map[string]bool{}
	seenG := map[string]bool{}
	var curGroup string

	scanner := bufio.NewScanner(strings.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			g := strings.TrimSpace(line[1 : len(line)-1])
			g = strings.Split(g, ":")[0] // strip :vars or :children suffix
			curGroup = g
			if g != "" && !seenG[g] {
				seenG[g] = true
				groups = append(groups, g)
			}
			continue
		}
		if curGroup == "" {
			continue
		}
		host := strings.Fields(line)[0]
		if host != "" && !strings.HasPrefix(host, "#") && !seenH[host] {
			seenH[host] = true
			hosts = append(hosts, host)
		}
	}

	sort.Strings(hosts)
	sort.Strings(groups)
	return hosts, groups
}
