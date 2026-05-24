package cmd

import (
	"context"
	"fmt"
	"log"

	"github.com/funkymonkeymonk/yaketyyak/temporal"
	"github.com/spf13/cobra"
)

var g2gScanCmd = &cobra.Command{
	Use:   "g2g-scan",
	Short: "Trigger a g2g scan signal",
	Run: func(cmd *cobra.Command, args []string) {
		c := temporal.NewClient()
		defer c.Close()

		ctx := context.Background()
		err := c.SignalWorkflow(ctx, temporal.WorkflowID, "", "g2g_signal", nil)
		if err != nil {
			log.Fatalf("Failed to send g2g scan signal: %v", err)
		}

		fmt.Println("Sent g2g scan signal")
	},
}
