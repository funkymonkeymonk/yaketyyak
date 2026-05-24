package cmd

import (
	"context"
	"fmt"
	"log"

	"github.com/funkymonkeymonk/yaketyyak/temporal"
	"github.com/spf13/cobra"
)

var prFeedbackCmd = &cobra.Command{
	Use:   "pr-feedback",
	Short: "Send PR review feedback",
	Run: func(cmd *cobra.Command, args []string) {
		prNumber, _ := cmd.Flags().GetInt("pr-number")
		comment, _ := cmd.Flags().GetString("comment")
		author, _ := cmd.Flags().GetString("author")

		c := temporal.NewClient()
		defer c.Close()

		ctx := context.Background()
		err := c.SignalWorkflow(ctx, temporal.WorkflowID, "", "pr_feedback", []interface{}{prNumber, comment, author})
		if err != nil {
			log.Fatalf("Failed to send PR feedback: %v", err)
		}

		fmt.Printf("Sent PR feedback for #%d\n", prNumber)
	},
}

func init() {
	prFeedbackCmd.Flags().Int("pr-number", 0, "Pull request number")
	prFeedbackCmd.Flags().String("comment", "", "Review body")
	prFeedbackCmd.Flags().String("author", "reviewer", "Reviewer handle")
	prFeedbackCmd.MarkFlagRequired("pr-number")
	prFeedbackCmd.MarkFlagRequired("comment")
}
