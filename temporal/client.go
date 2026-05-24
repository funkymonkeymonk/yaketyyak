package temporal

import (
	"log"

	"go.temporal.io/sdk/client"
)

const WorkflowID = "yaketyyak-yak-shaving"
const TaskQueue = "yaketyyak-tasks"

func NewClient() client.Client {
	c, err := client.Dial(client.Options{
		HostPort: "localhost:7233",
	})
	if err != nil {
		log.Fatalf("Failed to create Temporal client: %v", err)
	}
	return c
}
