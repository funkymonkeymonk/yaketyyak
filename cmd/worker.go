package cmd

import (
	"github.com/funkymonkeymonk/yaketyyak/temporal"
	"github.com/spf13/cobra"
)

var workerCmd = &cobra.Command{
	Use:   "worker",
	Short: "Start the Temporal worker",
	Long: `Start the Temporal worker that polls for tasks and executes barber/shave workflows.

Runs until interrupted with Ctrl+C.`,
	Run: func(cmd *cobra.Command, args []string) {
		temporal.RunWorker()
	},
}
