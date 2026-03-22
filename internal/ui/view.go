package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/port-inspector/port-inspector/internal/process"
)

// ── Styles ──────────────────────────────────────────────────────────────────

var (
	// Colors
	primaryColor   = lipgloss.Color("#7D56F4")
	secondaryColor = lipgloss.Color("#6C71C4")
	accentColor    = lipgloss.Color("#F25D94")
	successColor   = lipgloss.Color("#A6E3A1")
	warningColor   = lipgloss.Color("#FAB387")
	dangerColor    = lipgloss.Color("#F38BA8")
	mutedColor     = lipgloss.Color("#6C7086")
	textColor      = lipgloss.Color("#CDD6F4")
	bgColor        = lipgloss.Color("#1E1E2E")
	surfaceColor   = lipgloss.Color("#313244")
	overlayColor   = lipgloss.Color("#45475A")

	// Title bar
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#CDD6F4")).
			Background(primaryColor).
			Padding(0, 1)

	titleBarStyle = lipgloss.NewStyle().
			Background(surfaceColor).
			Width(80) // will be set dynamically

	// Table
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			BorderBottom(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(overlayColor)

	selectedRowStyle = lipgloss.NewStyle().
				Background(surfaceColor).
				Foreground(textColor).
				Bold(true)

	normalRowStyle = lipgloss.NewStyle().
			Foreground(textColor)

	portStyle = lipgloss.NewStyle().
			Foreground(accentColor).
			Bold(true)

	commandStyle = lipgloss.NewStyle().
			Foreground(successColor)

	pidStyle = lipgloss.NewStyle().
			Foreground(secondaryColor)

	stateListenStyle = lipgloss.NewStyle().
				Foreground(successColor).
				Bold(true)

	stateEstablishedStyle = lipgloss.NewStyle().
				Foreground(warningColor)

	stateOtherStyle = lipgloss.NewStyle().
			Foreground(mutedColor)

	// Status bar
	statusBarStyle = lipgloss.NewStyle().
			Background(surfaceColor).
			Foreground(textColor).
			Padding(0, 1)

	statusTagStyle = lipgloss.NewStyle().
			Background(primaryColor).
			Foreground(lipgloss.Color("#CDD6F4")).
			Padding(0, 1).
			Bold(true)

	statusErrorStyle = lipgloss.NewStyle().
				Foreground(dangerColor).
				Bold(true)

	statusSuccessStyle = lipgloss.NewStyle().
				Foreground(successColor).
				Bold(true)

	// Detail view
	detailBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(primaryColor).
				Padding(1, 2)

	detailLabelStyle = lipgloss.NewStyle().
				Foreground(primaryColor).
				Bold(true).
				Width(14)

	detailValueStyle = lipgloss.NewStyle().
				Foreground(textColor)

	// Confirm dialog
	dialogStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(dangerColor).
			Padding(1, 3).
			Width(60)

	dialogTitleStyle = lipgloss.NewStyle().
				Foreground(dangerColor).
				Bold(true)

	// Help view
	helpTitleStyle = lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true).
			MarginBottom(1)

	helpKeyStyle = lipgloss.NewStyle().
			Foreground(accentColor).
			Bold(true).
			Width(16)

	helpDescStyle = lipgloss.NewStyle().
			Foreground(textColor)

	// Search bar
	searchBarStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor).
			Padding(0, 1)

	// Indicator
	cursorStyle = lipgloss.NewStyle().
			Foreground(accentColor).
			Bold(true)
)

// ── View ────────────────────────────────────────────────────────────────────

func (m Model) View() string {
	if !m.ready {
		return "\n  Loading port-inspector..."
	}

	var sections []string

	// Title bar
	sections = append(sections, m.renderTitleBar())

	// Main content
	switch m.mode {
	case viewHelp:
		sections = append(sections, m.renderHelp())
	case viewConfirmKill:
		sections = append(sections, m.renderTable())
		sections = append(sections, m.renderConfirmKill())
	case viewDetail:
		sections = append(sections, m.renderDetail())
	case viewSearch:
		sections = append(sections, m.renderTable())
		sections = append(sections, m.renderSearchBar())
	default:
		sections = append(sections, m.renderTable())
	}

	// Status bar
	sections = append(sections, m.renderStatusBar())

	return strings.Join(sections, "\n")
}

func (m Model) renderTitleBar() string {
	title := titleStyle.Render(" PORT INSPECTOR ")

	var indicators []string

	// Connection count
	total := len(m.filteredRows)
	if m.scanResult != nil {
		totalAll := len(m.scanResult.Entries)
		if total != totalAll {
			indicators = append(indicators, fmt.Sprintf("%d/%d connections", total, totalAll))
		} else {
			indicators = append(indicators, fmt.Sprintf("%d connections", total))
		}
	}

	// Auto-refresh indicator
	if m.autoRefresh {
		indicators = append(indicators, lipgloss.NewStyle().Foreground(successColor).Render("● auto"))
	} else {
		indicators = append(indicators, lipgloss.NewStyle().Foreground(mutedColor).Render("○ paused"))
	}

	// Listen-only filter
	if m.showListening {
		indicators = append(indicators, lipgloss.NewStyle().Foreground(warningColor).Render("LISTEN"))
	}

	// System processes filter
	if m.showSystem {
		indicators = append(indicators, lipgloss.NewStyle().Foreground(mutedColor).Render("+ system"))
	}

	// Search query
	if m.searchQuery != "" {
		indicators = append(indicators,
			lipgloss.NewStyle().Foreground(accentColor).Render("search: "+m.searchQuery))
	}

	right := strings.Join(indicators, "  ")
	rightStyled := lipgloss.NewStyle().Foreground(textColor).Render(right)

	gap := m.width - lipgloss.Width(title) - lipgloss.Width(right) - 2
	if gap < 0 {
		gap = 0
	}

	bar := lipgloss.NewStyle().
		Background(surfaceColor).
		Width(m.width).
		Render(title + strings.Repeat(" ", gap) + rightStyled)

	return bar
}

func (m Model) renderTable() string {
	if m.err != nil {
		return fmt.Sprintf("\n  %s\n", statusErrorStyle.Render(fmt.Sprintf("Error: %v", m.err)))
	}

	if m.scanResult == nil {
		return "\n  Scanning ports..."
	}

	if len(m.filteredRows) == 0 {
		msg := "  No connections found."
		if m.searchQuery != "" {
			msg = fmt.Sprintf("  No results for \"%s\".", m.searchQuery)
		}
		return "\n" + lipgloss.NewStyle().Foreground(mutedColor).Render(msg) + "\n"
	}

	var b strings.Builder

	// Column widths
	colPort := 7
	colProto := 5
	colState := 14
	colPID := 8
	colUser := 12
	colCommand := 20
	colAddr := m.width - colPort - colProto - colState - colPID - colUser - colCommand - 12 // padding + cursor
	if colAddr < 10 {
		colAddr = 10
	}

	// Header
	header := fmt.Sprintf("  %s  %s  %s  %s  %s  %s  %s",
		headerStyle.Width(colPort).Render("PORT"),
		headerStyle.Width(colProto).Render("PROTO"),
		headerStyle.Width(colState).Render("STATE"),
		headerStyle.Width(colPID).Render("PID"),
		headerStyle.Width(colUser).Render("USER"),
		headerStyle.Width(colCommand).Render("COMMAND"),
		headerStyle.Width(colAddr).Render("ADDRESS"),
	)
	b.WriteString(header + "\n")

	// Visible rows
	visibleRows := m.tableVisibleRows()
	if visibleRows < 1 {
		visibleRows = 1
	}

	// Calculate scroll offset
	start := 0
	if m.cursor >= visibleRows {
		start = m.cursor - visibleRows + 1
	}
	end := start + visibleRows
	if end > len(m.filteredRows) {
		end = len(m.filteredRows)
	}

	for i := start; i < end; i++ {
		e := m.filteredRows[i]
		selected := i == m.cursor

		// Format state with color
		var stateStr string
		switch e.State {
		case "LISTEN":
			stateStr = stateListenStyle.Width(colState).Render(e.State)
		case "ESTABLISHED":
			stateStr = stateEstablishedStyle.Width(colState).Render(e.State)
		default:
			state := e.State
			if state == "" {
				state = "-"
			}
			stateStr = stateOtherStyle.Width(colState).Render(state)
		}

		cursor := "  "
		if selected {
			cursor = cursorStyle.Render("> ")
		}

		row := fmt.Sprintf("%s%s  %s  %s  %s  %s  %s  %s",
			cursor,
			portStyle.Width(colPort).Render(fmt.Sprintf("%d", e.Port)),
			lipgloss.NewStyle().Width(colProto).Foreground(mutedColor).Render(e.Protocol),
			stateStr,
			pidStyle.Width(colPID).Render(fmt.Sprintf("%d", e.PID)),
			lipgloss.NewStyle().Width(colUser).Foreground(textColor).Render(truncate(e.User, colUser)),
			commandStyle.Width(colCommand).Render(truncate(e.Command, colCommand)),
			lipgloss.NewStyle().Width(colAddr).Foreground(mutedColor).Render(truncate(e.LocalAddr, colAddr)),
		)

		if selected {
			row = selectedRowStyle.Width(m.width).Render(row)
		}

		b.WriteString(row + "\n")
	}

	// Pad remaining lines to prevent visual artifacts
	remaining := visibleRows - (end - start)
	for i := 0; i < remaining; i++ {
		b.WriteString("\n")
	}

	return b.String()
}

func (m Model) renderDetail() string {
	if m.cursor >= len(m.filteredRows) {
		return ""
	}

	e := m.filteredRows[m.cursor]

	// Get extended process info
	path := process.GetProcessPath(e.PID)
	cpu := process.GetProcessCPU(e.PID)
	mem := process.GetProcessMem(e.PID)
	startTime := process.GetProcessStartTime(e.PID)

	var details strings.Builder
	details.WriteString(lipgloss.NewStyle().Foreground(primaryColor).Bold(true).Render("Process Details") + "\n\n")

	fields := []struct {
		label string
		value string
	}{
		{"Command", e.Command},
		{"PID", fmt.Sprintf("%d", e.PID)},
		{"User", e.User},
		{"Port", fmt.Sprintf("%d", e.Port)},
		{"Protocol", e.Protocol},
		{"State", e.State},
		{"Address", e.LocalAddr},
		{"FD", e.FD},
		{"Type", e.Type},
		{"Path", path},
		{"CPU", cpu + "%"},
		{"Memory", mem + "%"},
		{"Started", startTime},
		{"Running", fmt.Sprintf("%v", process.IsRunning(e.PID))},
	}

	for _, f := range fields {
		details.WriteString(fmt.Sprintf("%s %s\n",
			detailLabelStyle.Render(f.label+":"),
			detailValueStyle.Render(f.value),
		))
	}

	details.WriteString("\n" + lipgloss.NewStyle().Foreground(mutedColor).Render(
		"Press x to kill  |  X to force kill  |  esc to go back"))

	boxWidth := m.width - 8
	if boxWidth < 40 {
		boxWidth = 40
	}

	box := detailBorderStyle.Width(boxWidth).Render(details.String())
	return lipgloss.Place(m.width, m.height-6, lipgloss.Center, lipgloss.Center, box)
}

func (m Model) renderConfirmKill() string {
	if m.killTarget == nil {
		return ""
	}

	action := "Kill"
	signal := "SIGTERM"
	if m.forceKill {
		action = "Force Kill"
		signal = "SIGKILL"
	}

	content := fmt.Sprintf(
		"%s\n\n%s %s\n%s %s\n%s %s\n\n%s",
		dialogTitleStyle.Render(fmt.Sprintf("  %s Process?", action)),
		detailLabelStyle.Render("Command:"),
		commandStyle.Render(m.killTarget.Command),
		detailLabelStyle.Render("PID:"),
		pidStyle.Render(fmt.Sprintf("%d", m.killTarget.PID)),
		detailLabelStyle.Render("Signal:"),
		lipgloss.NewStyle().Foreground(dangerColor).Bold(true).Render(signal),
		lipgloss.NewStyle().Foreground(mutedColor).Render("Press y to confirm, n/esc to cancel"),
	)

	dialog := dialogStyle.Render(content)
	return lipgloss.Place(m.width, 10, lipgloss.Center, lipgloss.Center, dialog)
}

func (m Model) renderSearchBar() string {
	bar := searchBarStyle.Width(m.width - 4).Render(
		fmt.Sprintf(" / %s", m.searchInput.View()),
	)
	return bar
}

func (m Model) renderStatusBar() string {
	var left string
	if m.statusMessage != "" {
		if strings.HasPrefix(m.statusMessage, "✓") {
			left = statusSuccessStyle.Render(m.statusMessage)
		} else {
			left = statusErrorStyle.Render(m.statusMessage)
		}
	} else {
		left = lipgloss.NewStyle().Foreground(mutedColor).Render(
			fmt.Sprintf("Last scan: %s", m.lastScan.Format(time.Kitchen)))
	}

	helpText := lipgloss.NewStyle().Foreground(mutedColor).Render(
		"j/k:navigate  enter:details  x:kill  /:search  l:listen  s:system  ?:help  q:quit")

	gap := m.width - lipgloss.Width(left) - lipgloss.Width(helpText) - 2
	if gap < 0 {
		gap = 0
		// Truncate help if needed
		helpText = lipgloss.NewStyle().Foreground(mutedColor).Render("?:help  q:quit")
		gap = m.width - lipgloss.Width(left) - lipgloss.Width(helpText) - 2
		if gap < 0 {
			gap = 0
		}
	}

	bar := statusBarStyle.Width(m.width).Render(left + strings.Repeat(" ", gap) + helpText)
	return bar
}

func (m Model) renderHelp() string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(helpTitleStyle.Render("  Keyboard Shortcuts") + "\n\n")

	sections := []struct {
		title string
		keys  []struct {
			key  string
			desc string
		}
	}{
		{
			"Navigation",
			[]struct {
				key  string
				desc string
			}{
				{"j / ↓", "Move down"},
				{"k / ↑", "Move up"},
				{"g", "Go to top"},
				{"G", "Go to bottom"},
				{"Ctrl+d", "Half page down"},
				{"Ctrl+u", "Half page up"},
				{"Enter", "View process details"},
			},
		},
		{
			"Actions",
			[]struct {
				key  string
				desc string
			}{
				{"x", "Kill process (SIGTERM)"},
				{"X", "Force kill process (SIGKILL)"},
				{"r", "Manual refresh"},
				{"a", "Toggle auto-refresh"},
				{"l", "Toggle listen-only filter"},
				{"s", "Toggle system processes"},
			},
		},
		{
			"Search",
			[]struct {
				key  string
				desc string
			}{
				{"/", "Open search"},
				{"Enter", "Apply search"},
				{"Esc", "Cancel search"},
				{"Ctrl+l", "Clear search"},
			},
		},
		{
			"General",
			[]struct {
				key  string
				desc string
			}{
				{"?", "Toggle help"},
				{"q / Esc", "Quit / Go back"},
				{"Ctrl+c", "Quit"},
			},
		},
	}

	for _, section := range sections {
		b.WriteString("  " + lipgloss.NewStyle().Foreground(primaryColor).Bold(true).Render(section.title) + "\n")
		for _, k := range section.keys {
			b.WriteString(fmt.Sprintf("    %s %s\n",
				helpKeyStyle.Render(k.key),
				helpDescStyle.Render(k.desc),
			))
		}
		b.WriteString("\n")
	}

	b.WriteString("  " + lipgloss.NewStyle().Foreground(mutedColor).Render("Press ? or Esc to close") + "\n")

	return b.String()
}

// ── Helpers ─────────────────────────────────────────────────────────────────

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max < 4 {
		return s[:max]
	}
	return s[:max-1] + "…"
}
