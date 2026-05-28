package temporal

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
