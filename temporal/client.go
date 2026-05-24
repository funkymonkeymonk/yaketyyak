package temporal

import (
	"fmt"

	"go.temporal.io/sdk/client"
)

const WorkflowID = "yaketyyak-yak-shaving"
const TaskQueue = "yaketyyak-tasks"

func NewClient() (client.Client, error) {
	c, err := client.Dial(client.Options{
		HostPort: "localhost:7233",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Temporal client: %w", err)
	}
	return c, nil
}
