package tui

import (
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

	"github.com/funkymonkeymonk/yaketyyak/temporal"
)

type signalMsg struct {
	action string
	err    error
}

type temporalAllMsg struct {
	states      []*temporal.WorkflowState
	shaveStates []*temporal.ShaveState
}

type editorDoneMsg struct {
	err error
}

type refreshMsg struct {
	repos []Repo
	err   error
}

type Model struct {
	repos     []Repo
	treeLines []treeLine
	prLines   []PRLine

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

	// Temporal / shave workflow
	llmConfig temporal.LLMConfig

	// History view
	showHistory            bool
	historyWfID            string
	historyStatus          string
	historyLines           []string
	historySpans           []activitySpan
	historyTimelineReverse bool
	historyViewport        int // scroll offset
}

func init() {
	zone.NewGlobal()
}

func New(repos []Repo, llmCfg temporal.LLMConfig) Model {
	m := Model{
		repos:      repos,
		prLines:    collectPRs(repos),
		width:      80,
		height:     24,
		ctx:        defaultContext(),
		keys:       DefaultKeyMap(),
		tabs:       newTabsModel(),
		listView:   newListViewport(),
		prActivity: newPRActivity(),
		sidePanel:  newSidebar(),
		statusBar:  newFooter(),
		llmConfig:  llmCfg,
		// default timeline sort: most recent first
		historyTimelineReverse: true,
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

const pollInterval = 30 * time.Second

type pollTickMsg struct{}

func (m *Model) Init() tea.Cmd {
	m.buildTree()
	m.listView.SetItems(m.treeLines)
	m.prActivity.SetItems(m.prLines)
	m.syncProgramContext()
	m.updateSidebar()
	return tea.Batch(
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

	case temporalAllMsg:
		for i, state := range msg.shaveStates {
			if i < len(m.repos) {
				m.repos[i].ShaveState = state
			}
		}
		_ = msg.states
		m.updateSidebar()
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case signalMsg:
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Signal %s failed: %v", msg.action, msg.err)
		} else {
			m.statusMsg = fmt.Sprintf("Signal %s sent", msg.action)
		}
		m.statusTimer = 3
		return m, tickStatus()

	case editorDoneMsg:
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Editor: %v", msg.err)
			m.statusTimer = 3
		}
		return m, tea.Batch(m.refresh(), tickStatus())

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

	case pollTickMsg:
		cmds := []tea.Cmd{
			m.refresh(),
			tea.Tick(pollInterval, func(t time.Time) tea.Msg { return pollTickMsg{} }),
		}
		if m.showHistory && m.historyWfID != "" {
			cmds = append(cmds, m.fetchHistory(m.historyWfID))
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

	statusView := m.statusBar.View(m.statusMsg, m.showHelp, len(m.repos), m.totalYaks(), "")

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

func (m *Model) renderConfirmContent(mainContent, _ string) string {
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
	if m.showConfirm {
		return m.handleConfirmKey(msg)
	}

	if m.showHistory {
		return m.handleHistoryKey(msg)
	}

	if m.showHelp {
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
		return m, tea.Batch(m.refresh(), tickStatus())

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
		if m.confirmAction == "shave" {
			return m.doShaveYak()
		}
	case key.Matches(msg, m.keys.Cancel):
		m.showConfirm = false
	}
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
		m.sidePanel.ShowYak(&repo.Yaks[tl.yakIdx], &repo, w)
	}
}

func (m *Model) currentTL() *treeLine {
	cursor := m.listView.Cursor()
	if cursor < 0 || cursor >= len(m.treeLines) {
		return nil
	}
	return &m.treeLines[cursor]
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

func (m *Model) refresh() tea.Cmd {
	return func() tea.Msg {
		cwd, _ := os.Getwd()
		repos, err := ScanRepos(cwd)
		return refreshMsg{repos: repos, err: err}
	}
}

// ---------- shave yak workflow ----------

func (m *Model) shaveYak() (tea.Model, tea.Cmd) {
	tl := m.currentTL()
	if tl == nil || tl.kind != treeYak {
		return m, nil
	}
	yak := m.repos[tl.repoIdx].Yaks[tl.yakIdx]
	m.showConfirm = true
	m.confirmMsg = fmt.Sprintf("Shave yak: %s?", yak.Name)
	m.confirmAction = "shave"
	return m, nil
}

func (m *Model) doShaveYak() (tea.Model, tea.Cmd) {
	tl := m.currentTL()
	if tl == nil || tl.kind != treeYak {
		return m, nil
	}
	repoIdx := tl.repoIdx
	yak := m.repos[repoIdx].Yaks[tl.yakIdx]

	m.repos[repoIdx].ShaveState = &temporal.ShaveState{
		YakName: yak.Name,
		Phase:   "starting",
	}
	m.statusMsg = fmt.Sprintf("Starting shave for %s", yak.Name)
	m.statusTimer = 5

	return m, func() tea.Msg {
		return signalMsg{action: "shave " + yak.Name, err: nil}
	}
}

// terminalShaveState returns a ShaveState for a workflow that has reached a terminal status.
func terminalShaveState(yakName string, status int32) *temporal.ShaveState {
	phase := "unknown"
	switch status {
	case 2: // WORKFLOW_EXECUTION_STATUS_COMPLETED
		phase = "done"
	case 3: // WORKFLOW_EXECUTION_STATUS_FAILED
		phase = "failed"
	case 4: // WORKFLOW_EXECUTION_STATUS_CANCELED
		phase = "cancelled"
	case 5: // WORKFLOW_EXECUTION_STATUS_TERMINATED
		phase = "terminated"
	case 6: // WORKFLOW_EXECUTION_STATUS_CONTINUED_AS_NEW
		phase = "continued"
	case 7: // WORKFLOW_EXECUTION_STATUS_TIMED_OUT
		phase = "timed_out"
	}
	return &temporal.ShaveState{YakName: yakName, Phase: phase}
}

// ---------- workflow history ----------

func (m *Model) historyTargetWFID() string {
	tl := m.currentTL()
	if tl != nil && tl.kind == treeYak {
		repo := m.repos[tl.repoIdx]
		if repo.ShaveState != nil && repo.ShaveState.Phase != "" && repo.ShaveState.Phase != "done" && repo.ShaveState.Phase != "failed" && repo.ShaveState.Phase != "cancelled" {
			return temporal.ShaveWorkflowID(repo.ShaveState.YakName)
		}
	}
	if tl != nil && tl.kind == treeYak {
		return m.repos[tl.repoIdx].WFID
	}
	if tl != nil && tl.kind == treeRepo {
		return m.repos[tl.repoIdx].WFID
	}
	return ""
}

func (m *Model) showWorkflowHistory() (tea.Model, tea.Cmd) {
	wfID := m.historyTargetWFID()
	m.showHistory = true
	m.historyWfID = wfID
	m.historyLines = []string{"Loading…"}
	return m, m.fetchHistory(wfID)
}

func (m *Model) fetchHistory(_ string) tea.Cmd {
	// Stub: in production this would call Temporal API.
	// Returns a no-op command for now; real implementation wired in separately.
	return func() tea.Msg { return nil }
}

func (m *Model) handleHistoryKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		m.showHistory = false
		return m, nil
	case "t":
		m.historyTimelineReverse = !m.historyTimelineReverse
		return m, nil
	}
	return m, nil
}

func (m *Model) renderHistoryView() string {
	statusStr := ""
	if m.historyStatus != "" {
		statusStr = " [" + m.historyStatus + "]"
	}
	header := lipgloss.NewStyle().
		Width(m.width).
		Background(lipgloss.Color("#2A2A2A")).
		PaddingLeft(1).
		Render(fmt.Sprintf("Workflow History: %s%s", m.historyWfID, statusStr))

	contentLines := m.historyLines

	// Render timeline spans if present
	var spanLines []string
	if len(m.historySpans) > 0 {
		// compute total duration for proportional bars
		var total time.Duration
		for _, s := range m.historySpans {
			end := s.offset + s.duration
			if end > total {
				total = end
			}
		}
		if total == 0 {
			total = time.Second
		}
		spanLines = append(spanLines, "")
		spanLines = append(spanLines, pinkBoldStyle.Render("Activity Timeline:"))
		barWidth := m.width - 4
		if barWidth < 20 {
			barWidth = 20
		}
		for _, s := range m.historySpans {
			spanLines = append(spanLines, "  "+formatSpanBar(s, barWidth, total))
		}
	}

	var allLines []string
	if m.historyTimelineReverse {
		rev := make([]string, len(contentLines))
		copy(rev, contentLines)
		for i, j := 0, len(rev)-1; i < j; i, j = i+1, j-1 {
			rev[i], rev[j] = rev[j], rev[i]
		}
		allLines = append(allLines, rev...)
	} else {
		allLines = append(allLines, contentLines...)
	}
	allLines = append(allLines, spanLines...)

	bodyHeight := m.height - FooterHeight - HeaderHeight
	if bodyHeight < 1 {
		bodyHeight = 1
	}
	for len(allLines) < bodyHeight {
		allLines = append(allLines, "")
	}
	if len(allLines) > bodyHeight {
		allLines = allLines[:bodyHeight]
	}

	body := strings.Join(allLines, "\n")

	footer := dimStyle.Render("esc:close  t:toggle order")
	footer = lipgloss.NewStyle().Width(m.width).Background(lipgloss.Color("#2A2A2A")).Render(footer)

	return lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
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
