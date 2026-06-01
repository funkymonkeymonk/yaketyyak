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

The workflow claims the yak, clones the repo, runs Pi via LiteLLM to implement
it, opens a draft PR, then waits for a human to review, approve, and merge.

Requires LITELLM_BASE_URL, LITELLM_API_KEY, and GITHUB_TOKEN in the worker
environment (injected via op run from .env.op).

  yy shave update documentation --repo-url https://github.com/you/repo
  yy shave update-documentation-to-reflect-current-architecture-7yf2 --repo-url https://github.com/funkymonkeymonk/yaketyyak`,
	Args: cobra.MinimumNArgs(1),
	Run:  runShave,
}

func init() {
	rootCmd.AddCommand(shaveCmd)
	shaveCmd.Flags().String("repo-url", "", "GitHub repo URL (required, e.g. https://github.com/owner/repo)")
	shaveCmd.Flags().String("pi-model", "", "LiteLLM model name (uses gateway default if unset)")
	shaveCmd.Flags().StringSlice("pi-tools", temporal.DefaultPiTools, "Comma-separated Pi tools to enable")
	shaveCmd.Flags().StringArray("pi-skill", nil, "Pi skill file paths to load (repeatable)")
	_ = shaveCmd.MarkFlagRequired("repo-url")
}

func runShave(cmd *cobra.Command, args []string) {
	yakName := strings.Join(args, " ")

	repoURL, _ := cmd.Flags().GetString("repo-url")
	model, _ := cmd.Flags().GetString("pi-model")
	tools, _ := cmd.Flags().GetStringSlice("pi-tools")
	skills, _ := cmd.Flags().GetStringArray("pi-skill")

	cfg := temporal.WorkflowConfig{
		RepoURL: repoURL,
		Pi: temporal.PiConfig{
			Model:  model,
			Tools:  tools,
			Skills: skills,
		},
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

	run, err := c.ExecuteWorkflow(cmd.Context(), opts, "YakWorkflow", yakName, cfg)
	if err != nil {
		log.Fatalf("Failed to start YakWorkflow: %v", err)
	}

	fmt.Printf("Started YakWorkflow for %q\n", yakName)
	fmt.Printf("  Workflow ID: %s\n", workflowID)
	fmt.Printf("  Run ID:      %s\n", run.GetRunID())
	fmt.Printf("  Repo URL:    %s\n", repoURL)
	if model != "" {
		fmt.Printf("  Model:       %s\n", model)
	}
	if repoURLFromEnv := os.Getenv("GITHUB_TOKEN"); repoURLFromEnv == "" {
		fmt.Println("  Warning: GITHUB_TOKEN not set locally (ok if running via devenv up)")
	}
}
