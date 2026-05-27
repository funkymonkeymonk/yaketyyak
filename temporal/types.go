package temporal

// PiConfig holds the configuration for invoking the Pi coding agent.
// Provider is always LiteLLM — configured via LITELLM_BASE_URL and LITELLM_API_KEY env vars.
type PiConfig struct {
	Model  string   // LiteLLM model name; defaults to DefaultPiModel
	Tools  []string // e.g. ["read","bash","edit","write"]
	Skills []string // paths to skill files loaded via --skill
}

// DefaultPiTools are the tools enabled for every shave run.
var DefaultPiTools = []string{"read", "bash", "edit", "write"}

// DefaultPiModel is the LiteLLM model used when no model is specified.
const DefaultPiModel = "moonshotai.kimi-k2.5"

// PRFeedback carries review comments to feed back to the agent.
type PRFeedback struct {
	PRNumber int
	Comment  string
	Author   string
}

// YakWorkflowState is the queryable state of a running YakWorkflow.
type YakWorkflowState struct {
	YakName   string `json:"yak_name"`
	Phase     string `json:"phase"`
	Workspace string `json:"workspace"`
	PRURL     string `json:"pr_url"`
	PRNumber  int    `json:"pr_number"`
}

// PRResult is returned by CreateDraftPR.
type PRResult struct {
	PRURL    string `json:"pr_url"`
	PRNumber int    `json:"pr_number"`
}

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
