package tui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"

	"go.temporal.io/sdk/client"

	"github.com/funkymonkeymonk/yaketyyak/temporal"
)

type temporalMsg struct {
	repoIdx int
	state   *temporal.WorkflowState
	err     error
}

type temporalAllMsg struct {
	states []*temporal.WorkflowState
	errs   []error
}

type signalMsg struct {
	action string
	err    error
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

func (m *Model) Init() tea.Cmd {
	m.buildTree()
	m.listView.SetItems(m.treeLines)
	m.prActivity.SetItems(m.prLines)
	m.syncProgramContext()
	m.updateSidebar()
	return tea.Batch(m.queryAllTemporal())
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

func (m *Model) renderConfirmContent(mainContent, statusView string) string {
	overlay := lipgloss.JoinVertical(lipgloss.Left,
		m.confirmMsg,
		"",
		dimStyle.Render(" [y] confirm  [n/esc] cancel"),
	)
	overlayW := 60
	overlayStyled := m.ctx.Styles.Common.OverlayStyle.Copy().Width(overlayW).Render(overlay)

	fullContent := lipgloss.JoinVertical(lipgloss.Left, mainContent, statusView)
	return lipgloss.Place(
		m.width, m.ctx.MainContentHeight+FooterHeight,
		lipgloss.Center, lipgloss.Center,
		overlayStyled,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.NoColor{}),
		lipgloss.WithWhitespaceBackground(lipgloss.NoColor{}),
	) + fullContent
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
		m.sidePanel.ShowYak(&repo.Yaks[tl.yakIdx], repo.WFState, w)
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
	repo := m.repos[tl.repoIdx]
	yakName := repo.Yaks[tl.yakIdx].Name
	m.statusMsg = fmt.Sprintf("Shaving yak: %s...", yakName)
	m.statusTimer = 3
	return m, tea.Batch(
		func() tea.Msg {
			cl := temporal.NewClient()
			defer cl.Close()
			wfID := temporal.ShaveWorkflowID(yakName)
			_, err := cl.ExecuteWorkflow(context.Background(), client.StartWorkflowOptions{
				ID:        wfID,
				TaskQueue: temporal.TaskQueue,
			}, "ShaveWorkflow", yakName, m.llmCfg, repo.Remote, repo.Root, 3)
			return signalMsg{action: fmt.Sprintf("shave %s", yakName), err: err}
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
			cl := temporal.NewClient()
			defer cl.Close()
			_, err := cl.ExecuteWorkflow(context.Background(), client.StartWorkflowOptions{
				ID:        wfID,
				TaskQueue: temporal.TaskQueue,
			}, "BarberWorkflow", repo.Remote, repo.Root, m.llmCfg, false, 60)
			return signalMsg{action: "start", err: err}
		}
	case "pause":
		return m, func() tea.Msg {
			cl := temporal.NewClient()
			defer cl.Close()
			err := cl.SignalWorkflow(context.Background(), wfID, "", "pause", nil)
			return signalMsg{action: "pause", err: err}
		}
	case "resume":
		return m, func() tea.Msg {
			cl := temporal.NewClient()
			defer cl.Close()
			err := cl.SignalWorkflow(context.Background(), wfID, "", "resume", nil)
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
			cl := temporal.NewClient()
			defer cl.Close()
			err := cl.SignalWorkflow(context.Background(), wfID, "", "g2g_signal", nil)
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
			cl := temporal.NewClient()
			defer cl.Close()
			payload := []interface{}{"success", "main", "", ""}
			err := cl.SignalWorkflow(context.Background(), wfID, "", "ci_signal", payload)
			return signalMsg{action: "ci_signal", err: err}
		},
		tickStatus(),
	)
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
		cl := temporal.NewClient()
		defer cl.Close()

		states := make([]*temporal.WorkflowState, len(m.repos))
		errs := make([]error, len(m.repos))

		for i, repo := range m.repos {
			value, err := cl.QueryWorkflow(context.Background(), repo.WFID, "", "status")
			if err != nil {
				if !isWorkflowNotFound(err) {
					errs[i] = err
				}
				continue
			}
			var state temporal.WorkflowState
			if err := value.Get(&state); err != nil {
				errs[i] = err
				continue
			}
			states[i] = &state
		}
		return temporalAllMsg{states: states, errs: errs}
	}
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
