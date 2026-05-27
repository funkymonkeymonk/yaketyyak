package tui

import "github.com/charmbracelet/bubbles/key"

type KeyMap struct {
	Up            key.Binding
	Down          key.Binding
	Home          key.Binding
	End           key.Binding
	ToggleHelp    key.Binding
	TogglePreview key.Binding
	Refresh       key.Binding
	Quit          key.Binding
	Select        key.Binding
	Edit          key.Binding
	Confirm       key.Binding
	Cancel        key.Binding
	NextTab       key.Binding
	PrevTab       key.Binding
	Tab1          key.Binding
	Tab2          key.Binding
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up:            key.NewBinding(key.WithKeys("k", "up", "ctrl+p"), key.WithHelp("↑/k", "up")),
		Down:          key.NewBinding(key.WithKeys("j", "down", "ctrl+n"), key.WithHelp("↓/j", "down")),
		Home:          key.NewBinding(key.WithKeys("ctrl+a"), key.WithHelp("ctrl+a", "top")),
		End:           key.NewBinding(key.WithKeys("ctrl+e"), key.WithHelp("ctrl+e", "bottom")),
		ToggleHelp:    key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		TogglePreview: key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "preview")),
		Refresh:       key.NewBinding(key.WithKeys("ctrl+r"), key.WithHelp("ctrl+r", "refresh")),
		Quit:          key.NewBinding(key.WithKeys("q", "esc", "ctrl+c"), key.WithHelp("q", "quit")),
		Select:        key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "context")),
		Edit:          key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit")),
		Confirm:       key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "confirm")),
		Cancel:        key.NewBinding(key.WithKeys("n", "esc"), key.WithHelp("n/esc", "cancel")),
		NextTab:       key.NewBinding(key.WithKeys("tab", "ctrl+right"), key.WithHelp("tab", "next tab")),
		PrevTab:       key.NewBinding(key.WithKeys("shift+tab", "ctrl+left"), key.WithHelp("shift+tab", "prev tab")),
		Tab1:          key.NewBinding(key.WithKeys("1"), key.WithHelp("1", "yaks tab")),
		Tab2:          key.NewBinding(key.WithKeys("2"), key.WithHelp("2", "prs tab")),
	}
}

func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Quit, k.ToggleHelp}
}

func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Home, k.End},
		{k.Select, k.Edit},
		{k.NextTab, k.PrevTab, k.Tab1, k.Tab2},
		{k.Refresh, k.ToggleHelp, k.Quit},
	}
}
