package tui

import "github.com/charmbracelet/lipgloss"

func defaultStyles() Styles {
	return Styles{
		Footer: FooterStyles{
			BarStyle: lipgloss.NewStyle().
				Width(80).
				Background(lipgloss.Color("#3A3A3A")),
			LeftStyle:   lipgloss.NewStyle(),
			RightStyle:  lipgloss.NewStyle().Foreground(lipgloss.Color("#666666")),
			KeyStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("#666666")),
			ActionStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("#FF69B4")).Bold(true),
		},
		Sidebar: SectionStyles{
			ContainerStyle: lipgloss.NewStyle(),
			KeyStyle:       lipgloss.NewStyle().Foreground(lipgloss.Color("#666666")),
			SelectStyle:    lipgloss.NewStyle().Background(lipgloss.Color("#3A3A3A")),
		},
		Common: CommonStyles{
			MainTextStyle: lipgloss.NewStyle(),
			FooterStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("#666666")),
			HelpStyle:     lipgloss.NewStyle().Background(lipgloss.Color("#3A3A3A")),
			OverlayStyle: lipgloss.NewStyle().
				Background(lipgloss.Color("#1E1E1E")).
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#FF69B4")).
				Padding(1, 2),
		},
	}
}
