package cmd

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/charmbracelet/lipgloss"
	"github.com/funkymonkeymonk/yaketyyak/temporal"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Query workflow status",
	Run: func(cmd *cobra.Command, args []string) {
		c := temporal.NewClient()
		defer c.Close()

		value, err := c.QueryWorkflow(cmd.Context(), temporal.WorkflowID, "", "status")
		if err != nil {
			log.Fatalf("Failed to query status: %v", err)
		}

		var state interface{}
		if err := value.Get(&state); err != nil {
			log.Fatalf("Failed to decode status: %v", err)
		}

		b, err := json.MarshalIndent(state, "", "  ")
		if err != nil {
			log.Fatalf("Failed to marshal status: %v", err)
		}

		infoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00")).Bold(true)
		fmt.Println(infoStyle.Render("✦ Workflow Status:"))
		fmt.Println(string(b))
	},
}
