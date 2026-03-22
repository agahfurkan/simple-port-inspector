package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/port-inspector/port-inspector/internal/process"
	"github.com/port-inspector/port-inspector/internal/scanner"
)

// View modes
type viewMode int

const (
	viewTable viewMode = iota
	viewDetail
	viewSearch
	viewConfirmKill
	viewHelp
)

// Focus panels (lazygit-style)
type panel int

const (
	panelProcesses panel = iota
	panelPorts
)

// Messages
type scanResultMsg *scanner.ScanResult
type scanErrMsg error
type killResultMsg struct {
	pid     int
	command string
	err     error
}
type tickMsg time.Time

// Model is the main Bubble Tea model.
type Model struct {
	// Data
	scanResult   *scanner.ScanResult
	filteredRows []scanner.PortEntry
	err          error

	// UI State
	mode         viewMode
	activePanel  panel
	cursor       int
	detailCursor int
	offset       int
	searchInput  textinput.Model
	searchQuery  string
	viewport     viewport.Model
	help         help.Model
	keys         keyMap

	// Confirm kill
	killTarget    *scanner.PortEntry
	forceKill     bool
	statusMessage string
	statusExpiry  time.Time

	// Dimensions
	width  int
	height int
	ready  bool

	// Auto-refresh
	autoRefresh bool
	lastScan    time.Time

	// Filter state
	showListening bool // only show LISTEN state
	showSystem    bool // show macOS system processes (default: hidden)
}

func NewModel() Model {
	ti := textinput.New()
	ti.Placeholder = "Search by port, process, or user..."
	ti.CharLimit = 64

	h := help.New()
	h.ShowAll = false

	return Model{
		searchInput:   ti,
		help:          h,
		keys:          defaultKeyMap(),
		autoRefresh:   false,
		showListening: false,
		showSystem:    false,
		mode:          viewTable,
		activePanel:   panelProcesses,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(doScan, tickCmd())
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true

		headerHeight := 3  // title bar
		footerHeight := 3  // status bar + help
		m.viewport = viewport.New(msg.Width, msg.Height-headerHeight-footerHeight)
		m.viewport.YPosition = headerHeight
		return m, nil

	case scanResultMsg:
		m.scanResult = msg
		m.lastScan = time.Now()
		m.err = nil
		m.applyFilter()
		return m, nil

	case scanErrMsg:
		m.err = msg
		return m, nil

	case killResultMsg:
		if msg.err != nil {
			m.statusMessage = fmt.Sprintf("✗ Failed to kill %s (PID %d): %v", msg.command, msg.pid, msg.err)
		} else {
			m.statusMessage = fmt.Sprintf("✓ Killed %s (PID %d)", msg.command, msg.pid)
		}
		m.statusExpiry = time.Now().Add(5 * time.Second)
		m.mode = viewTable
		m.killTarget = nil
		// Rescan after kill
		return m, doScan

	case tickMsg:
		// Auto-refresh every 3 seconds
		if m.autoRefresh && time.Since(m.lastScan) > 3*time.Second {
			return m, tea.Batch(doScan, tickCmd())
		}
		// Clear expired status messages
		if m.statusMessage != "" && time.Now().After(m.statusExpiry) {
			m.statusMessage = ""
		}
		return m, tickCmd()

	case tea.MouseMsg:
		return m.handleMouse(msg)

	case tea.KeyMsg:
		// Global keys
		switch {
		case key.Matches(msg, m.keys.Quit):
			if m.mode == viewSearch {
				m.mode = viewTable
				m.searchInput.Blur()
				return m, nil
			}
			if m.mode == viewConfirmKill {
				m.mode = viewTable
				m.killTarget = nil
				return m, nil
			}
			if m.mode == viewHelp {
				m.mode = viewTable
				return m, nil
			}
			if m.mode == viewDetail {
				m.mode = viewTable
				return m, nil
			}
			return m, tea.Quit
		}

		// Mode-specific keys
		switch m.mode {
		case viewSearch:
			return m.updateSearch(msg)
		case viewConfirmKill:
			return m.updateConfirmKill(msg)
		case viewHelp:
			if msg.String() == "q" || msg.String() == "?" || msg.String() == "esc" {
				m.mode = viewTable
			}
			return m, nil
		case viewDetail:
			return m.updateDetail(msg)
		case viewTable:
			return m.updateTable(msg)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m Model) updateTable(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Up):
		if m.cursor > 0 {
			m.cursor--
		}
	case key.Matches(msg, m.keys.Down):
		if m.cursor < len(m.filteredRows)-1 {
			m.cursor++
		}
	case key.Matches(msg, m.keys.Top):
		m.cursor = 0
	case key.Matches(msg, m.keys.Bottom):
		if len(m.filteredRows) > 0 {
			m.cursor = len(m.filteredRows) - 1
		}
	case key.Matches(msg, m.keys.HalfPageUp):
		visible := m.tableVisibleRows()
		m.cursor -= visible / 2
		if m.cursor < 0 {
			m.cursor = 0
		}
	case key.Matches(msg, m.keys.HalfPageDown):
		visible := m.tableVisibleRows()
		m.cursor += visible / 2
		if m.cursor >= len(m.filteredRows) {
			m.cursor = len(m.filteredRows) - 1
		}
		if m.cursor < 0 {
			m.cursor = 0
		}
	case key.Matches(msg, m.keys.Enter):
		if len(m.filteredRows) > 0 {
			m.mode = viewDetail
			m.detailCursor = 0
		}
	case key.Matches(msg, m.keys.Kill):
		if len(m.filteredRows) > 0 && m.cursor < len(m.filteredRows) {
			entry := m.filteredRows[m.cursor]
			m.killTarget = &entry
			m.forceKill = false
			m.mode = viewConfirmKill
		}
	case key.Matches(msg, m.keys.ForceKill):
		if len(m.filteredRows) > 0 && m.cursor < len(m.filteredRows) {
			entry := m.filteredRows[m.cursor]
			m.killTarget = &entry
			m.forceKill = true
			m.mode = viewConfirmKill
		}
	case key.Matches(msg, m.keys.Search):
		m.mode = viewSearch
		m.searchInput.Focus()
		return m, textinput.Blink
	case key.Matches(msg, m.keys.ClearSearch):
		m.searchQuery = ""
		m.searchInput.SetValue("")
		m.applyFilter()
		m.cursor = 0
	case key.Matches(msg, m.keys.Refresh):
		return m, doScan
	case key.Matches(msg, m.keys.ToggleAutoRefresh):
		m.autoRefresh = !m.autoRefresh
	case key.Matches(msg, m.keys.ToggleListening):
		m.showListening = !m.showListening
		m.applyFilter()
		m.cursor = 0
	case key.Matches(msg, m.keys.ToggleSystem):
		m.showSystem = !m.showSystem
		m.applyFilter()
		m.cursor = 0
	case key.Matches(msg, m.keys.Help):
		m.mode = viewHelp
	}

	return m, nil
}

func (m Model) updateDetail(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Kill):
		if len(m.filteredRows) > 0 && m.cursor < len(m.filteredRows) {
			entry := m.filteredRows[m.cursor]
			m.killTarget = &entry
			m.forceKill = false
			m.mode = viewConfirmKill
		}
	case key.Matches(msg, m.keys.ForceKill):
		if len(m.filteredRows) > 0 && m.cursor < len(m.filteredRows) {
			entry := m.filteredRows[m.cursor]
			m.killTarget = &entry
			m.forceKill = true
			m.mode = viewConfirmKill
		}
	case msg.String() == "esc" || msg.String() == "q" || msg.String() == "enter":
		m.mode = viewTable
	}
	return m, nil
}

func (m Model) updateSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.searchQuery = m.searchInput.Value()
		m.applyFilter()
		m.cursor = 0
		m.mode = viewTable
		m.searchInput.Blur()
		return m, nil
	case "esc":
		m.mode = viewTable
		m.searchInput.Blur()
		m.searchInput.SetValue(m.searchQuery)
		return m, nil
	}

	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)
	return m, cmd
}

func (m Model) updateConfirmKill(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		if m.killTarget != nil {
			target := *m.killTarget
			force := m.forceKill
			return m, func() tea.Msg {
				var err error
				if force {
					err = process.ForceKill(target.PID)
				} else {
					err = process.Kill(target.PID)
				}
				return killResultMsg{pid: target.PID, command: target.Command, err: err}
			}
		}
	case "n", "N", "esc", "q":
		m.mode = viewTable
		m.killTarget = nil
	}
	return m, nil
}

func (m Model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	// Only handle clicks in table view
	if m.mode != viewTable {
		return m, nil
	}

	// Handle scroll wheel
	if msg.Button == tea.MouseButtonWheelUp {
		if m.cursor > 0 {
			m.cursor--
		}
		return m, nil
	}
	if msg.Button == tea.MouseButtonWheelDown {
		if m.cursor < len(m.filteredRows)-1 {
			m.cursor++
		}
		return m, nil
	}

	// Only handle left click release (= completed click)
	if msg.Button != tea.MouseButtonLeft || msg.Action != tea.MouseActionRelease {
		return m, nil
	}

	if len(m.filteredRows) == 0 {
		return m, nil
	}

	// Layout: title(1) + header(1) + separator(1) = data starts at Y=3
	const dataStartY = 3

	clickY := msg.Y
	if clickY < dataStartY {
		return m, nil
	}

	// Compute scroll offset (same logic as renderTable)
	visibleRows := m.tableVisibleRows()
	if visibleRows < 1 {
		visibleRows = 1
	}
	scrollStart := 0
	if m.cursor >= visibleRows {
		scrollStart = m.cursor - visibleRows + 1
	}

	rowIdx := scrollStart + (clickY - dataStartY)
	if rowIdx < 0 || rowIdx >= len(m.filteredRows) {
		return m, nil
	}

	// If clicking the already-selected row, open detail view
	if rowIdx == m.cursor {
		m.mode = viewDetail
		m.detailCursor = 0
		return m, nil
	}

	// Otherwise, select the row
	m.cursor = rowIdx
	return m, nil
}

func (m *Model) applyFilter() {
	if m.scanResult == nil {
		m.filteredRows = nil
		return
	}

	var rows []scanner.PortEntry
	query := strings.ToLower(m.searchQuery)

	for _, e := range m.scanResult.Entries {
		// Hide system processes unless toggled on
		if !m.showSystem && scanner.IsSystemProcess(e.Command) {
			continue
		}

		// Filter to LISTEN only if toggled
		if m.showListening && e.State != "LISTEN" {
			continue
		}

		// Apply search filter
		if query != "" {
			portStr := fmt.Sprintf("%d", e.Port)
			haystack := strings.ToLower(fmt.Sprintf("%s %s %s %s %d %s",
				e.Command, e.User, e.Protocol, e.State, e.Port, e.LocalAddr))
			if !strings.Contains(haystack, query) && !strings.Contains(portStr, query) {
				continue
			}
		}

		rows = append(rows, e)
	}

	m.filteredRows = rows
}

func (m Model) tableVisibleRows() int {
	// title(1) + header(1) + separator(1) + status bar(1) = 4 lines of chrome
	return m.height - 4
}

// Commands
func doScan() tea.Msg {
	result, err := scanner.Scan()
	if err != nil {
		return scanErrMsg(err)
	}
	return scanResultMsg(result)
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}
