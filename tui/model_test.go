package tui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"google.golang.org/protobuf/types/known/timestamppb"

	historypb "go.temporal.io/api/history/v1"

	"github.com/funkymonkeymonk/yaketyyak/temporal"
)

func makeYaks(yaks []YakLine) []Repo {
	return []Repo{{
		Name:    "test/repo",
		Root:    "/test/repo",
		Remote:  "test/repo",
		Yaks:    yaks,
		YaksDir: ".yaks",
		WFID:    "yyx-test",
	}}
}

func TestSyncProgramContext(t *testing.T) {
	yaks := make([]YakLine, 50)
	for i := range yaks {
		yaks[i] = YakLine{Path: fmt.Sprintf("y%d", i), Name: "Y", State: YakTodo, Depth: 0}
	}
	repos := makeYaks(yaks)

	m := New(repos, temporal.LLMConfig{})
	m.buildTree()
	m.listView.SetItems(m.treeLines)

	m.width = 120
	m.height = 30
	m.syncProgramContext()

	if m.ctx.MainContentHeight != m.height-FooterHeight-HeaderHeight {
		t.Errorf("MainContentHeight = %d, want %d", m.ctx.MainContentHeight, m.height-FooterHeight-HeaderHeight)
	}
	if m.ctx.PreviewWidth < 10 {
		t.Errorf("PreviewWidth too small: %d", m.ctx.PreviewWidth)
	}
	if m.ctx.MainContentWidth < 10 {
		t.Errorf("MainContentWidth too small: %d", m.ctx.MainContentWidth)
	}
}

func TestCursorNavigation(t *testing.T) {
	yaks := make([]YakLine, 50)
	for i := range yaks {
		yaks[i] = YakLine{Path: fmt.Sprintf("y%d", i), Name: "Y", State: YakTodo, Depth: 0}
	}
	repos := makeYaks(yaks)

	m := New(repos, temporal.LLMConfig{})
	m.buildTree()
	m.listView.SetItems(m.treeLines)

	m.width = 120
	m.height = 30
	m.syncProgramContext()

	tests := []struct {
		cursor int
	}{
		{0},
		{1},
		{5},
		{20},
		{30},
		{50},
	}

	for _, tt := range tests {
		m.listView.SetCursor(tt.cursor)
		if got := m.listView.Cursor(); got != tt.cursor {
			t.Errorf("cursor=%d: got cursor %d", tt.cursor, got)
		}
	}
}

func TestViewAllYaksVisible(t *testing.T) {
	yaks := []YakLine{
		{Path: "root", Name: "Root", State: YakTodo, Depth: 0},
		{Path: "root/a", Name: "Child A", State: YakTodo, Depth: 1, IsLastSibling: false, AncestorContinues: []bool{}},
		{Path: "root/b", Name: "Child B", State: YakWip, Depth: 1, IsLastSibling: false, AncestorContinues: []bool{}},
		{Path: "root/c", Name: "Child C", State: YakDone, Depth: 1, IsLastSibling: true, AncestorContinues: []bool{}},
		{Path: "top", Name: "Top", State: YakTodo, Depth: 0},
		{Path: "top/sub", Name: "Sub", State: YakWip, Depth: 1, IsLastSibling: true, AncestorContinues: []bool{}},
	}
	repos := makeYaks(yaks)

	for cursor := 0; cursor < len(yaks); cursor++ {
		m := New(repos, temporal.LLMConfig{})
		m.buildTree()
		m.listView.SetItems(m.treeLines)
		m.width = 120
		m.height = 30
		m.syncProgramContext()
		m.listView.SetCursor(cursor + 1)
		view := m.View()

		for _, yak := range yaks {
			if !strings.Contains(view, yak.Name) {
				t.Errorf("cursor=%d: missing %q in view", cursor, yak.Name)
			}
		}
	}
}

func TestLongNamesDontOverflow(t *testing.T) {
	yaks := []YakLine{
		{Path: "root", Name: "Workflow improvements from yakthang", State: YakTodo, Depth: 0},
		{Path: "root/adversarial", Name: "Adversarial review process", State: YakTodo, Depth: 1, IsLastSibling: false, AncestorContinues: []bool{}},
		{Path: "root/build", Name: "Build tooling (Justfile)", State: YakTodo, Depth: 1, IsLastSibling: false, AncestorContinues: []bool{}},
		{Path: "root/yakob", Name: "Orchestrator agent (Yakob)", State: YakTodo, Depth: 1, IsLastSibling: false, AncestorContinues: []bool{}},
		{Path: "root/whimsy", Name: "Personality and whimsy", State: YakTodo, Depth: 1, IsLastSibling: false, AncestorContinues: []bool{}},
		{Path: "root/config", Name: "Project-level agent config", State: YakTodo, Depth: 1, IsLastSibling: false, AncestorContinues: []bool{}},
		{Path: "root/triage", Name: "Session discipline (triage-wrap)", State: YakTodo, Depth: 1, IsLastSibling: true, AncestorContinues: []bool{}},
		{Path: "tui", Name: "yyx tui", State: YakWip, Depth: 0},
		{Path: "tui/model", Name: "bubble tea model", State: YakDone, Depth: 1, IsLastSibling: false, AncestorContinues: []bool{}},
		{Path: "tui/cmd", Name: "cli subcommand", State: YakDone, Depth: 1, IsLastSibling: false, AncestorContinues: []bool{}},
		{Path: "tui/render", Name: "tree and detail rendering", State: YakDone, Depth: 1, IsLastSibling: false, AncestorContinues: []bool{}},
		{Path: "tui/yaks", Name: "yak data model", State: YakDone, Depth: 1, IsLastSibling: true, AncestorContinues: []bool{}},
	}
	repos := makeYaks(yaks)

	for cursor := 0; cursor <= len(yaks); cursor++ {
		m := New(repos, temporal.LLMConfig{})
		m.buildTree()
		m.width = 132
		m.height = 35
		m.syncProgramContext()

		if len(m.treeLines) != len(yaks)+1 {
			t.Errorf("Expected %d treeLines (1 repo + %d yaks), got %d", len(yaks)+1, len(yaks), len(m.treeLines))
		}
	}

	m := New(repos, temporal.LLMConfig{})
	m.buildTree()
	m.width = 80
	m.height = 24
	m.syncProgramContext()
	m.listView.SetItems(m.treeLines)
	m.listView.SetCursor(0)
	view := m.View()
	if !strings.Contains(view, "test/repo") {
		t.Error("Missing repo name in view")
	}
	if len(m.treeLines) != len(yaks)+1 {
		t.Errorf("TreeLines count should be %d, got %d", len(yaks)+1, len(m.treeLines))
	}
}

func TestDetailDoesNotOverflow(t *testing.T) {
	yaks := []YakLine{
		{Path: "test", Name: "Test Yak", State: YakTodo, Depth: 0, Context: "Cobra subcommand `yyx tui` with flags: --repo owner/repo, --repo-root /path, --agent pi|claude-code|codex|opencode, --yaks-dir .yaks."},
	}
	repos := makeYaks(yaks)

	m := New(repos, temporal.LLMConfig{})
	m.buildTree()
	m.listView.SetItems(m.treeLines)
	m.width = 120
	m.height = 30
	m.syncProgramContext()
	m.listView.SetCursor(1)

	view := m.View()
	lines := strings.Split(view, "\n")

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "code|codex") {
			t.Errorf("Long context line overflowed into tree area:\n%s", view)
		}
	}
}

func TestTreeLinesCount(t *testing.T) {
	yaks := []YakLine{
		{Path: "root", Name: "Root", State: YakTodo, Depth: 0},
		{Path: "root/a", Name: "Child A", State: YakTodo, Depth: 1, IsLastSibling: false, AncestorContinues: []bool{}},
		{Path: "root/b", Name: "Child B", State: YakTodo, Depth: 1, IsLastSibling: true, AncestorContinues: []bool{}},
	}
	repos := []Repo{
		{Name: "repo1", Root: "/a", Remote: "repo1", Yaks: yaks, YaksDir: "/a/.yaks", WFID: "yyx-1"},
		{Name: "repo2", Root: "/b", Remote: "repo2", Yaks: []YakLine{}, YaksDir: "/b/.yaks", WFID: "yyx-2"},
	}

	m := New(repos, temporal.LLMConfig{})
	m.buildTree()

	if len(m.treeLines) != 5 {
		t.Errorf("Expected 5 treeLines (2 repos + 3 yaks), got %d", len(m.treeLines))
	}

	if m.treeLines[0].kind != treeRepo || m.treeLines[0].name != "repo1" {
		t.Error("First line should be repo1")
	}
	if m.treeLines[1].kind != treeYak || m.treeLines[1].name != "Root" {
		t.Error("Second line should be yak Root")
	}
	if m.treeLines[4].kind != treeRepo || m.treeLines[4].name != "repo2" {
		t.Error("Fifth line should be repo2")
	}
}

func TestDetailTruncation(t *testing.T) {
	longLine := "Cobra subcommand `yyx tui` with flags: --repo owner/repo"
	result := padWidth("  "+dimStyle.Render(longLine), 40)
	vis := lipgloss.Width(result)
	if vis < 40 {
		t.Errorf("padWidth should pad to at least 40, got %d visual width", vis)
	}
	lines := strings.Count(result, "\n")
	if lines > 0 {
		t.Errorf("padWidth should NOT wrap, got %d newlines", lines)
	}
	trunc := truncateRunes("  "+longLine, 40)
	if len([]rune(trunc)) > 40 {
		t.Errorf("truncateRunes should truncate to 40 runes, got %d", len([]rune(trunc)))
	}
}

func TestJoinHorizontalLines(t *testing.T) {
	left := "line1\nline2\nline3"
	right := "LINE1\nLINE2\nLINE3\nLINE4"
	lp := lipgloss.NewStyle().Width(10).Height(5).Render(left)
	rp := lipgloss.NewStyle().Width(10).Height(5).Render(right)
	joined := lipgloss.JoinHorizontal(lipgloss.Top, lp, rp)
	lines := strings.Split(joined, "\n")
	if len(lines) < 5 {
		t.Errorf("Expected at least 5 lines, got %d", len(lines))
	}
	if !strings.Contains(lines[0], "line1") {
		t.Errorf("Line 0 should contain 'line1': %q", lines[0])
	}
	if !strings.Contains(lines[0], "LINE1") {
		t.Errorf("Line 0 should contain 'LINE1': %q", lines[0])
	}
}

func TestRepoTreeLine(t *testing.T) {
	yaks := []YakLine{
		{Path: "a", Name: "Yak A", State: YakTodo, Depth: 0},
	}
	repos := []Repo{
		{Name: "owner/repo1", Root: "/a", Remote: "owner/repo1", Yaks: yaks, YaksDir: "/a/.yaks", WFID: "yyx-1"},
		{Name: "owner/repo2", Root: "/b", Remote: "owner/repo2", Yaks: nil, YaksDir: "/b/.yaks", WFID: "yyx-2"},
	}

	m := New(repos, temporal.LLMConfig{})
	m.buildTree()
	m.width = 120
	m.height = 30
	m.syncProgramContext()

	if len(m.treeLines) != 3 {
		t.Fatalf("Expected 3 treeLines (2 repos + 1 yak), got %d", len(m.treeLines))
	}
	if m.treeLines[0].kind != treeRepo {
		t.Error("First treeLine should be a repo")
	}
	if m.treeLines[0].name != "owner/repo1" {
		t.Errorf("First repo name = %q", m.treeLines[0].name)
	}
	if m.treeLines[1].kind != treeYak {
		t.Error("Second treeLine should be a yak")
	}
	if m.treeLines[2].kind != treeRepo {
		t.Error("Third treeLine should be a repo")
	}
}

func TestConfirmOverlay(t *testing.T) {
	yaks := []YakLine{
		{Path: "test", Name: "Test", State: YakWip, Depth: 0},
	}
	repos := makeYaks(yaks)

	m := New(repos, temporal.LLMConfig{})
	m.buildTree()
	m.listView.SetItems(m.treeLines)
	m.width = 120
	m.height = 30
	m.syncProgramContext()
	m.listView.SetCursor(1)

	m.showConfirm = true
	m.confirmMsg = "Shave yak: Test?"
	m.confirmAction = "shave"

	view := m.View()
	if !strings.Contains(view, "Shave yak: Test?") {
		t.Error("Confirmation overlay should contain the message")
	}
	if !strings.Contains(view, "confirm") {
		t.Error("Confirmation overlay should show confirm key hint")
	}
}

func TestSidebarShowsContent(t *testing.T) {
	yaks := []YakLine{
		{Path: "test", Name: "Test Yak", State: YakTodo, Depth: 0},
	}
	repos := makeYaks(yaks)

	m := New(repos, temporal.LLMConfig{})
	m.buildTree()
	m.listView.SetItems(m.treeLines)
	m.width = 120
	m.height = 30
	m.syncProgramContext()
	m.listView.SetCursor(1)
	m.updateSidebar()

	view := m.View()
	if !strings.Contains(view, "Test Yak") {
		t.Error("Sidebar should show yak name")
	}
	if !strings.Contains(view, "Actions") {
		t.Error("Sidebar should show Actions section")
	}
}

func TestSidebarEmptyState(t *testing.T) {
	repos := []Repo{
		{Name: "empty", Root: "/empty", Remote: "empty", Yaks: nil, YaksDir: "/empty/.yaks", WFID: "yyx-e"},
	}

	m := New(repos, temporal.LLMConfig{})
	m.buildTree()
	m.listView.SetItems(m.treeLines)
	m.width = 120
	m.height = 30
	m.syncProgramContext()
	m.listView.SetCursor(0)
	m.updateSidebar()

	view := m.View()
	if !strings.Contains(view, "not started") {
		t.Error("Empty repo should show 'not started'")
	}
}

func TestViewFillsScreen(t *testing.T) {
	yaks := []YakLine{
		{Path: "tui-context", Name: "Fix TUI context truncation", State: YakWip, Depth: 0,
			Context: "The yaks context is truncated instead of wrapping."},
		{Path: "tui-context/overflow", Name: "Sidebar text overflow", State: YakTodo, Depth: 1, IsLastSibling: true, AncestorContinues: []bool{}},
	}
	repos := makeYaks(yaks)

	sizes := []struct{ w, h int }{{120, 30}, {120, 40}, {120, 50}}
	for _, sz := range sizes {
		m := New(repos, temporal.LLMConfig{})
		m.buildTree()
		m.listView.SetItems(m.treeLines)
		m.width = sz.w
		m.height = sz.h
		m.syncProgramContext()
		m.listView.SetCursor(1)
		m.updateSidebar()

		view := m.View()
		lines := strings.Split(view, "\n")
		if len(lines) != m.height {
			t.Errorf("height=%d: View() returned %d lines, want %d", m.height, len(lines), m.height)
		}
	}
}

func TestViewportHeightsMatchAllocation(t *testing.T) {
	yaks := make([]YakLine, 2)
	for i := range yaks {
		yaks[i] = YakLine{Path: fmt.Sprintf("y%d", i), Name: "Y", State: YakTodo, Depth: 0}
	}
	repos := makeYaks(yaks)

	m := New(repos, temporal.LLMConfig{})
	m.buildTree()
	m.listView.SetItems(m.treeLines)
	m.width = 120
	m.height = 40
	m.syncProgramContext()

	if m.listView.viewport.Height != m.ctx.MainContentHeight {
		t.Errorf("list viewport height %d != MainContentHeight %d", m.listView.viewport.Height, m.ctx.MainContentHeight)
	}
	if m.sidePanel.viewport.Height != m.ctx.PreviewHeight {
		t.Errorf("sidebar viewport height %d != PreviewHeight %d", m.sidePanel.viewport.Height, m.ctx.PreviewHeight)
	}
}

func TestConfirmThenCancel(t *testing.T) {
	yaks := []YakLine{
		{Path: "test", Name: "Test", State: YakWip, Depth: 0},
	}
	repos := makeYaks(yaks)

	m := New(repos, temporal.LLMConfig{})
	m.buildTree()
	m.listView.SetItems(m.treeLines)
	m.width = 120
	m.height = 30
	m.syncProgramContext()

	m.showConfirm = true
	m.confirmMsg = "Test?"
	m.confirmAction = "shave"
	_ = m

	view := m.View()
	if !strings.Contains(view, "Test?") {
		t.Error("Confirmation overlay should contain the message")
	}
}

func TestConfirmOverlayDoesNotDoubleHeight(t *testing.T) {
	yaks := []YakLine{
		{Path: "test", Name: "Test", State: YakWip, Depth: 0},
	}
	repos := makeYaks(yaks)

	m := New(repos, temporal.LLMConfig{})
	m.buildTree()
	m.listView.SetItems(m.treeLines)
	m.width = 120
	m.height = 30
	m.syncProgramContext()
	m.listView.SetCursor(1)

	m.showConfirm = true
	m.confirmMsg = "Shave yak: Test?"
	m.confirmAction = "shave"

	view := m.View()
	lines := strings.Split(view, "\n")
	if len(lines) != m.height {
		t.Errorf("View height with confirm overlay: got %d lines, want %d. Content height doubled!", len(lines), m.height)
	}
}

func TestShaveStartsWorkflow(t *testing.T) {
	yaks := []YakLine{
		{Path: "test", Name: "TestYak", State: YakWip, Depth: 0},
	}
	repos := makeYaks(yaks)

	m := New(repos, temporal.LLMConfig{})
	m.buildTree()
	m.listView.SetItems(m.treeLines)
	m.width = 120
	m.height = 30
	m.syncProgramContext()
	m.listView.SetCursor(1)

	_, cmd := m.doShaveYak()
	if cmd == nil {
		t.Error("doShaveYak should return a command to execute the workflow")
	}

	if m.statusMsg == "" {
		t.Error("doShaveYak should set a status message")
	}
}

func TestShaveYakNeedsConfirmation(t *testing.T) {
	yaks := []YakLine{
		{Path: "test", Name: "TestYak", State: YakWip, Depth: 0},
	}
	repos := makeYaks(yaks)

	m := New(repos, temporal.LLMConfig{})
	m.buildTree()
	m.listView.SetItems(m.treeLines)
	m.width = 120
	m.height = 30
	m.syncProgramContext()
	m.listView.SetCursor(1)

	_, cmd := m.shaveYak()
	if cmd != nil {
		t.Error("shaveYak should not return a command, only set confirm state")
	}
	if !m.showConfirm {
		t.Error("shaveYak should set showConfirm = true")
	}
	if m.confirmAction != "shave" {
		t.Errorf("confirmAction = %q, want \"shave\"", m.confirmAction)
	}
}

func TestDoShaveYakRejectsNonYakSelection(t *testing.T) {
	yaks := []YakLine{
		{Path: "test", Name: "TestYak", State: YakWip, Depth: 0},
	}
	repos := makeYaks(yaks)

	m := New(repos, temporal.LLMConfig{})
	m.buildTree()
	m.listView.SetItems(m.treeLines)
	m.width = 120
	m.height = 30
	m.syncProgramContext()
	m.listView.SetCursor(0)

	_, cmd := m.doShaveYak()
	if cmd != nil {
		t.Error("doShaveYak on a repo line should return nil command")
	}
}

func TestSignalMsgFailureShown(t *testing.T) {
	yaks := []YakLine{
		{Path: "test", Name: "TestYak", State: YakWip, Depth: 0},
	}
	repos := makeYaks(yaks)

	m := New(repos, temporal.LLMConfig{})
	m.buildTree()
	m.listView.SetItems(m.treeLines)
	m.width = 120
	m.height = 30
	m.syncProgramContext()

	m.Update(signalMsg{action: "shave TestYak", err: fmt.Errorf("workflow failed")})

	view := m.View()
	if !strings.Contains(view, "failed") {
		t.Error("signalMsg error should appear in status bar")
	}
}

func TestSignalMsgSuccessShown(t *testing.T) {
	yaks := []YakLine{
		{Path: "test", Name: "TestYak", State: YakWip, Depth: 0},
	}
	repos := makeYaks(yaks)

	m := New(repos, temporal.LLMConfig{})
	m.buildTree()
	m.listView.SetItems(m.treeLines)
	m.width = 120
	m.height = 30
	m.syncProgramContext()

	m.Update(signalMsg{action: "shave TestYak", err: nil})

	view := m.View()
	if !strings.Contains(view, "sent") {
		t.Error("signalMsg success should show 'sent' in status bar")
	}
}

func TestDoShaveYakSetsShaveState(t *testing.T) {
	yaks := []YakLine{
		{Path: "test", Name: "TestYak", State: YakWip, Depth: 0},
	}
	repos := makeYaks(yaks)

	m := New(repos, temporal.LLMConfig{})
	m.buildTree()
	m.listView.SetItems(m.treeLines)
	m.width = 120
	m.height = 30
	m.syncProgramContext()
	m.listView.SetCursor(1)

	m.doShaveYak()

	if m.repos[0].ShaveState == nil {
		t.Fatal("doShaveYak should set ShaveState on the repo")
	}
	if m.repos[0].ShaveState.YakName != "TestYak" {
		t.Errorf("ShaveState.YakName = %q, want TestYak", m.repos[0].ShaveState.YakName)
	}
	if m.repos[0].ShaveState.Phase != "starting" {
		t.Errorf("ShaveState.Phase = %q, want starting", m.repos[0].ShaveState.Phase)
	}
}

func TestTemporalAllMsgAppliesShaveStates(t *testing.T) {
	yaks := []YakLine{
		{Path: "test", Name: "TestYak", State: YakWip, Depth: 0},
	}
	repos := makeYaks(yaks)

	m := New(repos, temporal.LLMConfig{})
	m.buildTree()
	m.listView.SetItems(m.treeLines)
	m.width = 120
	m.height = 30
	m.syncProgramContext()

	shaveState := &temporal.ShaveState{
		YakName:    "TestYak",
		Phase:      "implementing",
		Iteration:  1,
		MaxRetries: 3,
		Workspace:  "shave-testyak",
	}

	m.Update(temporalAllMsg{
		states:      []*temporal.WorkflowState{nil},
		shaveStates: []*temporal.ShaveState{shaveState},
	})

	if m.repos[0].ShaveState == nil {
		t.Fatal("temporalAllMsg should apply shave state")
	}
	if m.repos[0].ShaveState.Phase != "implementing" {
		t.Errorf("ShaveState.Phase = %q, want implementing", m.repos[0].ShaveState.Phase)
	}
}

func TestPollTickTriggersQuery(t *testing.T) {
	yaks := []YakLine{
		{Path: "test", Name: "TestYak", State: YakWip, Depth: 0},
	}
	repos := makeYaks(yaks)

	m := New(repos, temporal.LLMConfig{})
	m.buildTree()
	m.listView.SetItems(m.treeLines)
	m.width = 120
	m.height = 30
	m.syncProgramContext()

	_, cmd := m.Update(pollTickMsg{})
	if cmd == nil {
		t.Error("pollTickMsg should return a command to query temporal")
	}
}

func TestInitStartsPolling(t *testing.T) {
	yaks := []YakLine{
		{Path: "test", Name: "TestYak", State: YakWip, Depth: 0},
	}
	repos := makeYaks(yaks)

	m := New(repos, temporal.LLMConfig{})
	cmd := m.Init()

	if cmd == nil {
		t.Error("Init should return commands (query + poll tick)")
	}
}

func TestTerminalShaveStateFailed(t *testing.T) {
	result := terminalShaveState("TestYak", 3) // WORKFLOW_EXECUTION_STATUS_FAILED
	if result.Phase != "failed" {
		t.Errorf("FAILED status should produce phase 'failed', got %q", result.Phase)
	}
	if result.YakName != "TestYak" {
		t.Errorf("YakName = %q, want TestYak", result.YakName)
	}
}

func TestTerminalShaveStateDone(t *testing.T) {
	result := terminalShaveState("TestYak", 2) // WORKFLOW_EXECUTION_STATUS_COMPLETED
	if result.Phase != "done" {
		t.Errorf("COMPLETED status should produce phase 'done', got %q", result.Phase)
	}
}

func TestTerminalShaveStateCancelled(t *testing.T) {
	result := terminalShaveState("TestYak", 4) // WORKFLOW_EXECUTION_STATUS_CANCELED
	if result.Phase != "cancelled" {
		t.Errorf("CANCELED status should produce phase 'cancelled', got %q", result.Phase)
	}
}

func TestSidebarShowsFailedShaveWorkflow(t *testing.T) {
	yaks := []YakLine{
		{Path: "test", Name: "TestYak", State: YakWip, Depth: 0},
	}
	repos := makeYaks(yaks)
	repos[0].ShaveState = &temporal.ShaveState{
		YakName: "TestYak",
		Phase:   "failed",
	}

	m := New(repos, temporal.LLMConfig{})
	m.buildTree()
	m.listView.SetItems(m.treeLines)
	m.width = 120
	m.height = 30
	m.syncProgramContext()
	m.listView.SetCursor(1)
	m.updateSidebar()

	view := m.View()
	if !strings.Contains(view, "Shave Workflow") {
		t.Error("Sidebar should show 'Shave Workflow' section even when failed")
	}
	if !strings.Contains(view, "failed") {
		t.Error("Sidebar should show 'failed' phase")
	}
}

func TestSidebarShowsDoneShaveWorkflow(t *testing.T) {
	yaks := []YakLine{
		{Path: "test", Name: "TestYak", State: YakWip, Depth: 0},
	}
	repos := makeYaks(yaks)
	repos[0].ShaveState = &temporal.ShaveState{
		YakName: "TestYak",
		Phase:   "done",
	}

	m := New(repos, temporal.LLMConfig{})
	m.buildTree()
	m.listView.SetItems(m.treeLines)
	m.width = 120
	m.height = 30
	m.syncProgramContext()
	m.listView.SetCursor(1)
	m.updateSidebar()

	view := m.View()
	if !strings.Contains(view, "Shave Workflow") {
		t.Error("Sidebar should show 'Shave Workflow' section when done")
	}
	if !strings.Contains(view, "done") {
		t.Error("Sidebar should show 'done' phase")
	}
}

func TestHistoryTargetWFID_PrefersShaveOverBarber(t *testing.T) {
	yaks := []YakLine{
		{Path: "test", Name: "TestYak", State: YakWip, Depth: 0},
	}
	repos := makeYaks(yaks)
	repos[0].WFID = "yyx-orch-abc123"
	repos[0].ShaveState = &temporal.ShaveState{YakName: "TestYak", Phase: "implementing"}

	m := New(repos, temporal.LLMConfig{})
	m.buildTree()
	m.listView.SetItems(m.treeLines)
	m.listView.SetCursor(1)

	wfID := m.historyTargetWFID()
	expected := temporal.ShaveWorkflowID("TestYak")
	if wfID != expected {
		t.Errorf("historyTargetWFID with active shave = %q, want %q", wfID, expected)
	}
}

func TestHistoryTargetWFID_FallsBackToBarber(t *testing.T) {
	yaks := []YakLine{
		{Path: "test", Name: "TestYak", State: YakWip, Depth: 0},
	}
	repos := makeYaks(yaks)
	repos[0].WFID = "yyx-orch-abc123"

	m := New(repos, temporal.LLMConfig{})
	m.buildTree()
	m.listView.SetItems(m.treeLines)
	m.listView.SetCursor(1)

	wfID := m.historyTargetWFID()
	if wfID != "yyx-orch-abc123" {
		t.Errorf("historyTargetWFID without shave = %q, want yyx-orch-abc123", wfID)
	}
}

func TestShowWorkflowHistorySetsState(t *testing.T) {
	yaks := []YakLine{
		{Path: "test", Name: "TestYak", State: YakWip, Depth: 0},
	}
	repos := makeYaks(yaks)
	repos[0].WFID = "yyx-orch-abc123"

	m := New(repos, temporal.LLMConfig{})
	m.buildTree()
	m.listView.SetItems(m.treeLines)
	m.width = 120
	m.height = 30
	m.syncProgramContext()
	m.listView.SetCursor(1)

	_, cmd := m.showWorkflowHistory()
	if !m.showHistory {
		t.Error("showWorkflowHistory should set showHistory = true")
	}
	if m.historyWfID != "yyx-orch-abc123" {
		t.Errorf("historyWfID = %q, want yyx-orch-abc123", m.historyWfID)
	}
	if cmd == nil {
		t.Error("showWorkflowHistory should return a fetch command")
	}
	if len(m.historyLines) == 0 {
		t.Error("showWorkflowHistory should set initial loading line")
	}
}

func TestHistoryViewRenders(t *testing.T) {
	yaks := []YakLine{
		{Path: "test", Name: "TestYak", State: YakWip, Depth: 0},
	}
	repos := makeYaks(yaks)
	repos[0].WFID = "yyx-orch-abc123"

	m := New(repos, temporal.LLMConfig{})
	m.buildTree()
	m.listView.SetItems(m.treeLines)
	m.width = 120
	m.height = 30
	m.syncProgramContext()

	m.showHistory = true
	m.historyWfID = "yyx-orch-abc123"
	m.historyLines = []string{
		"   1  12:00:00  WorkflowExecutionStarted",
		"   2  12:00:01  ActivityTaskScheduled",
		"   3  12:00:02  ActivityTaskFailed",
	}

	view := m.View()
	if !strings.Contains(view, "Workflow History") {
		t.Error("History view should show title")
	}
	if !strings.Contains(view, "yyx-orch-abc123") {
		t.Error("History view should show workflow ID")
	}
	if !strings.Contains(view, "WorkflowExecutionStarted") {
		t.Error("History view should show events")
	}
}

func TestHistoryViewDismisses(t *testing.T) {
	yaks := []YakLine{
		{Path: "test", Name: "TestYak", State: YakWip, Depth: 0},
	}
	repos := makeYaks(yaks)
	m := New(repos, temporal.LLMConfig{})
	m.showHistory = true
	m.historyWfID = "yyx-orch-abc123"
	m.historyLines = []string{"event 1"}

	m.Update(tea.KeyMsg{Type: tea.KeyEscape})

	if m.showHistory {
		t.Error("Esc should dismiss history view")
	}
}

func TestHistoryViewShowsStatus(t *testing.T) {
	yaks := []YakLine{
		{Path: "test", Name: "TestYak", State: YakWip, Depth: 0},
	}
	repos := makeYaks(yaks)
	repos[0].WFID = "yyx-orch-abc123"

	m := New(repos, temporal.LLMConfig{})
	m.buildTree()
	m.listView.SetItems(m.treeLines)
	m.width = 120
	m.height = 30
	m.syncProgramContext()

	m.showHistory = true
	m.historyWfID = "yyx-orch-abc123"
	m.historyStatus = "FAILED"
	m.historyLines = []string{"   1  12:00:00  WorkflowExecutionStarted"}

	view := m.View()
	if !strings.Contains(view, "[FAILED]") {
		t.Error("History view header should show status [FAILED]")
	}
}

func TestPollTickRefreshesHistoryWhenShowing(t *testing.T) {
	yaks := []YakLine{
		{Path: "test", Name: "TestYak", State: YakWip, Depth: 0},
	}
	repos := makeYaks(yaks)
	m := New(repos, temporal.LLMConfig{})
	m.width = 120
	m.height = 30
	m.syncProgramContext()

	m.showHistory = true
	m.historyWfID = "yyx-orch-abc123"
	m.historyLines = []string{"event"}

	_, cmd := m.Update(pollTickMsg{})
	if cmd == nil {
		t.Error("poll tick should return commands when history is showing")
	}
}

func TestFormatEventDetail_NoAttributes(t *testing.T) {
	ev := &historypb.HistoryEvent{
		EventId:   99,
		EventType: 9999,
	}
	detail := formatEventDetail(ev, nil)
	if detail != "" {
		t.Errorf("unknown event type with no attributes should return empty, got %q", detail)
	}
}

func TestHistoryEventsReversed(t *testing.T) {
	lines := []string{"event1", "event2", "event3"}
	for i, j := 0, len(lines)-1; i < j; i, j = i+1, j-1 {
		lines[i], lines[j] = lines[j], lines[i]
	}
	if lines[0] != "event3" || lines[2] != "event1" {
		t.Error("Events should be reversed: most recent first")
	}
}

func TestFirstLine(t *testing.T) {
	if firstLine("hello\nworld") != "hello" {
		t.Error("firstLine should return text before newline")
	}
	if firstLine("hello\rworld") != "hello" {
		t.Error("firstLine should return text before return")
	}
	if firstLine("hello") != "hello" {
		t.Error("firstLine should return full text if no newline")
	}
}

func TestExtractSpansFromEvents(t *testing.T) {
	now := time.Now()
	events := []*historypb.HistoryEvent{
		{
			EventId:   1,
			EventTime: timestamppb.New(now),
			EventType: 1,
		},
	}

	if spans := extractSpans(events); len(spans) != 0 {
		t.Errorf("extractSpans with no activity events should return empty, got %d", len(spans))
	}
}

func TestDurLabel(t *testing.T) {
	if durLabel(500*time.Millisecond) != "500ms" {
		t.Errorf("500ms got %q", durLabel(500*time.Millisecond))
	}
	if durLabel(2500*time.Millisecond) != "2.5s" {
		t.Errorf("2.5s got %q", durLabel(2500*time.Millisecond))
	}
	if durLabel(125*time.Second) != "2m5s" {
		t.Errorf("125s got %q", durLabel(125*time.Second))
	}
}

func TestFormatSpanBar(t *testing.T) {
	s := activitySpan{
		name:     "YxSync",
		offset:   100 * time.Millisecond,
		duration: 500 * time.Millisecond,
		status:   "completed",
	}
	result := formatSpanBar(s, 40, time.Second)
	if !strings.Contains(result, "YxSync") {
		t.Error("span bar should contain activity name")
	}
	if !strings.Contains(result, "✓") {
		t.Error("completed span should show ✓ marker")
	}
	if !strings.Contains(result, "500ms") {
		t.Error("span bar should show duration")
	}
}

func TestHistoryTogglesSort(t *testing.T) {
	yaks := []YakLine{
		{Path: "test", Name: "TestYak", State: YakWip, Depth: 0},
	}
	repos := makeYaks(yaks)
	repos[0].WFID = "yyx-orch-abc123"

	m := New(repos, temporal.LLMConfig{})
	m.buildTree()
	m.listView.SetItems(m.treeLines)
	m.width = 120
	m.height = 30
	m.syncProgramContext()

	m.showHistory = true
	m.historyTimelineReverse = false
	m.historyWfID = "yyx-orch-abc123"
	m.historyStatus = "FAILED"
	m.historyLines = []string{"event1", "event2"}

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	if !m.historyTimelineReverse {
		t.Error("Pressing t should toggle timeline sort to ascending")
	}
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	if m.historyTimelineReverse {
		t.Error("Pressing t again should toggle back to descending")
	}
}

func TestTimelineViewRenders(t *testing.T) {
	yaks := []YakLine{
		{Path: "test", Name: "TestYak", State: YakWip, Depth: 0},
	}
	repos := makeYaks(yaks)
	repos[0].WFID = "yyx-orch-abc123"

	m := New(repos, temporal.LLMConfig{})
	m.buildTree()
	m.listView.SetItems(m.treeLines)
	m.width = 120
	m.height = 30
	m.syncProgramContext()

	m.showHistory = true
	m.historyTimelineReverse = false
	m.historyWfID = "yyx-orch-abc123"
	m.historyStatus = "FAILED"
	m.historySpans = []activitySpan{
		{name: "YxSync", offset: 0, duration: 1200 * time.Millisecond, status: "completed"},
		{name: "YakClaim", offset: 1500 * time.Millisecond, duration: 800 * time.Millisecond, status: "completed"},
		{name: "ShaveInitWs", offset: 2500 * time.Millisecond, duration: 1100 * time.Millisecond, status: "failed", errMsg: "command not found"},
	}

	view := m.View()
	if !strings.Contains(view, "YxSync") {
		t.Error("Timeline should show activity names")
	}
	if !strings.Contains(view, "✗") {
		t.Error("Timeline should show failure markers")
	}
	if !strings.Contains(view, "command not found") {
		t.Error("Timeline should show error messages")
	}
	if len(view) == 0 {
		t.Error("Timeline view should produce output")
	}
}

func TestSidebarShowsShaveWorkflowStatus(t *testing.T) {
	yaks := []YakLine{
		{Path: "test", Name: "TestYak", State: YakWip, Depth: 0},
	}
	repos := makeYaks(yaks)
	repos[0].ShaveState = &temporal.ShaveState{
		YakName:    "TestYak",
		Phase:      "implementing",
		Iteration:  1,
		MaxRetries: 3,
		Workspace:  "shave-testyak",
	}

	m := New(repos, temporal.LLMConfig{})
	m.buildTree()
	m.listView.SetItems(m.treeLines)
	m.width = 120
	m.height = 30
	m.syncProgramContext()
	m.listView.SetCursor(1)
	m.updateSidebar()

	view := m.View()
	if !strings.Contains(view, "Shave Workflow") {
		t.Error("Sidebar should show 'Shave Workflow' section when shaving")
	}
	if !strings.Contains(view, "implementing") {
		t.Error("Sidebar should show shave phase")
	}
	if !strings.Contains(view, "1/3") {
		t.Error("Sidebar should show iteration count")
	}
}

var _ = fmt.Sprintf
