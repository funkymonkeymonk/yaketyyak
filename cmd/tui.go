package cmd

import (
	"github.com/spf13/cobra"
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Launch the yyx terminal UI",
	Long: `Launch an interactive terminal UI for browsing yaks and managing workflows.

Scans the current directory and below for git repos with .yaks/ directories.
Keyboard-driven: j/k navigate, single keys for actions.`,
	Run: runTUI,
}
