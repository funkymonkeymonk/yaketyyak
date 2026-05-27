package cmd

import (
	"log"
	"os"

	"github.com/charmbracelet/bubbletea"
	"github.com/funkymonkeymonk/yaketyyak/tui"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "yyx",
	Short: "yaketyyak — autonomous yak shaving",
	Long: `yyx — manage and monitor autonomous yak-shaving workflows.

Browse yaks with the TUI, start shave workflows with 'yyx shave <yak>',
and watch them run to completion.`,
	Run: runTUI,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runTUI(cmd *cobra.Command, args []string) {
	cwd, _ := os.Getwd()
	repos, err := tui.ScanRepos(cwd)
	if err != nil {
		log.Fatalf("Failed to scan repos: %v", err)
	}

	m := tui.New(repos)
	p := tea.NewProgram(&m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		log.Fatalf("TUI error: %v", err)
	}
}
