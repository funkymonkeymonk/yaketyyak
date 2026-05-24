package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type footerModel struct {
	width  int
	styles *Styles
}

func newFooter() footerModel {
	return footerModel{}
}

func (f *footerModel) SetSize(width int) {
	f.width = width
}

func (f *footerModel) syncStyles(s *Styles) {
	f.styles = s
}

func (f *footerModel) Update(msg tea.Msg) tea.Cmd {
	return nil
}

func (f *footerModel) View(statusMsg string, showHelp bool, repoCount, yakCount int, phase string) string {
	if showHelp {
		return lipgloss.NewStyle().Width(f.width).Background(lipgloss.Color("#3A3A3A")).
			Render("  ?:close  j/k:nav  s:start  v:shave  p:pause  r:resume  g:g2g  c:ci  e:edit  ENTER:context  q:quit")
	}

	var parts []string
	parts = append(parts, fmt.Sprintf("%d repos %d yaks", repoCount, yakCount))

	if phase != "" {
		parts = append(parts, fmt.Sprintf("Phase: %s", cyanStyle.Render(phase)))
	}

	if statusMsg != "" {
		parts = append(parts, statusMsg)
	}

	left := strings.Join(parts, " | ")
	right := dimStyle.Render("j/k |:nav  ?:help  q:quit")

	leftW := lipgloss.Width(left)
	rightW := lipgloss.Width(right)
	pad := f.width - leftW - rightW
	if pad < 1 {
		pad = 1
	}

	return lipgloss.NewStyle().Width(f.width).Background(lipgloss.Color("#3A3A3A")).
		Render(left + strings.Repeat(" ", pad) + right)
}
