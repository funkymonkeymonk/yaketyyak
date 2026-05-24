package tui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"

	"go.temporal.io/api/enums/v1"
	historypb "go.temporal.io/api/history/v1"
	"go.temporal.io/sdk/client"

	"github.com/funkymonkeymonk/yaketyyak/temporal"
)

type temporalMsg struct {
	repoIdx int
	state   *temporal.WorkflowState
	err     error
}

type temporalAllMsg struct {
	states      []*temporal.WorkflowState
	shaveStates []*temporal.ShaveState
	errs        []error
}

type signalMsg struct {
	action string
	err    error
}

type editorDoneMsg struct {
	err error
}

type activitySpan struct {
	name     string
	duration time.Duration
	offset   time.Duration
	status   string
	errMsg   string
	eventIDs []int64
}

type historyMsg struct {
	wfID   string
	status string
	lines  []string
	spans  []activitySpan
	events []*historypb.HistoryEvent
	err    error
}

type refreshMsg struct {
	repos []Repo
	err   error
}

type Model struct {
	repos     []Repo
	treeLines []treeLine
	prLines   []PRLine
	llmCfg    temporal.LLMConfig

	width  int
	height int

	ctx ProgramContext

	keys KeyMap

	tabs       tabsModel
	listView   listViewportModel
	prActivity prActivityModel
	sidePanel  sidebarModel
	statusBar  footerModel

	showConfirm   bool
	confirmMsg    string
	confirmAction string

	showHelp    bool
	statusMsg   string
	statusTimer int

	showHistory            bool
	historyPanelFocus      string
	historyTimelineCur     int
	historyTimelineReverse bool
	historyFilter          string
	historyWfID            string
	historyStatus          string
	historyLines           []string
	historyEvents          []*historypb.HistoryEvent
	historySpans           []activitySpan
	historyEventsOffset    int
	historyTimelineOffset  int
}

func init() {
	zone.NewGlobal()
}

func New(repos []Repo, llmCfg temporal.LLMConfig) Model {
	m := Model{
		repos:      repos,
		prLines:    collectPRs(repos),
		llmCfg:     llmCfg,
		width:      80,
		height:     24,
		ctx:        defaultContext(),
		keys:       DefaultKeyMap(),
		tabs:       newTabsModel(),
		listView:   newListViewport(),
		prActivity: newPRActivity(),
		sidePanel:  newSidebar(),
		statusBar:  newFooter(),
	}
	m.prActivity.SetItems(m.prLines)
	return m
}

func (m *Model) buildTree() {
	m.treeLines = nil
	for ri, repo := range m.repos {
		m.treeLines = append(m.treeLines, treeLine{
			kind:    treeRepo,
			repoIdx: ri,
			yakIdx:  -1,
			name:    repo.Name,
			depth:   0,
		})
		for yi, yak := range repo.Yaks {
			m.treeLines = append(m.treeLines, treeLine{
				kind:              treeYak,
				repoIdx:           ri,
				yakIdx:            yi,
				name:              yak.Name,
				depth:             yak.Depth + 1,
				state:             yak.State,
				prURL:             yak.PRURL,
				hasChildren:       yak.HasChildren,
				isLastSibling:     yak.IsLastSibling,
				ancestorContinues: shiftContinues(yak.AncestorContinues),
			})
		}
	}
}

func shiftContinues(c []bool) []bool {
	out := make([]bool, 0, len(c)+1)
	out = append(out, true)
	out = append(out, c...)
	return out
}

const pollInterval = 5 * time.Second

type pollTickMsg struct{}

func (m *Model) Init() tea.Cmd {
	m.buildTree()
	m.listView.SetItems(m.treeLines)
	m.prActivity.SetItems(m.prLines)
	m.syncProgramContext()
	m.updateSidebar()
	return tea.Batch(
		m.queryAllTemporal(),
		tea.Tick(pollInterval, func(t time.Time) tea.Msg { return pollTickMsg{} }),
	)
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.syncProgramContext()
		m.updateSidebar()
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case temporalAllMsg:
		for i, st := range msg.states {
			if i < len(m.repos) {
				m.repos[i].WFState = st
			}
		}
		for i, ss := range msg.shaveStates {
			if i < len(m.repos) {
				m.repos[i].ShaveState = ss
			}
		}
		m.updateSidebar()
		return m, nil

	case signalMsg:
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Signal %s failed: %v", msg.action, msg.err)
		} else {
			m.statusMsg = fmt.Sprintf("Signal %s sent", msg.action)
		}
		m.statusTimer = 3
		return m, tea.Batch(m.queryAllTemporal(), tickStatus())

	case editorDoneMsg:
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Editor: %v", msg.err)
			m.statusTimer = 3
		}
		return m, tea.Batch(m.refresh(), m.queryAllTemporal(), tickStatus())

	case refreshMsg:
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Refresh: %v", msg.err)
			m.statusTimer = 3
			return m, tickStatus()
		}
		m.repos = msg.repos
		m.prLines = collectPRs(msg.repos)
		m.buildTree()
		m.listView.SetItems(m.treeLines)
		m.prActivity.SetItems(m.prLines)
		m.listView.SetCursor(m.listView.Cursor())
		m.updateSidebar()
		return m, nil

	case historyMsg:
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("History: %v", msg.err)
			m.showHistory = false
			return m, nil
		}
		m.historyStatus = msg.status
		m.historyLines = msg.lines
		m.historyEvents = msg.events
		m.historySpans = msg.spans
		if m.historyEventsOffset >= len(msg.lines) {
			m.historyEventsOffset = max(0, len(msg.lines)-1)
		}
		if m.historyTimelineOffset >= len(msg.spans) {
			m.historyTimelineOffset = max(0, len(msg.spans)-1)
		}
		return m, nil

	case pollTickMsg:
		cmds := []tea.Cmd{
			m.queryAllTemporal(),
			tea.Tick(pollInterval, func(t time.Time) tea.Msg { return pollTickMsg{} }),
		}
		if m.showHistory {
			cmds = append(cmds, m.fetchWorkflowHistory(m.historyWfID))
		}
		return m, tea.Batch(cmds...)

	case statusTickMsg:
		return m, m.tickStatus()
	}

	var cmds []tea.Cmd

	cmd := m.listView.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	cmd = m.prActivity.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	cmd = m.sidePanel.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	cmd = m.statusBar.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) View() string {
	if m.showHistory {
		return m.renderHistoryView()
	}

	if len(m.repos) == 0 {
		return centerText(m.width, m.height, "No repos with .yaks/ found.\nRun `yx add <name>` to create a task.\n\nRefresh: Ctrl+R")
	}

	tabRow := lipgloss.NewStyle().
		Width(m.width).
		Background(lipgloss.Color("#2A2A2A")).
		PaddingLeft(1).
		Render(m.tabs.View())

	if m.tabs.Active() == TabYaks {
		if len(m.treeLines) == 0 {
			content := centerText(m.width, m.ctx.MainContentHeight, "No yaks found.\nRun `yx add <name>` to create one.\n\nRefresh: Ctrl+R")
			return lipgloss.JoinVertical(lipgloss.Left, tabRow, content, m.statusBar.View(m.statusMsg, m.showHelp, len(m.repos), m.totalYaks(), ""))
		}
		return m.renderYaksView(tabRow)
	}

	return m.renderPRsView(tabRow)
}

func (m *Model) renderYaksView(tabRow string) string {
	treeView := m.listView.View()
	sidebarView := m.sidePanel.View()

	var mainContent string
	if m.ctx.SidebarOpen && m.ctx.PreviewWidth > 0 {
		mainContent = lipgloss.JoinHorizontal(lipgloss.Top, treeView, sidebarView)
	} else {
		mainContent = treeView
	}

	phase := ""
	if cur := m.currentTL(); cur != nil && cur.repoIdx < len(m.repos) {
		if wf := m.repos[cur.repoIdx].WFState; wf != nil {
			phase = wf.Phase
		}
	}
	statusView := m.statusBar.View(m.statusMsg, m.showHelp, len(m.repos), m.totalYaks(), phase)

	if m.showConfirm {
		return lipgloss.JoinVertical(lipgloss.Left, tabRow, m.renderConfirmContent(mainContent, statusView))
	}

	return lipgloss.JoinVertical(lipgloss.Left, tabRow, mainContent, statusView)
}

func (m *Model) renderPRsView(tabRow string) string {
	prView := m.prActivity.View()

	var mainContent string
	if m.ctx.SidebarOpen && m.ctx.PreviewWidth > 0 {
		sidebarView := m.sidePanel.View()
		mainContent = lipgloss.JoinHorizontal(lipgloss.Top, prView, sidebarView)
	} else {
		mainContent = prView
	}

	if m.showConfirm {
		return lipgloss.JoinVertical(lipgloss.Left, tabRow, m.renderConfirmContent(mainContent, ""))
	}

	statusView := m.statusBar.View(m.statusMsg, m.showHelp, len(m.repos), len(m.prLines), "")
	return lipgloss.JoinVertical(lipgloss.Left, tabRow, mainContent, statusView)
}

func (m *Model) renderHistoryView() string {
	statusStyle := cyanStyle
	switch m.historyStatus {
	case "FAILED", "WORKFLOW_EXECUTION_STATUS_FAILED":
		statusStyle = wipStyle
	case "COMPLETED", "WORKFLOW_EXECUTION_STATUS_COMPLETED":
		statusStyle = greenStyle
	case "RUNNING", "WORKFLOW_EXECUTION_STATUS_RUNNING":
		statusStyle = greenStyle
	}

	filterLabel := "all"
	if m.historyFilter != "" {
		filterLabel = m.historyFilter
	}
	sortLabel := "▼"
	if m.historyTimelineReverse {
		sortLabel = "▲"
	}

	header := lipgloss.JoinHorizontal(lipgloss.Top,
		headerStyle.Render(fmt.Sprintf("✦ Workflow History — %s", m.historyWfID)),
		"  ",
		statusStyle.Render("["+m.historyStatus+"]"),
		"  ",
		dimStyle.Render("filter:"+filterLabel+"  sort:"+sortLabel),
	)

	const fixedRows = 8 // header + help + separator + events-label + separator + timeline-label + separator + footer

	avail := m.ctx.MainContentHeight - fixedRows
	if avail < 4 {
		avail = 4
	}

	timelineContentHeight := len(m.historySpans)
	if timelineContentHeight < 1 {
		timelineContentHeight = 1
	}
	maxTimeline := avail / 3
	if timelineContentHeight > maxTimeline {
		timelineContentHeight = maxTimeline
	}
	if timelineContentHeight < 1 {
		timelineContentHeight = 1
	}

	eventsContentHeight := avail - timelineContentHeight
	if eventsContentHeight < 1 {
		eventsContentHeight = 1
	}

	focusEvents := m.historyPanelFocus == "events"
	eventsLabel := "Events ▼"
	timelineLabel := "Timeline"
	if focusEvents {
		eventsLabel = pinkBoldStyle.Render("Events ▼")
	} else {
		timelineLabel = pinkBoldStyle.Render("Timeline")
	}

	help := dimStyle.Render("↑/↓ scroll  tab focus  t sort  enter filter  q/esc back  (auto-refreshes)")

	chartWidth := m.width - 22
	if chartWidth < 20 {
		chartWidth = 20
	}

	lines := m.filteredEvents()
	totalDur := m.totalSpanDuration()

	var body strings.Builder
	body.WriteString(header)
	body.WriteString("\n")
	body.WriteString(help)
	body.WriteString("\n")
	body.WriteString(dimStyle.Render(strings.Repeat("─", m.width-2)))
	body.WriteString("\n")

	body.WriteString(eventsLabel)
	body.WriteString("\n")

	start := m.historyEventsOffset
	end := start + eventsContentHeight
	if end > len(lines) {
		end = len(lines)
	}
	renderedEventLines := end - start
	if renderedEventLines < 0 {
		renderedEventLines = 0
	}
	for i := start; i < end; i++ {
		body.WriteString(lines[i])
		body.WriteString("\n")
	}
	for i := renderedEventLines; i < eventsContentHeight; i++ {
		body.WriteString("\n")
	}

	body.WriteString(dimStyle.Render(strings.Repeat("·", m.width-2)))
	body.WriteString("\n")
	body.WriteString(timelineLabel)
	body.WriteString("\n")

	if len(m.historySpans) == 0 {
		body.WriteString(dimStyle.Render("  No activity spans found."))
		body.WriteString("\n")
	} else {
		tlStart := m.historyTimelineOffset
		tlEnd := tlStart + timelineContentHeight
		if tlEnd > len(m.historySpans) {
			tlEnd = len(m.historySpans)
		}

		for idx := tlStart; idx < tlEnd; idx++ {
			i := idx
			if m.historyTimelineReverse {
				i = len(m.historySpans) - 1 - idx
			}
			if i < 0 || i >= len(m.historySpans) {
				continue
			}
			cursor := ""
			if m.historyPanelFocus == "timeline" && i == m.historyTimelineCur {
				cursor = pinkBoldStyle.Render("▶ ")
			} else {
				cursor = "  "
			}
			body.WriteString(cursor)
			body.WriteString(formatSpanBar(m.historySpans[i], chartWidth, totalDur))
			body.WriteString("\n")
		}
	}

	body.WriteString(dimStyle.Render(strings.Repeat("─", m.width-2)))
	body.WriteString("\n")
	body.WriteString(fmt.Sprintf("events: %d  activities: %d  total: %s", len(lines), len(m.historySpans), durLabel(totalDur)))

	return m.finalizeHistoryView(body.String())
}

func (m *Model) finalizeHistoryView(body string) string {
	statusView := m.statusBar.View(m.statusMsg, false, len(m.repos), m.totalYaks(), "")
	return lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.NewStyle().Width(m.width).Background(lipgloss.Color("#2A2A2A")).Render("  Workflow History"),
		body,
		statusView,
	)
}

func (m *Model) filteredEvents() []string {
	if m.historyFilter == "" {
		return m.historyLines
	}
	var ids map[int64]bool
	for _, s := range m.historySpans {
		if s.name == m.historyFilter {
			ids = map[int64]bool{}
			for _, id := range s.eventIDs {
				ids[id] = true
			}
			break
		}
	}
	if ids == nil {
		return m.historyLines
	}
	var filtered []string
	for i, line := range m.historyLines {
		if i < len(m.historyEvents) && ids[m.historyEvents[i].EventId] {
			filtered = append(filtered, line)
		}
	}
	return filtered
}

func (m *Model) totalSpanDuration() time.Duration {
	var d time.Duration
	for _, s := range m.historySpans {
		end := s.offset + s.duration
		if end > d {
			d = end
		}
	}
	if d == 0 {
		d = time.Second
	}
	return d
}

func formatSpanBar(s activitySpan, width int, totalDur time.Duration) string {
	off := 0
	if totalDur > 0 {
		off = int(float64(s.offset) / float64(totalDur) * float64(width))
	}
	w := 0
	if totalDur > 0 {
		w = int(float64(s.duration) / float64(totalDur) * float64(width))
		if w < 1 {
			w = 1
		}
	}

	var bar strings.Builder
	for i := 0; i < width; i++ {
		if i >= off && i < off+w {
			bar.WriteRune('█')
		} else {
			bar.WriteRune('·')
		}
	}

	style := greenStyle
	marker := "✓"
	switch s.status {
	case "failed":
		style = wipStyle
		marker = "✗"
	case "timed_out":
		style = wipStyle
		marker = "⏱"
	case "running":
		style = cyanStyle
		marker = "…"
	}

	namePart := fmt.Sprintf("%-16s", s.name)
	if len(s.name) > 16 {
		namePart = s.name[:13] + "..."
	}

	line := fmt.Sprintf("%s %s %s", namePart, style.Render(bar.String()), style.Render(" "+marker+" "+durLabel(s.duration)))
	if s.errMsg != "" {
		line += "  " + dimStyle.Render(s.errMsg)
	}
	return line
}

func durLabel(d time.Duration) string {
	switch {
	case d < time.Second:
		return fmt.Sprintf("%dms", d.Milliseconds())
	case d < time.Minute:
		return fmt.Sprintf("%.1fs", d.Seconds())
	default:
		m := d / time.Minute
		s := (d % time.Minute) / time.Second
		return fmt.Sprintf("%dm%ds", m, s)
	}
}

func (m *Model) renderConfirmContent(mainContent, statusView string) string {
	overlay := lipgloss.JoinVertical(lipgloss.Left,
		m.confirmMsg,
		"",
		dimStyle.Render(" [enter] confirm  [n/esc] cancel"),
	)
	overlayW := 60
	overlayStyled := m.ctx.Styles.Common.OverlayStyle.Copy().Width(overlayW).Render(overlay)

	height := m.ctx.MainContentHeight + FooterHeight
	return lipgloss.Place(
		m.width, height,
		lipgloss.Center, lipgloss.Center,
		overlayStyled,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.NoColor{}),
		lipgloss.WithWhitespaceBackground(lipgloss.NoColor{}),
	)
}

func (m *Model) totalYaks() int {
	total := 0
	for _, repo := range m.repos {
		total += len(repo.Yaks)
	}
	return total
}

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.showHistory {
		switch {
		case key.Matches(msg, m.keys.Quit), key.Matches(msg, m.keys.Cancel):
			m.showHistory = false
			m.historyFilter = ""
			return m, nil
		case len(msg.Runes) == 1 && (msg.Runes[0] == 't' || msg.Runes[0] == 'T'):
			m.historyTimelineReverse = !m.historyTimelineReverse
			m.historyTimelineOffset = 0
			return m, nil
		case key.Matches(msg, m.keys.NextTab):
			if m.historyPanelFocus == "timeline" {
				m.historyPanelFocus = "events"
			} else {
				m.historyPanelFocus = "timeline"
			}
			m.historyEventsOffset = 0
			m.historyTimelineOffset = 0
			return m, nil
		case key.Matches(msg, m.keys.Select):
			if m.historyPanelFocus == "timeline" {
				if m.historyTimelineCur < len(m.historySpans) {
					name := m.historySpans[m.historyTimelineCur].name
					if m.historyFilter == name {
						m.historyFilter = ""
					} else {
						m.historyFilter = name
					}
					m.historyEventsOffset = 0
				}
				return m, nil
			}
			m.historyFilter = ""
			m.historyEventsOffset = 0
			m.historyTimelineOffset = 0
			return m, nil
		case key.Matches(msg, m.keys.Down):
			if m.historyPanelFocus == "timeline" {
				if m.historyTimelineCur < len(m.historySpans)-1 {
					m.historyTimelineCur++
				}
				if m.historyTimelineCur >= m.historyTimelineOffset+timelineVisibleRows(m) && m.historyTimelineOffset < len(m.historySpans)-1 {
					m.historyTimelineOffset++
				}
			} else {
				maxScroll := max(0, len(m.filteredEvents())-eventsVisibleRows(m))
				if m.historyEventsOffset < maxScroll {
					m.historyEventsOffset++
				}
			}
			return m, nil
		case key.Matches(msg, m.keys.Up):
			if m.historyPanelFocus == "timeline" {
				if m.historyTimelineCur > 0 {
					m.historyTimelineCur--
				}
				if m.historyTimelineCur < m.historyTimelineOffset {
					m.historyTimelineOffset = m.historyTimelineCur
				}
			} else {
				if m.historyEventsOffset > 0 {
					m.historyEventsOffset--
				}
			}
			return m, nil
		case key.Matches(msg, m.keys.Home):
			m.historyEventsOffset = 0
			m.historyTimelineOffset = 0
			m.historyTimelineCur = 0
			return m, nil
		case key.Matches(msg, m.keys.End):
			if m.historyPanelFocus == "timeline" {
				m.historyTimelineCur = max(0, len(m.historySpans)-1)
				m.historyTimelineOffset = max(0, len(m.historySpans)-timelineVisibleRows(m))
			} else {
				m.historyEventsOffset = max(0, len(m.filteredEvents())-eventsVisibleRows(m))
			}
			return m, nil
		}
		return m, nil
	}

	if m.showConfirm {
		return m.handleConfirmKey(msg)
	}

	if m.showHelp {
		if key.Matches(msg, m.keys.ToggleHelp) {
			m.showHelp = false
			return m, nil
		}
		m.showHelp = false
		return m, nil
	}

	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit

	case key.Matches(msg, m.keys.ToggleHelp):
		m.showHelp = true
		return m, nil

	case key.Matches(msg, m.keys.NextTab):
		m.tabs.Next()
		m.refreshActiveTab()
		return m, nil

	case key.Matches(msg, m.keys.PrevTab):
		m.tabs.Prev()
		m.refreshActiveTab()
		return m, nil

	case key.Matches(msg, m.keys.Down):
		if m.tabs.Active() == TabYaks {
			m.listView.MoveCursor(1)
		} else {
			m.prActivity.MoveCursor(1)
		}
		m.updateSidebar()
		return m, nil

	case key.Matches(msg, m.keys.Up):
		if m.tabs.Active() == TabYaks {
			m.listView.MoveCursor(-1)
		} else {
			m.prActivity.MoveCursor(-1)
		}
		m.updateSidebar()
		return m, nil

	case key.Matches(msg, m.keys.Home):
		if m.tabs.Active() == TabYaks {
			m.listView.GotoTop()
		} else {
			m.prActivity.GotoTop()
		}
		m.updateSidebar()
		return m, nil

	case key.Matches(msg, m.keys.End):
		if m.tabs.Active() == TabYaks {
			m.listView.GotoBottom()
		} else {
			m.prActivity.GotoBottom()
		}
		m.updateSidebar()
		return m, nil

	case key.Matches(msg, m.keys.Refresh):
		return m, tea.Batch(m.refresh(), m.queryAllTemporal(), tickStatus())

	case key.Matches(msg, m.keys.Select):
		if m.tabs.Active() == TabPRs {
			return m.openPR()
		}
		return m.openContext()

	case key.Matches(msg, m.keys.Edit):
		if m.tabs.Active() == TabPRs {
			return m, nil
		}
		return m, m.editContext()

	case key.Matches(msg, m.keys.Start):
		if m.tabs.Active() == TabPRs {
			return m, nil
		}
		return m.sendSignal("start")

	case key.Matches(msg, m.keys.Shave):
		if m.tabs.Active() == TabPRs {
			return m, nil
		}
		return m.shaveYak()

	case key.Matches(msg, m.keys.Pause):
		if m.tabs.Active() == TabPRs {
			return m, nil
		}
		return m.sendSignal("pause")

	case key.Matches(msg, m.keys.Resume):
		if m.tabs.Active() == TabPRs {
			return m, nil
		}
		return m.sendSignal("resume")

	case key.Matches(msg, m.keys.G2G):
		if m.tabs.Active() == TabPRs {
			return m, nil
		}
		return m.g2gScan()

	case key.Matches(msg, m.keys.CI):
		if m.tabs.Active() == TabPRs {
			return m, nil
		}
		return m.ciSignal()

	case key.Matches(msg, m.keys.History):
		return m.showWorkflowHistory()

	case key.Matches(msg, m.keys.Tab1):
		m.tabs.SetActive(TabYaks)
		m.refreshActiveTab()
		return m, nil

	case key.Matches(msg, m.keys.Tab2):
		m.tabs.SetActive(TabPRs)
		m.refreshActiveTab()
		return m, nil
	}

	return m, nil
}

func (m *Model) handleConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Confirm):
		m.showConfirm = false
		return m.runConfirmAction()

	case key.Matches(msg, m.keys.Cancel):
		m.showConfirm = false
		return m, nil
	}
	return m, nil
}

func (m *Model) runConfirmAction() (tea.Model, tea.Cmd) {
	switch m.confirmAction {
	case "start":
		return m.doSendSignal("start")
	case "pause":
		return m.doSendSignal("pause")
	case "resume":
		return m.doSendSignal("resume")
	case "shave":
		return m.doShaveYak()
	case "g2g":
		return m.doG2GScan()
	case "ci":
		return m.doCiSignal()
	}
	return m, nil
}

func (m *Model) confirmBefore(action string, msg string) (tea.Model, tea.Cmd) {
	m.showConfirm = true
	m.confirmMsg = msg
	m.confirmAction = action
	return m, nil
}

func (m *Model) updateSidebar() {
	w := m.ctx.PreviewWidth
	if w < 10 {
		w = 10
	}

	if m.tabs.Active() == TabPRs {
		pr := m.prActivity.SelectedPR()
		if pr != nil {
			m.sidePanel.ShowPR(pr, w)
		} else {
			m.sidePanel.ShowPRActivityEmpty(w)
		}
		return
	}

	cursor := m.listView.Cursor()
	if cursor < 0 || cursor >= len(m.treeLines) {
		return
	}
	tl := m.treeLines[cursor]
	if tl.kind == treeRepo {
		m.sidePanel.ShowRepo(&m.repos[tl.repoIdx], w)
	} else {
		repo := m.repos[tl.repoIdx]
		m.sidePanel.ShowYak(&repo.Yaks[tl.yakIdx], repo.WFState, repo.ShaveState, w)
	}
}

func (m *Model) currentTL() *treeLine {
	cursor := m.listView.Cursor()
	if cursor < 0 || cursor >= len(m.treeLines) {
		return nil
	}
	return &m.treeLines[cursor]
}

func (m *Model) currentRepo() *Repo {
	tl := m.currentTL()
	if tl == nil {
		return nil
	}
	if tl.repoIdx < 0 || tl.repoIdx >= len(m.repos) {
		return nil
	}
	return &m.repos[tl.repoIdx]
}

func (m *Model) currentWFID() string {
	r := m.currentRepo()
	if r == nil {
		return ""
	}
	return r.WFID
}

func (m *Model) refreshActiveTab() {
	m.updateSidebar()
}

func (m *Model) openPR() (tea.Model, tea.Cmd) {
	pr := m.prActivity.SelectedPR()
	if pr == nil || pr.PRURL == "" {
		return m, nil
	}
	return m, tea.ExecProcess(
		exec.Command("open", pr.PRURL),
		func(err error) tea.Msg { return editorDoneMsg{err} },
	)
}

func (m *Model) openContext() (tea.Model, tea.Cmd) {
	tl := m.currentTL()
	if tl == nil || tl.kind != treeYak {
		return m, nil
	}
	repo := m.repos[tl.repoIdx]
	contextPath := filepath.Join(repo.YaksDir, repo.Yaks[tl.yakIdx].Path, ".context.md")
	if _, err := os.Stat(contextPath); err == nil {
		return m, tea.ExecProcess(
			exec.Command("less", "-R", contextPath),
			func(err error) tea.Msg { return editorDoneMsg{err} },
		)
	}
	return m, nil
}

func (m *Model) editContext() tea.Cmd {
	tl := m.currentTL()
	if tl == nil || tl.kind != treeYak {
		return nil
	}
	repo := m.repos[tl.repoIdx]
	contextPath := filepath.Join(repo.YaksDir, repo.Yaks[tl.yakIdx].Path, ".context.md")
	os.MkdirAll(filepath.Dir(contextPath), 0755)
	if _, err := os.Stat(contextPath); os.IsNotExist(err) {
		os.WriteFile(contextPath, []byte{}, 0644)
	}
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		editor = "vim"
	}
	return tea.ExecProcess(
		exec.Command(editor, contextPath),
		func(err error) tea.Msg { return editorDoneMsg{err} },
	)
}

func (m *Model) shaveYak() (tea.Model, tea.Cmd) {
	tl := m.currentTL()
	if tl == nil || tl.kind != treeYak {
		m.statusMsg = "Select a yak to shave"
		m.statusTimer = 3
		return m, tickStatus()
	}
	return m.confirmBefore("shave", fmt.Sprintf("Shave yak: %s?", m.repos[tl.repoIdx].Yaks[tl.yakIdx].Name))
}

func (m *Model) doShaveYak() (tea.Model, tea.Cmd) {
	tl := m.currentTL()
	if tl == nil || tl.kind != treeYak {
		return m, nil
	}
	repo := &m.repos[tl.repoIdx]
	yakName := repo.Yaks[tl.yakIdx].Name
	wfID := temporal.ShaveWorkflowID(yakName)

	repo.ShaveState = &temporal.ShaveState{
		YakName: yakName,
		Phase:   "starting",
	}

	m.statusMsg = fmt.Sprintf("Shaving yak: %s...", yakName)
	m.statusTimer = 3
	return m, tea.Batch(
		func() tea.Msg {
			cl, err := temporal.NewClient()
			if err != nil {
				return signalMsg{action: fmt.Sprintf("shave %s", yakName), err: fmt.Errorf("connect: %w", err)}
			}
			defer cl.Close()
			_, err = cl.ExecuteWorkflow(context.Background(), client.StartWorkflowOptions{
				ID:        wfID,
				TaskQueue: temporal.TaskQueue,
			}, "ShaveWorkflow", yakName, m.llmCfg, repo.Remote, repo.Root, 3)
			if err != nil {
				return signalMsg{action: fmt.Sprintf("shave %s", yakName), err: fmt.Errorf("start: %w", err)}
			}

			time.Sleep(2 * time.Second)

			desc, descErr := cl.DescribeWorkflowExecution(context.Background(), wfID, "")
			if descErr != nil {
				return signalMsg{action: fmt.Sprintf("shave %s (%s)", yakName, wfID), err: nil}
			}

			status := desc.WorkflowExecutionInfo.Status
			if status == enums.WORKFLOW_EXECUTION_STATUS_FAILED {
				return signalMsg{action: fmt.Sprintf("shave %s", yakName),
					err: fmt.Errorf("workflow failed — check Temporal UI: %s", wfID)}
			}
			if status == enums.WORKFLOW_EXECUTION_STATUS_TERMINATED {
				return signalMsg{action: fmt.Sprintf("shave %s", yakName),
					err: fmt.Errorf("workflow terminated: %s", wfID)}
			}
			if status == enums.WORKFLOW_EXECUTION_STATUS_TIMED_OUT {
				return signalMsg{action: fmt.Sprintf("shave %s", yakName),
					err: fmt.Errorf("workflow timed out: %s", wfID)}
			}

			return signalMsg{action: fmt.Sprintf("shave %s (%s)", yakName, wfID), err: nil}
		},
		tickStatus(),
	)
}

func (m *Model) sendSignal(action string) (tea.Model, tea.Cmd) {
	wfID := m.currentWFID()
	if wfID == "" {
		return m, nil
	}
	repo := m.currentRepo()
	if repo == nil {
		return m, nil
	}

	switch action {
	case "start":
		if repo.WFState != nil {
			m.statusMsg = "Workflow already running"
			m.statusTimer = 3
			return m, tickStatus()
		}
		return m.confirmBefore("start", fmt.Sprintf("Start workflow for %s?", repo.Name))
	case "pause":
		return m.confirmBefore("pause", fmt.Sprintf("Pause workflow for %s?", repo.Name))
	case "resume":
		return m.confirmBefore("resume", fmt.Sprintf("Resume workflow for %s?", repo.Name))
	}

	return m, nil
}

func (m *Model) doSendSignal(action string) (tea.Model, tea.Cmd) {
	wfID := m.currentWFID()
	if wfID == "" {
		return m, nil
	}
	repo := m.currentRepo()

	switch action {
	case "start":
		return m, func() tea.Msg {
			cl, err := temporal.NewClient()
			if err != nil {
				return signalMsg{action: "start", err: err}
			}
			defer cl.Close()
			_, err = cl.ExecuteWorkflow(context.Background(), client.StartWorkflowOptions{
				ID:        wfID,
				TaskQueue: temporal.TaskQueue,
			}, "BarberWorkflow", repo.Remote, repo.Root, m.llmCfg, false, 60)
			return signalMsg{action: "start", err: err}
		}
	case "pause":
		return m, func() tea.Msg {
			cl, err := temporal.NewClient()
			if err != nil {
				return signalMsg{action: "pause", err: err}
			}
			defer cl.Close()
			err = cl.SignalWorkflow(context.Background(), wfID, "", "pause", nil)
			return signalMsg{action: "pause", err: err}
		}
	case "resume":
		return m, func() tea.Msg {
			cl, err := temporal.NewClient()
			if err != nil {
				return signalMsg{action: "resume", err: err}
			}
			defer cl.Close()
			err = cl.SignalWorkflow(context.Background(), wfID, "", "resume", nil)
			return signalMsg{action: "resume", err: err}
		}
	}

	return m, nil
}

func (m *Model) g2gScan() (tea.Model, tea.Cmd) {
	wfID := m.currentWFID()
	if wfID == "" {
		return m, nil
	}
	return m.confirmBefore("g2g", "Send g2g scan signal?")
}

func (m *Model) doG2GScan() (tea.Model, tea.Cmd) {
	wfID := m.currentWFID()
	if wfID == "" {
		return m, nil
	}
	m.statusMsg = "Sending g2g signal..."
	return m, tea.Batch(
		func() tea.Msg {
			cl, err := temporal.NewClient()
			if err != nil {
				return signalMsg{action: "g2g_scan", err: err}
			}
			defer cl.Close()
			err = cl.SignalWorkflow(context.Background(), wfID, "", "g2g_signal", nil)
			return signalMsg{action: "g2g_scan", err: err}
		},
		tickStatus(),
	)
}

func (m *Model) ciSignal() (tea.Model, tea.Cmd) {
	wfID := m.currentWFID()
	if wfID == "" {
		return m, nil
	}
	return m.confirmBefore("ci", "Send CI success signal?")
}

func (m *Model) doCiSignal() (tea.Model, tea.Cmd) {
	wfID := m.currentWFID()
	if wfID == "" {
		return m, nil
	}
	m.statusMsg = "Sending CI signal..."
	return m, tea.Batch(
		func() tea.Msg {
			cl, err := temporal.NewClient()
			if err != nil {
				return signalMsg{action: "ci_signal", err: err}
			}
			defer cl.Close()
			payload := []interface{}{"success", "main", "", ""}
			err = cl.SignalWorkflow(context.Background(), wfID, "", "ci_signal", payload)
			return signalMsg{action: "ci_signal", err: err}
		},
		tickStatus(),
	)
}

func (m *Model) showWorkflowHistory() (tea.Model, tea.Cmd) {
	wfID := m.historyTargetWFID()
	if wfID == "" {
		return m, nil
	}
	m.showHistory = true
	m.historyFilter = ""
	m.historyTimelineReverse = false
	m.historyTimelineCur = 0
	m.historyTimelineOffset = 0
	m.historyPanelFocus = "events"
	m.historyWfID = wfID
	m.historyLines = []string{"Loading history..."}
	m.historySpans = nil
	m.historyEventsOffset = 0
	return m, m.fetchWorkflowHistory(wfID)
}

func timelineVisibleRows(m *Model) int {
	const fixedRows = 8
	avail := m.ctx.MainContentHeight - fixedRows
	if avail < 4 {
		avail = 4
	}
	n := len(m.historySpans)
	if n < 1 {
		n = 1
	}
	maxTimeline := avail / 3
	if n > maxTimeline {
		n = maxTimeline
	}
	return max(1, n)
}

func eventsVisibleRows(m *Model) int {
	const fixedRows = 8
	avail := m.ctx.MainContentHeight - fixedRows
	if avail < 4 {
		avail = 4
	}
	tlr := timelineVisibleRows(m)
	return max(1, avail-tlr)
}

func (m *Model) historyScrollMax() int {
	if m.historyPanelFocus == "timeline" {
		return max(0, len(m.historySpans)-timelineVisibleRows(m))
	}
	return max(0, len(m.filteredEvents())-eventsVisibleRows(m))
}

func (m *Model) historyTargetWFID() string {
	repo := m.currentRepo()
	if repo == nil {
		return ""
	}
	if repo.ShaveState != nil && repo.ShaveState.YakName != "" {
		return temporal.ShaveWorkflowID(repo.ShaveState.YakName)
	}
	return repo.WFID
}

func (m *Model) fetchWorkflowHistory(wfID string) tea.Cmd {
	return func() tea.Msg {
		cl, err := temporal.NewClient()
		if err != nil {
			return historyMsg{wfID: wfID, err: err}
		}
		defer cl.Close()

		status := ""
		desc, descErr := cl.DescribeWorkflowExecution(context.Background(), wfID, "")
		if descErr == nil {
			status = desc.WorkflowExecutionInfo.Status.String()
		}

		iter := cl.GetWorkflowHistory(context.Background(), wfID, "", false,
			enums.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT)
		var events []*historypb.HistoryEvent
		for iter.HasNext() {
			ev, err := iter.Next()
			if err != nil {
				return historyMsg{wfID: wfID, status: status, err: err}
			}
			events = append(events, ev)
		}

		actNames := map[int64]string{}
		for _, ev := range events {
			if a := ev.GetActivityTaskScheduledEventAttributes(); a != nil {
				if a.ActivityType != nil {
					actNames[ev.EventId] = a.ActivityType.GetName()
				}
			}
		}

		var lines []string
		for _, ev := range events {
			lines = append(lines, formatHistoryEvent(ev, actNames))
		}

		for i, j := 0, len(lines)-1; i < j; i, j = i+1, j-1 {
			lines[i], lines[j] = lines[j], lines[i]
		}

		return historyMsg{wfID: wfID, status: status, lines: lines, events: events, spans: extractSpans(events)}
	}
}

func extractSpans(events []*historypb.HistoryEvent) []activitySpan {
	if len(events) == 0 {
		return nil
	}

	type actRef struct {
		schedTime time.Time
		startTime time.Time
		name      string
		eventIDs  []int64
	}
	scheduled := map[int64]*actRef{}

	for _, ev := range events {
		if a := ev.GetActivityTaskScheduledEventAttributes(); a != nil {
			name := ""
			if a.ActivityType != nil {
				name = a.ActivityType.GetName()
			}
			tm := time.Time{}
			if ev.EventTime != nil {
				tm = ev.EventTime.AsTime()
			}
			scheduled[ev.EventId] = &actRef{schedTime: tm, name: name, eventIDs: []int64{ev.EventId}}
		}
		if a := ev.GetActivityTaskStartedEventAttributes(); a != nil {
			if ref, ok := scheduled[a.GetScheduledEventId()]; ok {
				if ev.EventTime != nil {
					ref.startTime = ev.EventTime.AsTime()
				}
				ref.eventIDs = append(ref.eventIDs, ev.EventId)
			}
		}
	}

	baseTime := time.Time{}
	if len(events) > 0 && events[0].EventTime != nil {
		baseTime = events[0].EventTime.AsTime()
	}

	var spans []activitySpan
	for _, ev := range events {
		appendFinish := func(ref *actRef, status, errMsg string) {
			spans = append(spans, activitySpan{
				name:     ref.name,
				offset:   ref.startTime.Sub(baseTime),
				duration: ev.EventTime.AsTime().Sub(ref.startTime),
				status:   status,
				errMsg:   errMsg,
				eventIDs: append(ref.eventIDs, ev.EventId),
			})
		}
		if a := ev.GetActivityTaskCompletedEventAttributes(); a != nil {
			ref, ok := scheduled[a.GetScheduledEventId()]
			if ok && !ref.startTime.IsZero() {
				appendFinish(ref, "completed", "")
			}
		}
		if a := ev.GetActivityTaskFailedEventAttributes(); a != nil {
			ref, ok := scheduled[a.GetScheduledEventId()]
			if ok && !ref.startTime.IsZero() {
				errMsg := ""
				if a.Failure != nil {
					errMsg = firstLine(a.Failure.GetMessage())
				}
				appendFinish(ref, "failed", errMsg)
			}
		}
		if a := ev.GetActivityTaskTimedOutEventAttributes(); a != nil {
			ref, ok := scheduled[a.GetScheduledEventId()]
			if ok && !ref.startTime.IsZero() {
				appendFinish(ref, "timed_out", "")
			}
		}
	}
	return spans
}

func formatHistoryEvent(ev *historypb.HistoryEvent, actNames map[int64]string) string {
	ts := ""
	if ev.EventTime != nil {
		ts = ev.EventTime.AsTime().Format("15:04:05")
	}
	detail := formatEventDetail(ev, actNames)
	if detail != "" {
		return fmt.Sprintf("%4d  %s  %-35s  %s", ev.EventId, ts, ev.EventType.String(), detail)
	}
	return fmt.Sprintf("%4d  %s  %s", ev.EventId, ts, ev.EventType.String())
}

func formatEventDetail(ev *historypb.HistoryEvent, actNames map[int64]string) string {
	switch {
	case ev.GetWorkflowExecutionStartedEventAttributes() != nil:
		a := ev.GetWorkflowExecutionStartedEventAttributes()
		if a.WorkflowType != nil {
			return wipStyle.Render(a.WorkflowType.GetName())
		}
	case ev.GetActivityTaskScheduledEventAttributes() != nil:
		a := ev.GetActivityTaskScheduledEventAttributes()
		if a.ActivityType != nil {
			return cyanStyle.Render(a.ActivityType.GetName())
		}
	case ev.GetActivityTaskStartedEventAttributes() != nil:
		a := ev.GetActivityTaskStartedEventAttributes()
		name := actNames[a.GetScheduledEventId()]
		if name == "" {
			name = "?"
		}
		return fmt.Sprintf("%s %s", cyanStyle.Render(name), fmt.Sprintf("attempt %d", a.GetAttempt()))
	case ev.GetActivityTaskCompletedEventAttributes() != nil:
		a := ev.GetActivityTaskCompletedEventAttributes()
		name := actNames[a.GetScheduledEventId()]
		if name != "" {
			return greenStyle.Render(name + " ✓")
		}
		return greenStyle.Render("completed")
	case ev.GetActivityTaskFailedEventAttributes() != nil:
		a := ev.GetActivityTaskFailedEventAttributes()
		name := actNames[a.GetScheduledEventId()]
		if name == "" {
			name = "?"
		}
		errMsg := ""
		if a.Failure != nil {
			errMsg = firstLine(a.Failure.GetMessage())
		}
		if errMsg != "" {
			return wipStyle.Render(fmt.Sprintf("%s ✗ %s", name, errMsg))
		}
		return wipStyle.Render(name + " ✗")
	case ev.GetWorkflowExecutionFailedEventAttributes() != nil:
		a := ev.GetWorkflowExecutionFailedEventAttributes()
		if a.Failure != nil {
			return wipStyle.Render("error: " + firstLine(a.Failure.GetMessage()))
		}
		return wipStyle.Render("failed")
	case ev.GetActivityTaskTimedOutEventAttributes() != nil:
		a := ev.GetActivityTaskTimedOutEventAttributes()
		name := actNames[a.GetScheduledEventId()]
		if name != "" {
			return wipStyle.Render(name + " ⏱")
		}
		return wipStyle.Render("timed out")
	case ev.GetWorkflowExecutionCompletedEventAttributes() != nil:
		return greenStyle.Render("completed")
	case ev.GetWorkflowExecutionCanceledEventAttributes() != nil:
		return wipStyle.Render("canceled")
	case ev.GetWorkflowExecutionTerminatedEventAttributes() != nil:
		return wipStyle.Render("terminated")
	case ev.GetWorkflowExecutionTimedOutEventAttributes() != nil:
		return wipStyle.Render("timed out")
	case ev.GetTimerStartedEventAttributes() != nil:
		a := ev.GetTimerStartedEventAttributes()
		if d := a.GetStartToFireTimeout(); d != nil {
			return dimStyle.Render(d.AsDuration().String())
		}
	case ev.GetTimerFiredEventAttributes() != nil:
		return dimStyle.Render("fired")
	case ev.GetWorkflowTaskCompletedEventAttributes() != nil:
		return dimStyle.Render("completed")
	}
	return ""
}

func firstLine(s string) string {
	if idx := strings.IndexAny(s, "\n\r"); idx >= 0 {
		return s[:idx]
	}
	return s
}

func (m *Model) refresh() tea.Cmd {
	return func() tea.Msg {
		cwd, _ := os.Getwd()
		repos, err := ScanRepos(cwd)
		return refreshMsg{repos: repos, err: err}
	}
}

func (m *Model) queryAllTemporal() tea.Cmd {
	return func() tea.Msg {
		cl, err := temporal.NewClient()
		if err != nil {
			return temporalAllMsg{errs: []error{err}}
		}
		defer cl.Close()

		states := make([]*temporal.WorkflowState, len(m.repos))
		shaveStates := make([]*temporal.ShaveState, len(m.repos))
		errs := make([]error, len(m.repos))

		for i, repo := range m.repos {
			value, err := cl.QueryWorkflow(context.Background(), repo.WFID, "", "status")
			if err != nil {
				if !isWorkflowNotFound(err) {
					errs[i] = err
				}
			} else {
				var state temporal.WorkflowState
				if err := value.Get(&state); err != nil {
					errs[i] = err
				} else {
					states[i] = &state
				}
			}

			if repo.ShaveState != nil {
				wfID := temporal.ShaveWorkflowID(repo.ShaveState.YakName)

				desc, descErr := cl.DescribeWorkflowExecution(context.Background(), wfID, "")
				if descErr == nil && isTerminalStatus(desc.WorkflowExecutionInfo.Status) {
					shaveStates[i] = terminalShaveState(repo.ShaveState.YakName, desc.WorkflowExecutionInfo.Status)
				} else {
					sval, serr := cl.QueryWorkflow(context.Background(), wfID, "", "shave_status")
					if serr == nil {
						var shaveState temporal.ShaveState
						if serr := sval.Get(&shaveState); serr == nil {
							shaveStates[i] = &shaveState
						}
					}
				}
			}
		}
		return temporalAllMsg{states: states, shaveStates: shaveStates, errs: errs}
	}
}

func terminalShaveState(yakName string, status enums.WorkflowExecutionStatus) *temporal.ShaveState {
	phase := status.String()
	switch status {
	case enums.WORKFLOW_EXECUTION_STATUS_FAILED:
		phase = "failed"
	case enums.WORKFLOW_EXECUTION_STATUS_COMPLETED:
		phase = "done"
	case enums.WORKFLOW_EXECUTION_STATUS_CANCELED:
		phase = "cancelled"
	case enums.WORKFLOW_EXECUTION_STATUS_TERMINATED:
		phase = "terminated"
	case enums.WORKFLOW_EXECUTION_STATUS_TIMED_OUT:
		phase = "timed_out"
	}
	return &temporal.ShaveState{YakName: yakName, Phase: phase}
}

func isTerminalStatus(status enums.WorkflowExecutionStatus) bool {
	switch status {
	case enums.WORKFLOW_EXECUTION_STATUS_FAILED,
		enums.WORKFLOW_EXECUTION_STATUS_COMPLETED,
		enums.WORKFLOW_EXECUTION_STATUS_CANCELED,
		enums.WORKFLOW_EXECUTION_STATUS_TERMINATED,
		enums.WORKFLOW_EXECUTION_STATUS_TIMED_OUT:
		return true
	}
	return false
}

func isWorkflowNotFound(err error) bool {
	return strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "NotFound")
}

func centerText(w, h int, text string) string {
	lines := strings.Split(text, "\n")
	var result strings.Builder
	topPad := (h - len(lines)) / 2
	for i := 0; i < topPad; i++ {
		result.WriteString("\n")
	}
	for _, line := range lines {
		result.WriteString(lipgloss.NewStyle().Width(w).Align(lipgloss.Center).Render(line))
		result.WriteString("\n")
	}
	return result.String()
}

type statusTickMsg struct{}

func tickStatus() tea.Cmd {
	return func() tea.Msg { return statusTickMsg{} }
}

func (m *Model) tickStatus() tea.Cmd {
	if m.statusTimer > 0 {
		m.statusTimer--
		if m.statusTimer == 0 {
			m.statusMsg = ""
		}
	}
	return nil
}
