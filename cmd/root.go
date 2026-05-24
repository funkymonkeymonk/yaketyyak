package cmd

import (
	"log"
	"os"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/funkymonkeymonk/yaketyyak/temporal"
	"github.com/funkymonkeymonk/yaketyyak/tui"
	"github.com/spf13/cobra"
)

var style = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF69B4"))

var rootCmd = &cobra.Command{
	Use:   "yyx",
	Short: "yaketyyak Temporal CLI",
	Long: style.Render(`✦ yaketyyak — CI-triggered autonomous yak shaving

Send signals and start barber/shave workflows on Temporal.`) + "\n\n" +
		`Use subcommands to start workflows, send CI/g2g signals, provide PR feedback, and check status.`,
	Run: runTUI,
}

func Execute() {
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(ciSignalCmd)
	rootCmd.AddCommand(g2gScanCmd)
	rootCmd.AddCommand(prFeedbackCmd)
	rootCmd.AddCommand(pauseCmd)
	rootCmd.AddCommand(resumeCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(workerCmd)
	rootCmd.AddCommand(tuiCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runTUI(cmd *cobra.Command, args []string) {
	baseURL, _ := cmd.Flags().GetString("llm-base-url")
	model, _ := cmd.Flags().GetString("llm-model")

	cfg := temporal.LLMConfig{
		BaseURL: baseURL,
		Model:   model,
		APIKey:  llmAPIKey(),
	}

	cwd, _ := os.Getwd()
	repos, err := tui.ScanRepos(cwd)
	if err != nil {
		log.Fatalf("Failed to scan repos: %v", err)
	}

	m := tui.New(repos, cfg)
	p := tea.NewProgram(&m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		log.Fatalf("TUI error: %v", err)
	}
}

func init() {
	rootCmd.PersistentFlags().String("llm-base-url", "http://localhost:11434/v1",
		"OpenAI-compatible base URL (ollama, bifrost, etc.)")
	rootCmd.PersistentFlags().String("llm-model", "llama3.2", "LLM model name")
}
