// Package ui is the Bubble Tea front-end for ansi-navi: a search-first palette
// that opens into a single-page run wizard for Ansible playbooks, with an
// inline host picker and a shell escape hatch. All data/exec logic lives in
// internal/ansible; this file is layout, flow, and keybindings only.
package ui

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/phr34k/ansi-navi/internal/ansible"
)

type mode int

const (
	modeSearch mode = iota
	modeWizard
	modePicker
	modeDonate
)

type result struct {
	e    ansible.Entry
	idxs []int
}

// Model is the whole application state.
type Model struct {
	root      string
	all       []ansible.Entry
	initLimit string // pre-populated from --limit flag

	width, height int

	// search
	query   string
	results []result
	sel     int

	// wizard
	mode     mode
	entry    ansible.Entry
	fields   []ansible.Field
	fi       int
	enumOpen bool
	enumIdx  int

	// inventory / hosts
	detectedInventory string
	hosts, groups     []string
	hostsState        string // "loading" | "ready"
	wantPicker        bool

	// picker
	pickItems   []pickItem
	pickChecked map[int]bool
	pickIdx     int

	// donate dialog
	donateBtnIdx int

	status    string
	statusErr bool
}

type pickItem struct {
	group bool
	val   string
}

type hostsMsg struct{ hosts, groups []string }
type clearStatusMsg struct{}
type shellDoneMsg struct {
	dump    string
	cleanup func()
}
type runDoneMsg struct {
	name string
	code int
}

// New builds the initial model. limit is an optional comma-separated list of
// hosts/groups to pre-populate the limit field when the wizard opens.
func New(root, limit string) Model {
	startMode := modeSearch
	if shouldShowDonateDialog(incrementStartupCount()) {
		startMode = modeDonate
	}
	m := Model{
		root:              root,
		all:               ansible.LoadAll(root),
		mode:              startMode,
		initLimit:         limit,
		detectedInventory: ansible.DetectInventory(root),
		hostsState:        "loading",
		pickChecked:       map[int]bool{},
		donateBtnIdx:      1,
	}
	m.refilter()
	return m
}

func (m Model) Init() tea.Cmd {
	return m.loadHostsCmd()
}

func (m Model) loadHostsCmd() tea.Cmd {
	root := m.root
	inv := m.detectedInventory
	return func() tea.Msg {
		if inv == "" {
			return hostsMsg{nil, nil}
		}
		h, g := ansible.LoadHosts(filepath.Join(root, inv))
		return hostsMsg{h, g}
	}
}

// ------------------------------------------------------------------ Update --

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case hostsMsg:
		m.hosts, m.groups = msg.hosts, msg.groups
		m.hostsState = "ready"
		if m.wantPicker && m.mode == modeWizard && m.currentField().Role == "limit" {
			m.openPicker()
		}
		m.wantPicker = false
		return m, nil

	case shellDoneMsg:
		n := ansible.MergeEnvDump(msg.dump)
		if msg.cleanup != nil {
			msg.cleanup()
		}
		if n > 0 {
			m.setStatus(fmt.Sprintf("back from shell · %d env var(s) updated", n), false)
		} else {
			m.setStatus("back from shell", false)
		}
		return m, clearStatusAfter(3 * time.Second)

	case runDoneMsg:
		m.mode = modeSearch
		m.setStatus(fmt.Sprintf("last run of %s exited %d", msg.name, msg.code), msg.code != 0)
		return m, nil

	case clearStatusMsg:
		m.status = ""
		return m, nil

	case tea.MouseMsg:
		if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
			cmd := m.handleMouseClick(msg.X, msg.Y)
			return m, cmd
		}
		return m, nil

	case tea.KeyMsg:
		var cmd tea.Cmd
		switch m.mode {
		case modeSearch:
			cmd = m.handleSearchKey(msg)
		case modeWizard:
			cmd = m.handleWizardKey(msg)
		case modePicker:
			m.handlePickerKey(msg)
		case modeDonate:
			cmd = m.handleDonateKey(msg)
		}
		return m, cmd
	}
	return m, nil
}

// ------------------------------------------------------------------ search --

func (m *Model) refilter() {
	q := strings.TrimSpace(m.query)
	type scored struct {
		score float64
		e     ansible.Entry
		idxs  []int
	}
	var ss []scored
	for _, e := range m.all {
		if q == "" {
			ss = append(ss, scored{0, e, nil})
			continue
		}
		if mm, ok := ansible.Fuzzy(q, e.Name); ok {
			ss = append(ss, scored{mm.Score, e, mm.Indexes})
		} else if e.Path != e.Name {
			if pm, ok := ansible.Fuzzy(q, e.Path); ok {
				ss = append(ss, scored{pm.Score - 50, e, nil})
			} else if e.Desc != "" {
				if dm, ok := ansible.Fuzzy(q, e.Desc); ok {
					ss = append(ss, scored{dm.Score - 100, e, nil})
				}
			}
		} else if e.Desc != "" {
			if dm, ok := ansible.Fuzzy(q, e.Desc); ok {
				ss = append(ss, scored{dm.Score - 100, e, nil})
			}
		}
	}
	if q != "" {
		sort.SliceStable(ss, func(i, j int) bool {
			if ss[i].score != ss[j].score {
				return ss[i].score > ss[j].score
			}
			return ss[i].e.Path < ss[j].e.Path
		})
	}
	m.results = make([]result, len(ss))
	for i, s := range ss {
		m.results[i] = result{s.e, s.idxs}
	}
	if m.sel > len(m.results)-1 {
		m.sel = max(0, len(m.results)-1)
	}
}

func (m *Model) current() (ansible.Entry, bool) {
	if len(m.results) == 0 {
		return ansible.Entry{}, false
	}
	return m.results[min(m.sel, len(m.results)-1)].e, true
}

func (m *Model) handleSearchKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "ctrl+c", "esc":
		if m.query != "" {
			m.query, m.sel = "", 0
			m.refilter()
			return nil
		}
		return tea.Quit
	case "ctrl+t":
		return m.openShellCmd()
	case "ctrl+d":
		if err := openBrowser(donateURL); err != nil {
			m.setStatus("could not open browser", true)
		} else {
			m.setStatus("opening donation page…", false)
		}
		return clearStatusAfter(3 * time.Second)
	case "up":
		m.sel = max(0, m.sel-1)
	case "down":
		m.sel = min(len(m.results)-1, m.sel+1)
	case "pgup":
		m.sel = max(0, m.sel-10)
	case "pgdown":
		m.sel = min(len(m.results)-1, m.sel+10)
	case "enter":
		m.openWizard()
	case "backspace":
		if r := []rune(m.query); len(r) > 0 {
			m.query = string(r[:len(r)-1])
			m.sel = 0
			m.refilter()
		}
	default:
		if typed := typable(msg); typed != "" {
			m.query += typed
			m.sel = 0
			m.refilter()
		}
	}
	return nil
}

func typable(msg tea.KeyMsg) string {
	if msg.Type != tea.KeyRunes && msg.Type != tea.KeySpace {
		return ""
	}
	var b strings.Builder
	for _, r := range msg.String() {
		if r >= 0x20 && r != 0x7f {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// ------------------------------------------------------------------ wizard --

func (m *Model) currentField() ansible.Field {
	if m.fi < len(m.fields) {
		return m.fields[m.fi]
	}
	return ansible.Field{}
}

func (m *Model) openWizard() {
	e, ok := m.current()
	if !ok {
		return
	}
	m.entry = e
	m.fields = ansible.BuildFields()

	// Pre-populate --inventory with the auto-detected file.
	if m.detectedInventory != "" {
		for i := range m.fields {
			if m.fields[i].Name == "inventory" && m.fields[i].Value == "" {
				m.fields[i].Value = m.detectedInventory
				break
			}
		}
	}
	// Pre-populate --limit from CLI flag.
	if m.initLimit != "" {
		for i := range m.fields {
			if m.fields[i].Role == "limit" && m.fields[i].Value == "" {
				m.fields[i].Value = m.initLimit
				break
			}
		}
	}

	m.fields = append(m.fields,
		ansible.Field{Name: "Run", Role: "button", Widget: "button"},
		ansible.Field{Name: "Cancel", Role: "button", Widget: "button"})
	m.fi = 0
	m.enumOpen = false
	m.mode = modeWizard
	m.status = ""
}

func (m *Model) handleWizardKey(msg tea.KeyMsg) tea.Cmd {
	if m.enumOpen {
		m.handleEnumKey(msg)
		return nil
	}
	f := &m.fields[m.fi]
	switch msg.String() {
	case "ctrl+t":
		return m.openShellCmd()
	case "ctrl+r":
		return m.runWizard()
	case "ctrl+c", "esc":
		m.wantPicker = false
		m.mode = modeSearch
		m.status = ""
		return nil
	case "down", "tab":
		m.wantPicker = false
		m.fi = (m.fi + 1) % len(m.fields)
		return nil
	case "up", "shift+tab":
		m.wantPicker = false
		m.fi = (m.fi - 1 + len(m.fields)) % len(m.fields)
		return nil
	}

	switch f.Widget {
	case "button":
		switch msg.String() {
		case "enter", " ":
			if f.Name == "Run" {
				return m.runWizard()
			}
			m.mode = modeSearch
			m.status = ""
		case "left":
			if prev := (m.fi - 1 + len(m.fields)) % len(m.fields); m.fields[prev].Widget == "button" {
				m.fi = prev
			}
		case "right":
			if next := (m.fi + 1) % len(m.fields); m.fields[next].Widget == "button" {
				m.fi = next
			}
		}
	case "enum":
		switch msg.String() {
		case " ":
			m.enumOpen = true
			m.enumIdx = f.CI
		case "enter":
			m.fi = (m.fi + 1) % len(m.fields)
		}
	default: // text
		if f.Role == "limit" {
			switch msg.String() {
			case "enter":
				m.fi = (m.fi + 1) % len(m.fields)
				return nil
			case " ":
				if m.hostsState == "ready" {
					m.openPicker()
				} else {
					m.wantPicker = true
					m.setStatus("hosts still loading…", false)
				}
				return nil
			}
		}
		m.editText(f, msg)
	}
	return nil
}

func (m *Model) editText(f *ansible.Field, msg tea.KeyMsg) {
	switch msg.String() {
	case "enter":
		m.fi = (m.fi + 1) % len(m.fields)
	case "backspace":
		if r := []rune(f.Value); len(r) > 0 {
			f.Value = string(r[:len(r)-1])
		}
	default:
		f.Value += typable(msg)
	}
}

func (m *Model) handleEnumKey(msg tea.KeyMsg) {
	f := &m.fields[m.fi]
	n := len(f.Choices)
	if n == 0 {
		m.enumOpen = false
		return
	}
	switch msg.String() {
	case "up", "shift+tab":
		m.enumIdx = (m.enumIdx - 1 + n) % n
	case "down", "tab":
		m.enumIdx = (m.enumIdx + 1) % n
	case "enter", " ":
		f.CI = m.enumIdx
		f.Value = f.Choices[f.CI]
		m.enumOpen = false
	case "esc", "ctrl+c":
		m.enumOpen = false
	}
}

func (m *Model) runWizard() tea.Cmd {
	if miss := ansible.MissingRequired(m.fields); len(miss) > 0 {
		m.setStatus("missing required: "+strings.Join(miss, ", "), true)
		for i, f := range m.fields {
			if contains(miss, f.Name) {
				m.fi = i
				break
			}
		}
		return nil
	}
	argv := ansible.BuildArgv(m.entry.Path, m.fields)
	return runExec(m.root, m.entry.Name, argv, ansible.Preview(argv))
}

func (m *Model) openShellCmd() tea.Cmd {
	cmd, dump, cleanup, err := ansible.ShellCmd()
	if err != nil {
		m.setStatus(fmt.Sprintf("cannot open shell: %v", err), true)
		return nil
	}
	return tea.ExecProcess(cmd, func(error) tea.Msg {
		return shellDoneMsg{dump: dump, cleanup: cleanup}
	})
}

// ------------------------------------------------------------------ picker --

func (m *Model) openPicker() {
	m.pickItems = m.pickItems[:0]
	for _, g := range m.groups {
		m.pickItems = append(m.pickItems, pickItem{group: true, val: g})
	}
	for _, h := range m.hosts {
		m.pickItems = append(m.pickItems, pickItem{group: false, val: h})
	}
	if len(m.pickItems) == 0 {
		m.setStatus("no hosts found in inventory", true)
		return
	}
	m.pickChecked = map[int]bool{}
	m.pickIdx = 0
	m.mode = modePicker
}

func (m *Model) handlePickerKey(msg tea.KeyMsg) {
	switch msg.String() {
	case "up", "k":
		m.pickIdx = max(0, m.pickIdx-1)
	case "down", "j":
		m.pickIdx = min(len(m.pickItems)-1, m.pickIdx+1)
	case "home":
		m.pickIdx = 0
	case "end":
		m.pickIdx = len(m.pickItems) - 1
	case " ":
		m.pickChecked[m.pickIdx] = !m.pickChecked[m.pickIdx]
	case "a":
		for i := range m.pickItems {
			m.pickChecked[i] = true
		}
	case "n":
		m.pickChecked = map[int]bool{}
	case "enter":
		var picked []string
		for i, it := range m.pickItems {
			if m.pickChecked[i] {
				picked = append(picked, it.val)
			}
		}
		if m.fi < len(m.fields) {
			m.fields[m.fi].Value = strings.Join(picked, ",")
		}
		m.mode = modeWizard
	case "esc", "ctrl+c":
		m.mode = modeWizard
	}
}

func (m *Model) handleMouseClick(x, y int) tea.Cmd {
	if m.mode != modeSearch || m.status != "" {
		return nil
	}
	donateW := len([]rune(donateLabel))
	donateCol := m.width - donateW
	if y == m.height-1 && x >= donateCol && x < m.width {
		if err := openBrowser(donateURL); err != nil {
			m.setStatus("could not open browser", true)
		} else {
			m.setStatus("opening donation page…", false)
		}
		return clearStatusAfter(3 * time.Second)
	}
	return nil
}

func clearStatusAfter(d time.Duration) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(d)
		return clearStatusMsg{}
	}
}

// ------------------------------------------------------------------ donate --

func (m *Model) handleDonateKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "left", "right", "tab", "shift+tab":
		m.donateBtnIdx = 1 - m.donateBtnIdx
	case "enter", " ":
		m.mode = modeSearch
		if m.donateBtnIdx == 0 {
			if err := openBrowser(donateURL); err != nil {
				m.setStatus("could not open browser", true)
			} else {
				m.setStatus("opening donation page…", false)
			}
			return clearStatusAfter(3 * time.Second)
		}
	case "esc", "ctrl+c":
		m.mode = modeSearch
	}
	return nil
}

// ------------------------------------------------------------------ helpers --

func (m *Model) setStatus(s string, isErr bool) {
	m.status, m.statusErr = s, isErr
}

func contains(ss []string, s string) bool {
	for _, x := range ss {
		if x == s {
			return true
		}
	}
	return false
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

var (
	stAccent = lipgloss.NewStyle().Foreground(lipgloss.Color("13")).Bold(true)
	stDim    = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	stPlay   = lipgloss.NewStyle().Background(lipgloss.Color("2")).Foreground(lipgloss.Color("0")).Bold(true)
	stReq    = lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Bold(true)
	stOk     = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
	stErr    = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)
	stSel    = lipgloss.NewStyle().Background(lipgloss.Color("5")).Foreground(lipgloss.Color("0")).Bold(true)
	stBold   = lipgloss.NewStyle().Bold(true)
)
