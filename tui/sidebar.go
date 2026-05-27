package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
)

type sidebarModel struct {
	viewport viewport.Model
	width    int
	height   int
	styles   *Styles
	zoneID   string
}

func newSidebar() sidebarModel {
	vp := viewport.New(40, 20)
	vp.MouseWheelEnabled = true
	vp.MouseWheelDelta = 3

	return sidebarModel{
		viewport: vp,
		zoneID:   zone.NewPrefix(),
	}
}

func (s *sidebarModel) SetSize(width, height int) {
	s.width = width
	s.height = height
	s.viewport.Width = width
	s.viewport.Height = height
}

func (s *sidebarModel) syncStyles(st *Styles) {
	s.styles = st
}

func (s *sidebarModel) ShowRepo(repo *Repo, width int) {
	var lines []string
	sep := dimStyle.Render(strings.Repeat("─", 40))

	lines = append(lines, padWidth(headerStyle.Render(repo.Name), width))
	lines = append(lines, padWidth(sep, width))

	lines = append(lines, padWidth(fmt.Sprintf("Path:   %s", dimStyle.Render(repo.Root)), width))
	lines = append(lines, padWidth(fmt.Sprintf("Remote: %s", dimStyle.Render(repo.Remote)), width))
	lines = append(lines, padWidth(fmt.Sprintf("Yaks:   %d", len(repo.Yaks)), width))
	lines = append(lines, padWidth(sep, width))

	lines = append(lines, padWidth(pinkBoldStyle.Render("Actions:"), width))
	lines = append(lines, padWidth(fmt.Sprintf("  [%s] Refresh", dimStyle.Render("Ctrl+R")), width))

	s.viewport.GotoTop()
	s.viewport.SetContent(strings.Join(lines, "\n"))
}

func (s *sidebarModel) ShowYak(yak *YakLine, width int) {
	var lines []string
	sep := dimStyle.Render(strings.Repeat("─", 40))

	lines = append(lines, padWidth(headerStyle.Render(yak.Name), width))
	lines = append(lines, padWidth(sep, width))

	lines = append(lines, padWidth(fmt.Sprintf("ID:    %s", dimStyle.Render(yak.ID)), width))
	stateLine := fmt.Sprintf("State: %s %s", colorFor(yak.State).Render(symbolFor(yak.State)), yak.State)
	lines = append(lines, padWidth(stateLine, width))

	if yak.PRURL != "" {
		lines = append(lines, padWidth(fmt.Sprintf("PR:    %s", cyanStyle.Render(yak.PRURL)), width))
	}

	lines = append(lines, padWidth(sep, width))

	if yak.Context != "" {
		lines = append(lines, padWidth(pinkBoldStyle.Render("Context:"), width))
		ctxWrap := width - 2
		if ctxWrap < 10 {
			ctxWrap = 10
		}
		for _, l := range wrapText(yak.Context, ctxWrap) {
			lines = append(lines, padWidth("  "+dimStyle.Render(l), width))
		}
		lines = append(lines, padWidth(sep, width))
	}

	lines = append(lines, padWidth(pinkBoldStyle.Render("Actions:"), width))
	lines = append(lines, padWidth(fmt.Sprintf("  [%s] Edit context   [%s] Full context",
		dimStyle.Render("e"), dimStyle.Render("↲")), width))

	s.viewport.GotoTop()
	s.viewport.SetContent(strings.Join(lines, "\n"))
}

func (s *sidebarModel) ShowPR(pr *PRLine, width int) {
	s.viewport.GotoTop()
	s.viewport.SetContent(showPRDetail(pr, width))
}

func showPRDetail(pr *PRLine, width int) string {
	var lines []string
	sep := dimStyle.Render(strings.Repeat("─", 40))

	lines = append(lines, padWidth(headerStyle.Render(pr.YakName), width))
	lines = append(lines, padWidth(sep, width))

	lines = append(lines, padWidth(fmt.Sprintf("Repo:  %s", dimStyle.Render(pr.Repo)), width))
	lines = append(lines, padWidth(fmt.Sprintf("PR:    #%d", pr.PRNum), width))
	lines = append(lines, padWidth(fmt.Sprintf("URL:   %s", pr.PRURL), width))

	stateDisplay := pr.State
	stateStyle := dimStyle
	switch pr.State {
	case "OPEN", "open":
		stateStyle = greenStyle
	case "MERGED", "merged":
		stateStyle = doneStyle
	case "CLOSED", "closed":
		stateStyle = wipStyle
	}
	lines = append(lines, padWidth(fmt.Sprintf("State: %s", stateStyle.Render(stateDisplay)), width))

	if pr.Title != "" {
		lines = append(lines, padWidth(fmt.Sprintf("Title: %s", pr.Title), width))
	}

	if pr.CIState != "" {
		ciStyle := dimStyle
		switch pr.CIState {
		case "success":
			ciStyle = greenStyle
		case "failure":
			ciStyle = wipStyle
		}
		lines = append(lines, padWidth(fmt.Sprintf("CI:    %s", ciStyle.Render(pr.CIState)), width))
	}

	lines = append(lines, padWidth(sep, width))
	lines = append(lines, padWidth(pinkBoldStyle.Render("Actions:"), width))
	lines = append(lines, padWidth(fmt.Sprintf("  [%s] Open PR in browser", dimStyle.Render("o")), width))

	return strings.Join(lines, "\n")
}

func (s *sidebarModel) ShowPRActivityEmpty(width int) {
	var lines []string
	lines = append(lines, padWidth(headerStyle.Render("PR Activity"), width))
	lines = append(lines, "")
	lines = append(lines, padWidth(dimStyle.Render("No open PRs found."), width))
	lines = append(lines, "")
	lines = append(lines, padWidth(dimStyle.Render("PRs are created when yaks"), width))
	lines = append(lines, padWidth(dimStyle.Render("are shaved via the workflow."), width))
	s.viewport.GotoTop()
	s.viewport.SetContent(strings.Join(lines, "\n"))
}

func (s *sidebarModel) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.MouseMsg:
		if msg.Action == tea.MouseActionPress || msg.Action == tea.MouseActionRelease {
			return nil
		}
		if z := zone.Get(s.zoneID); z != nil && z.InBounds(msg) {
			vp, cmd := s.viewport.Update(msg)
			s.viewport = vp
			return cmd
		}
		return nil
	}

	vp, cmd := s.viewport.Update(msg)
	s.viewport = vp
	return cmd
}

func (s *sidebarModel) View() string {
	style := lipgloss.NewStyle().Width(s.width)
	v := s.viewport.View()
	return zone.Mark(s.zoneID, style.Render(v))
}
