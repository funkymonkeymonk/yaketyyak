package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
)

const listItemHeight = 1

type listViewportModel struct {
	viewport viewport.Model
	items    []treeLine
	cursor   int
	width    int
	height   int
	styles   *Styles
	zoneID   string
}

func newListViewport() listViewportModel {
	vp := viewport.New(40, 20)
	vp.MouseWheelEnabled = true
	vp.MouseWheelDelta = 3

	return listViewportModel{
		viewport: vp,
		zoneID:   zone.NewPrefix(),
	}
}

func (lv *listViewportModel) SetItems(items []treeLine) {
	lv.items = items
	if lv.cursor >= len(lv.items) && len(lv.items) > 0 {
		lv.cursor = len(lv.items) - 1
	}
}

func (lv *listViewportModel) Cursor() int {
	return lv.cursor
}

func (lv *listViewportModel) SetCursor(c int) {
	if c < 0 {
		c = 0
	}
	if c >= len(lv.items) {
		c = len(lv.items) - 1
	}
	lv.cursor = c
	lv.ensureCursorVisible()
}

func (lv *listViewportModel) MoveCursor(delta int) {
	if len(lv.items) == 0 {
		return
	}
	next := lv.cursor + delta
	if next < 0 {
		next = 0
	}
	if next >= len(lv.items) {
		next = len(lv.items) - 1
	}
	lv.cursor = next
	lv.ensureCursorVisible()
}

func (lv *listViewportModel) GotoTop() {
	if len(lv.items) > 0 {
		lv.cursor = 0
		lv.viewport.GotoTop()
	}
}

func (lv *listViewportModel) GotoBottom() {
	if len(lv.items) > 0 {
		lv.cursor = len(lv.items) - 1
		lv.viewport.GotoBottom()
	}
}

func (lv *listViewportModel) ensureCursorVisible() {
	cursorY := lv.cursor * listItemHeight
	viewH := lv.viewport.Height

	if cursorY < lv.viewport.YOffset {
		lv.viewport.SetYOffset(cursorY)
	}
	if cursorY+listItemHeight > lv.viewport.YOffset+viewH {
		lv.viewport.SetYOffset(cursorY - viewH + listItemHeight)
	}
}

func (lv *listViewportModel) SetSize(width, height int) {
	lv.width = width
	lv.height = height
	lv.viewport.Width = width
	lv.viewport.Height = height
}

func (lv *listViewportModel) syncStyles(s *Styles) {
	lv.styles = s
}

func (lv *listViewportModel) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.MouseMsg:
		if msg.Action == tea.MouseActionPress || msg.Action == tea.MouseActionRelease {
			return nil
		}
		if z := zone.Get(lv.zoneID); z != nil && z.InBounds(msg) {
			vp, cmd := lv.viewport.Update(msg)
			lv.viewport = vp
			return cmd
		}
		return nil
	}

	vp, cmd := lv.viewport.Update(msg)
	lv.viewport = vp
	return cmd
}

func (lv *listViewportModel) View() string {
	var lines []string
	for i := 0; i < len(lv.items); i++ {
		lines = append(lines, renderTreeLine(&lv.items[i], i == lv.cursor, lv.width))
	}
	content := strings.Join(lines, "\n")

	if len(lv.items) <= lv.height {
		styled := lipgloss.NewStyle().
			Width(lv.width).
			Height(lv.height).
			Render(content)
		return zone.Mark(lv.zoneID, styled)
	}

	lv.viewport.SetContent(content)
	v := lv.viewport.View()
	return zone.Mark(lv.zoneID, v)
}
