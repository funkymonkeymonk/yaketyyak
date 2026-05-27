package cmd

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/funkymonkeymonk/yaketyyak/temporal"
	"github.com/spf13/cobra"
	"go.temporal.io/sdk/client"
)

var shaveCmd = &cobra.Command{
	Use:   "shave <yak-name>",
	Short: "Start a YakWorkflow to shave a yak",
	Long: `Start a long-running YakWorkflow for the named yak.

The workflow claims the yak, runs Pi to implement it, opens a draft PR,
then waits for a human to review, approve, and merge. When merged, the
yak is automatically closed.

The yak name can be given as space-separated words or hyphenated:
  yyx shave build the shave workflow
  yyx shave build-the-shave-workflow-drkq`,
	Args: cobra.MinimumNArgs(1),
	Run:  runShave,
}

func init() {
	rootCmd.AddCommand(shaveCmd)
	shaveCmd.Flags().String("repo-root", "", "Path to repo root (defaults to current directory)")
	shaveCmd.Flags().String("pi-provider", "openai", "Pi LLM provider (openai, anthropic, google, ...)")
	shaveCmd.Flags().String("pi-model", "", "Pi model (e.g. sonnet, gpt-4o); uses provider default if unset")
	shaveCmd.Flags().StringSlice("pi-tools", temporal.DefaultPiTools, "Comma-separated Pi tools to enable")
	shaveCmd.Flags().StringArray("pi-skill", nil, "Pi skill file paths to load (repeatable)")
}

func runShave(cmd *cobra.Command, args []string) {
	yakName := strings.Join(args, " ")

	repoRoot, _ := cmd.Flags().GetString("repo-root")
	if repoRoot == "" {
		var err error
		repoRoot, err = os.Getwd()
		if err != nil {
			log.Fatalf("Failed to get working directory: %v", err)
		}
	}

	provider, _ := cmd.Flags().GetString("pi-provider")
	model, _ := cmd.Flags().GetString("pi-model")
	tools, _ := cmd.Flags().GetStringSlice("pi-tools")
	skills, _ := cmd.Flags().GetStringArray("pi-skill")

	cfg := temporal.PiConfig{
		Provider: provider,
		Model:    model,
		Tools:    tools,
		Skills:   skills,
		APIKey:   piAPIKey(provider),
	}

	c, err := temporal.NewClient()
	if err != nil {
		log.Fatalf("Failed to connect to Temporal: %v", err)
	}
	defer c.Close()

	workflowID := temporal.YakWorkflowID(yakName)

	opts := client.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: temporal.TaskQueue,
	}

	run, err := c.ExecuteWorkflow(cmd.Context(), opts, temporal.YakWorkflow, yakName, repoRoot, cfg)
	if err != nil {
		log.Fatalf("Failed to start YakWorkflow: %v", err)
	}

	fmt.Printf("Started YakWorkflow for %q\n", yakName)
	fmt.Printf("  Workflow ID: %s\n", workflowID)
	fmt.Printf("  Run ID:      %s\n", run.GetRunID())
	fmt.Printf("  Repo root:   %s\n", repoRoot)
	fmt.Printf("  Provider:    %s\n", provider)
	if model != "" {
		fmt.Printf("  Model:       %s\n", model)
	}
}

// piAPIKey reads the API key for the given provider from standard env vars.
func piAPIKey(provider string) string {
	switch strings.ToLower(provider) {
	case "anthropic":
		if k := os.Getenv("ANTHROPIC_API_KEY"); k != "" {
			return k
		}
	case "openai":
		if k := os.Getenv("OPENAI_API_KEY"); k != "" {
			return k
		}
	case "google":
		if k := os.Getenv("GEMINI_API_KEY"); k != "" {
			return k
		}
	case "openrouter":
		if k := os.Getenv("OPENROUTER_API_KEY"); k != "" {
			return k
		}
	}
	// Fallback: allow callers to use a single unified env var.
	return os.Getenv("PI_API_KEY")
}
