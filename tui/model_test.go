package tui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"

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

var _ = fmt.Sprintf
