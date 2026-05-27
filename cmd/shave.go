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

The workflow claims the yak, runs Pi via LiteLLM to implement it, opens a draft PR,
then waits for a human to review, approve, and merge. When merged, the yak is closed.

Requires LITELLM_BASE_URL and LITELLM_API_KEY in the worker environment (via op run).

The yak name can be given as space-separated words or hyphenated:
  yyx shave update documentation
  yyx shave update-documentation-to-reflect-current-architecture-7yf2`,
	Args: cobra.MinimumNArgs(1),
	Run:  runShave,
}

func init() {
	rootCmd.AddCommand(shaveCmd)
	shaveCmd.Flags().String("repo-root", "", "Path to repo root (defaults to current directory)")
	shaveCmd.Flags().String("pi-model", "", "LiteLLM model name (uses gateway default if unset)")
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

	model, _ := cmd.Flags().GetString("pi-model")
	tools, _ := cmd.Flags().GetStringSlice("pi-tools")
	skills, _ := cmd.Flags().GetStringArray("pi-skill")

	cfg := temporal.PiConfig{
		Model:  model,
		Tools:  tools,
		Skills: skills,
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
	if model != "" {
		fmt.Printf("  Model:       %s\n", model)
	}
}
