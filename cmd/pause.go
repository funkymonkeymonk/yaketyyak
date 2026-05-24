package cmd

import (
	"context"
	"fmt"
	"log"

	"github.com/funkymonkeymonk/yaketyyak/temporal"
	"github.com/spf13/cobra"
)

var pauseCmd = &cobra.Command{
	Use:   "pause",
	Short: "Pause the workflow",
	Run: func(cmd *cobra.Command, args []string) {
		c := temporal.NewClient()
		defer c.Close()

		ctx := context.Background()
		err := c.SignalWorkflow(ctx, temporal.WorkflowID, "", "pause", nil)
		if err != nil {
			log.Fatalf("Failed to pause workflow: %v", err)
		}

		fmt.Println("Paused workflow")
	},
}
