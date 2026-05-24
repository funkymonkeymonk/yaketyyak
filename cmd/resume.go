package cmd

import (
	"context"
	"fmt"
	"log"

	"github.com/funkymonkeymonk/yaketyyak/temporal"
	"github.com/spf13/cobra"
)

var resumeCmd = &cobra.Command{
	Use:   "resume",
	Short: "Resume the workflow",
	Run: func(cmd *cobra.Command, args []string) {
		c := temporal.NewClient()
		defer c.Close()

		ctx := context.Background()
		err := c.SignalWorkflow(ctx, temporal.WorkflowID, "", "resume", nil)
		if err != nil {
			log.Fatalf("Failed to resume workflow: %v", err)
		}

		fmt.Println("Resumed workflow")
	},
}
