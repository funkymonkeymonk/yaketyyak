package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
)

type PRLine struct {
	YakName string
	Repo    string
	PRURL   string
	PRNum   int
	Title   string
	State   string
	CIState string
}

type prActivityModel struct {
	viewport viewport.Model
	items    []PRLine
	cursor   int
	width    int
	height   int
	styles   *Styles
	zoneID   string
}

func newPRActivity() prActivityModel {
	vp := viewport.New(40, 20)
	vp.MouseWheelEnabled = true
	vp.MouseWheelDelta = 3

	return prActivityModel{
		viewport: vp,
		zoneID:   zone.NewPrefix(),
	}
}

func (p *prActivityModel) SetItems(prs []PRLine) {
	p.items = prs
	if p.cursor >= len(p.items) && len(p.items) > 0 {
		p.cursor = len(p.items) - 1
	}
}

func (p *prActivityModel) Cursor() int { return p.cursor }
func (p *prActivityModel) Count() int  { return len(p.items) }

func (p *prActivityModel) SelectedPR() *PRLine {
	if p.cursor < 0 || p.cursor >= len(p.items) {
		return nil
	}
	return &p.items[p.cursor]
}

func (p *prActivityModel) MoveCursor(delta int) {
	if len(p.items) == 0 {
		return
	}
	next := p.cursor + delta
	if next < 0 {
		next = 0
	}
	if next >= len(p.items) {
		next = len(p.items) - 1
	}
	p.cursor = next
	p.ensureCursorVisible()
}

func (p *prActivityModel) GotoTop() {
	if len(p.items) > 0 {
		p.cursor = 0
		p.viewport.GotoTop()
	}
}

func (p *prActivityModel) GotoBottom() {
	if len(p.items) > 0 {
		p.cursor = len(p.items) - 1
		p.viewport.GotoBottom()
	}
}

func (p *prActivityModel) ensureCursorVisible() {
	cursorY := p.cursor * 2
	viewH := p.viewport.Height
	if cursorY < p.viewport.YOffset {
		p.viewport.SetYOffset(cursorY)
	}
	if cursorY+2 > p.viewport.YOffset+viewH {
		p.viewport.SetYOffset(cursorY - viewH + 2)
	}
}

func (p *prActivityModel) SetSize(width, height int) {
	p.width = width
	p.height = height
	p.viewport.Width = width
	p.viewport.Height = height
}

func (p *prActivityModel) syncStyles(s *Styles) {
	p.styles = s
}

func (p *prActivityModel) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.MouseMsg:
		if msg.Action == tea.MouseActionPress || msg.Action == tea.MouseActionRelease {
			return nil
		}
		if z := zone.Get(p.zoneID); z != nil && z.InBounds(msg) {
			vp, cmd := p.viewport.Update(msg)
			p.viewport = vp
			return cmd
		}
		return nil
	}

	vp, cmd := p.viewport.Update(msg)
	p.viewport = vp
	return cmd
}

func (p *prActivityModel) View() string {
	var lines []string

	for i, pr := range p.items {
		line := p.renderPRLine(&pr, i == p.cursor, p.width)
		lines = append(lines, line)
	}

	content := strings.Join(lines, "\n")

	if len(p.items) <= p.height {
		styled := lipgloss.NewStyle().
			Width(p.width).
			Height(p.height).
			Render(content)
		return zone.Mark(p.zoneID, styled)
	}

	p.viewport.SetContent(content)
	return zone.Mark(p.zoneID, p.viewport.View())
}

func (p *prActivityModel) renderPRLine(pr *PRLine, selected bool, width int) string {
	prefix := "  "
	sym := "  "
	stateColor := dimStyle
	switch pr.State {
	case "OPEN", "open":
		sym = greenStyle.Render("○")
	case "MERGED", "merged":
		sym = doneStyle.Render("✓")
	case "CLOSED", "closed":
		sym = wipColorStyle().Render("✗")
	}

	ciStatus := ""
	if pr.CIState != "" {
		switch pr.CIState {
		case "success":
			ciStatus = " " + greenStyle.Render("CI:✓")
		case "failure":
			ciStatus = " " + wipColorStyle().Render("CI:✗")
		case "pending":
			ciStatus = " " + wipColorStyle().Render("CI:…")
		default:
			ciStatus = " " + dimStyle.Render("CI:"+pr.CIState)
		}
	}

	repoName := dimStyle.Render("[" + pr.Repo + "]")
	status := stateColor
	_ = status

	prInfo := fmt.Sprintf("%s%s %s #%d", prefix, sym, pr.YakName, pr.PRNum)
	if ciStatus != "" {
		prInfo += ciStatus
	}

	line := fmt.Sprintf("%s %s", prInfo, repoName)
	if selected {
		line = bgSelected.Render(line)
	}

	return padWidth(line, width)
}

func wipColorStyle() lipgloss.Style {
	return wipStyle
}

func collectPRs(repos []Repo) []PRLine {
	var prs []PRLine
	for _, repo := range repos {
		for _, yak := range repo.Yaks {
			if yak.PRURL == "" {
				continue
			}
			prNum := parsePRNumber(yak.PRURL)
			prs = append(prs, PRLine{
				YakName: yak.Name,
				Repo:    repo.Name,
				PRURL:   yak.PRURL,
				PRNum:   prNum,
			})
		}
	}
	return prs
}

func parsePRNumber(url string) int {
	parts := strings.Split(strings.TrimRight(url, "/"), "/")
	if len(parts) == 0 {
		return 0
	}
	last := parts[len(parts)-1]
	n, err := strconv.Atoi(last)
	if err != nil {
		return 0
	}
	return n
}
