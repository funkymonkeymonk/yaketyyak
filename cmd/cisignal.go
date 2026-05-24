package cmd

import (
	"context"
	"fmt"
	"log"

	"github.com/funkymonkeymonk/yaketyyak/temporal"
	"github.com/spf13/cobra"
)

var ciSignalCmd = &cobra.Command{
	Use:   "ci",
	Short: "Send a CI signal",
	Long: `Send a CI pipeline completion signal to the workflow.

Triggers a yak shaving turn: sync yaks, triage, implement, PR, watch CI, merge.`,
	Run: func(cmd *cobra.Command, args []string) {
		conclusion, _ := cmd.Flags().GetString("conclusion")
		branch, _ := cmd.Flags().GetString("branch")
		sha, _ := cmd.Flags().GetString("sha")
		details, _ := cmd.Flags().GetString("details")

		c := temporal.NewClient()
		defer c.Close()

		ctx := context.Background()
		err := c.SignalWorkflow(ctx, temporal.WorkflowID, "", "ci_signal", []interface{}{conclusion, branch, sha, details})
		if err != nil {
			log.Fatalf("Failed to send CI signal: %v", err)
		}

		fmt.Printf("Sent CI signal: %s on %s\n", conclusion, branch)
	},
}

func init() {
	ciSignalCmd.Flags().String("conclusion", "", "CI conclusion (success, failure, cancelled)")
	ciSignalCmd.Flags().String("branch", "", "Git branch")
	ciSignalCmd.Flags().String("sha", "", "Commit SHA")
	ciSignalCmd.Flags().String("details", "", "Optional JSON with run IDs, check URLs")
	ciSignalCmd.MarkFlagRequired("conclusion")
	ciSignalCmd.MarkFlagRequired("branch")
	ciSignalCmd.MarkFlagRequired("sha")
}
