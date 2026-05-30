package temporal

import "go.temporal.io/api/enums/v1"

// LLMConfig holds the configuration for the LLM-backed coding agent.
type LLMConfig struct {
	BaseURL string `json:"baseUrl,omitempty"`
	Model   string `json:"model,omitempty"`
	APIKey  string `json:"apiKey,omitempty"`
}

// WorkflowState holds the current observable state of a running workflow.
type WorkflowState struct {
	WorkflowID string                        `json:"workflowId"`
	Status     enums.WorkflowExecutionStatus `json:"status"`
}

// ShaveState tracks progress of a ShaveWorkflow.
type ShaveState struct {
	YakName    string `json:"yakName"`
	Phase      string `json:"phase"`
	Iteration  int    `json:"iteration"`
	MaxRetries int    `json:"maxRetries"`
	Workspace  string `json:"workspace"`
}

// ShaveWorkflowID returns the deterministic Temporal workflow ID for a shave run.
func ShaveWorkflowID(yakName string) string {
	return "yyx-shave-" + sanitizeWorkflowID(yakName)
}

// WorkflowConfig is passed to YakWorkflow at start time.
type WorkflowConfig struct {
	RepoURL string   `json:"repoUrl"` // e.g. "https://github.com/funkymonkeymonk/yaketyyak"
	Pi      PiConfig `json:"pi"`
}

// PiConfig holds the configuration for invoking the Pi coding agent.
type PiConfig struct {
	Model  string   `json:"model,omitempty"`
	Tools  []string `json:"tools,omitempty"`
	Skills []string `json:"skills,omitempty"`
	// MaxRunTimeSeconds caps the wall-clock time for a single agent run.
	// The worker aborts the Pi session after this many seconds and fails the
	// activity.  Defaults to 7200 (2 hours) when zero.
	MaxRunTimeSeconds int `json:"maxRunTimeSeconds,omitempty"`
}

// DefaultPiTools are the tools enabled for every shave run.
var DefaultPiTools = []string{"read", "bash", "edit", "write"}

// DefaultPiModel is the LiteLLM model used when no model is specified.
const DefaultPiModel = "claude-sonnet-4-6"

// YakWorkflowID returns the deterministic Temporal workflow ID for a yak.
func YakWorkflowID(yakName string) string {
	return "yyx-yak-" + sanitizeWorkflowID(yakName)
}

func sanitizeWorkflowID(name string) string {
	result := make([]byte, 0, len(name))
	for i := 0; i < len(name); i++ {
		c := name[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_' {
			result = append(result, c)
		} else {
			result = append(result, '-')
		}
	}
	return string(result)
}
