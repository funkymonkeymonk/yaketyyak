package temporal

const G2GTag = "@g2g"

type LLMConfig struct {
	BaseURL string `json:"base_url"`
	Model   string `json:"model"`
	APIKey  string `json:"api_key"`
}

type FileChange struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type CISignal struct {
	Conclusion string
	Branch     string
	SHA        string
	Details    string
}

type PRFeedback struct {
	PRNumber int
	Comment  string
	Author   string
}

type YakInfo struct {
	Name     string
	State    string
	Context  string
	Tags     []string
	PRNumber int
	PRURL    string
	G2G      bool
}

type WorkflowState struct {
	Phase             string
	CurrentYak        *YakInfo
	PendingCISignals  []CISignal
	PendingPRFeedback []PRFeedback
	PendingG2GScans   int
	CompletedYaks     int
	FailedYaks        int
	Repo              string
	RepoRoot          string
	G2GMode           bool
}

type G2GYak struct {
	Name string   `json:"name"`
	Tags []string `json:"tags"`
	G2G  bool     `json:"g2g"`
}

type ClaimResult struct {
	Name    string `json:"name"`
	State   string `json:"state"`
	Claimed bool   `json:"claimed"`
}

type AgentResult struct {
	PRNumber int    `json:"pr_number"`
	PRURL    string `json:"pr_url"`
	Branch   string `json:"branch"`
}

type ShaveState struct {
	YakName    string    `json:"yak_name"`
	LLMConfig  LLMConfig `json:"llm_config"`
	Repo       string    `json:"repo"`
	RepoRoot   string    `json:"repo_root"`
	MaxRetries int       `json:"max_retries"`
	Iteration  int       `json:"iteration"`
	Phase      string    `json:"phase"`
	Workspace  string    `json:"workspace"`
	PRNumber   int       `json:"pr_number"`
	PRURL      string    `json:"pr_url"`
}

type ShaveValidationResult struct {
	Passed bool   `json:"passed"`
	Output string `json:"output"`
}

type ShaveReviewResult struct {
	Passed bool     `json:"passed"`
	Issues string   `json:"issues"`
	Items  []string `json:"items"`
}

func ShaveWorkflowID(yakName string) string {
	return "yyx-shave-" + sanitizeWorkflowID(yakName)
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
