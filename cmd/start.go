package cmd

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/funkymonkeymonk/yaketyyak/temporal"
	"github.com/spf13/cobra"
	"go.temporal.io/sdk/client"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the barber workflow",
	Long: `Start the barber workflow that continuously monitors yaks and processes tasks.

The barber syncs yaks, triages actionable items, dispatches LLM calls, and merges PRs — all durably on Temporal.`,
	Run: func(cmd *cobra.Command, args []string) {
		repo, _ := cmd.Flags().GetString("repo")
		repoRoot, _ := cmd.Flags().GetString("repo-root")
		baseURL, _ := cmd.Flags().GetString("llm-base-url")
		model, _ := cmd.Flags().GetString("llm-model")
		g2gMode, _ := cmd.Flags().GetBool("g2g-mode")
		g2gInterval, _ := cmd.Flags().GetInt("g2g-scan-interval")

		cfg := temporal.LLMConfig{
			BaseURL: baseURL,
			Model:   model,
			APIKey:  llmAPIKey(),
		}

		c := temporal.NewClient()
		defer c.Close()

		ctx := context.Background()
		handle, err := c.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
			ID:        temporal.WorkflowID,
			TaskQueue: temporal.TaskQueue,
		}, "BarberWorkflow", repo, repoRoot, cfg, g2gMode, g2gInterval)
		if err != nil {
			log.Fatalf("Failed to start workflow: %v", err)
		}

		infoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00")).Bold(true)
		fmt.Println(infoStyle.Render("✦ Started workflow:"), handle.GetID())
		if g2gMode {
			fmt.Printf("  g2g mode: ON (scan interval: %dm)\n", g2gInterval)
		}
	},
}

func init() {
	startCmd.Flags().String("repo", "", "GitHub owner/repo")
	startCmd.Flags().String("repo-root", "", "Absolute path to local checkout")
	startCmd.Flags().String("llm-base-url", "http://localhost:11434/v1", "OpenAI-compatible base URL (ollama, bifrost, etc.)")
	startCmd.Flags().String("llm-model", "llama3.2", "LLM model name")
	startCmd.Flags().Bool("g2g-mode", false, "Only process yaks tagged @g2g")
	startCmd.Flags().Int("g2g-scan-interval", 60, "Minutes between periodic g2g scans")
	startCmd.MarkFlagRequired("repo")
	startCmd.MarkFlagRequired("repo-root")
}

func llmAPIKey() string {
	if k := os.Getenv("YYX_LLM_API_KEY"); k != "" {
		return k
	}
	return os.Getenv("LLM_API_KEY")
}
