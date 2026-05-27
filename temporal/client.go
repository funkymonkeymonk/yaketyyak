package temporal

import "go.temporal.io/sdk/client"

const TaskQueue = "yaketyyak-tasks"

func NewClient() (client.Client, error) {
	return client.Dial(client.Options{
		HostPort: "localhost:7233",
	})
}
