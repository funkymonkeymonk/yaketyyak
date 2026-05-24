package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type TabID int

const (
	TabYaks TabID = iota
	TabPRs
)

var tabNames = map[TabID]string{
	TabYaks: "Yaks",
	TabPRs:  "PRs",
}

var tabKeys = map[TabID]string{
	TabYaks: "1",
	TabPRs:  "2",
}

type tabsModel struct {
	active TabID
	tabs   []TabID
	styles *Styles
}

func newTabsModel() tabsModel {
	return tabsModel{
		active: TabYaks,
		tabs:   []TabID{TabYaks, TabPRs},
	}
}

func (t *tabsModel) Active() TabID {
	return t.active
}

func (t *tabsModel) SetActive(id TabID) {
	for _, tab := range t.tabs {
		if tab == id {
			t.active = id
			return
		}
	}
}

func (t *tabsModel) Next() {
	for i, tab := range t.tabs {
		if tab == t.active {
			next := (i + 1) % len(t.tabs)
			t.active = t.tabs[next]
			return
		}
	}
}

func (t *tabsModel) Prev() {
	for i, tab := range t.tabs {
		if tab == t.active {
			prev := (i - 1 + len(t.tabs)) % len(t.tabs)
			t.active = t.tabs[prev]
			return
		}
	}
}

func (t *tabsModel) syncStyles(s *Styles) {
	t.styles = s
}

func (t *tabsModel) View() string {
	activeTabStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FF69B4")).
		Bold(true).
		Underline(true)

	inactiveTabStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666666"))

	sepStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#444444"))

	var parts []string
	for i, id := range t.tabs {
		name := tabNames[id]
		if id == t.active {
			parts = append(parts, activeTabStyle.Render(name))
		} else {
			parts = append(parts, inactiveTabStyle.Render(name+" ["+tabKeys[id]+"]"))
		}
		if i < len(t.tabs)-1 {
			parts = append(parts, sepStyle.Render(" │ "))
		}
	}

	return strings.Join(parts, "")
}
