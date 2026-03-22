package ui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Up                key.Binding
	Down              key.Binding
	Top               key.Binding
	Bottom            key.Binding
	HalfPageUp        key.Binding
	HalfPageDown      key.Binding
	Enter             key.Binding
	Kill              key.Binding
	ForceKill         key.Binding
	Search            key.Binding
	ClearSearch       key.Binding
	Refresh           key.Binding
	ToggleAutoRefresh key.Binding
	ToggleListening   key.Binding
	ToggleSystem      key.Binding
	Help              key.Binding
	Quit              key.Binding
}

func defaultKeyMap() keyMap {
	return keyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Top: key.NewBinding(
			key.WithKeys("g"),
			key.WithHelp("g", "top"),
		),
		Bottom: key.NewBinding(
			key.WithKeys("G"),
			key.WithHelp("G", "bottom"),
		),
		HalfPageUp: key.NewBinding(
			key.WithKeys("ctrl+u"),
			key.WithHelp("ctrl+u", "half page up"),
		),
		HalfPageDown: key.NewBinding(
			key.WithKeys("ctrl+d"),
			key.WithHelp("ctrl+d", "half page down"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "details"),
		),
		Kill: key.NewBinding(
			key.WithKeys("x"),
			key.WithHelp("x", "kill (SIGTERM)"),
		),
		ForceKill: key.NewBinding(
			key.WithKeys("X"),
			key.WithHelp("X", "force kill (SIGKILL)"),
		),
		Search: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "search"),
		),
		ClearSearch: key.NewBinding(
			key.WithKeys("ctrl+l"),
			key.WithHelp("ctrl+l", "clear search"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
		),
		ToggleAutoRefresh: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "toggle auto-refresh"),
		),
		ToggleListening: key.NewBinding(
			key.WithKeys("l"),
			key.WithHelp("l", "toggle listen-only"),
		),
		ToggleSystem: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "toggle system processes"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "esc", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
	}
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		k.Up, k.Down, k.Enter, k.Kill, k.Search, k.Refresh, k.ToggleListening, k.ToggleSystem, k.Help, k.Quit,
	}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Top, k.Bottom, k.HalfPageUp, k.HalfPageDown},
		{k.Enter, k.Kill, k.ForceKill},
		{k.Search, k.ClearSearch, k.Refresh, k.ToggleAutoRefresh, k.ToggleListening, k.ToggleSystem},
		{k.Help, k.Quit},
	}
}
