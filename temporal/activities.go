package temporal

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func YxSync(ctx context.Context) (string, error) {
	out, err := exec.CommandContext(ctx, "yx", "sync").CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if strings.Contains(msg, "not configured") {
			return "", nil
		}
		return "", fmt.Errorf("yx sync: %s", msg)
	}
	return string(out), nil
}

func YakTriageG2G(ctx context.Context) ([]G2GYak, error) {
	out, err := exec.CommandContext(ctx, "yx", "ls", "--format", "json").Output()
	if err != nil {
		return nil, fmt.Errorf("yx ls: %w", err)
	}

	var all []yxEntry
	if err := json.Unmarshal(out, &all); err != nil {
		return nil, fmt.Errorf("parse yx ls: %w", err)
	}

	flat := flattenYaks(all)
	var g2g []G2GYak
	for _, y := range flat {
		if y.State == "todo" && hasTag(y.Tags, G2GTag) {
			g2g = append(g2g, G2GYak{
				Name: y.Name,
				Tags: y.Tags,
				G2G:  true,
			})
		}
	}
	return g2g, nil
}

func YakTriage(ctx context.Context) ([]G2GYak, error) {
	home, _ := os.UserHomeDir()
	script := filepath.Join(home, ".config/opencode/skills/shave-yaks/scripts/yak-triage.sh")
	if _, err := exec.LookPath(script); err == nil {
		out, err := exec.CommandContext(ctx, script, "--names").Output()
		if err == nil {
			names := strings.Fields(string(out))
			yaks := make([]G2GYak, 0, len(names))
			for _, n := range names {
				yaks = append(yaks, G2GYak{Name: n})
			}
			return yaks, nil
		}
	}

	out, err := exec.CommandContext(ctx, "yx", "ls", "--format", "json").Output()
	if err != nil {
		return nil, fmt.Errorf("yx ls: %w", err)
	}
	var all []yxEntry
	if err := json.Unmarshal(out, &all); err != nil {
		return nil, fmt.Errorf("parse yx ls: %w", err)
	}
	flat := flattenYaks(all)
	var yaks []G2GYak
	for _, y := range flat {
		if y.State == "todo" {
			yaks = append(yaks, G2GYak{Name: y.Name})
		}
	}
	return yaks, nil
}

func YakClaim(ctx context.Context, name string) (ClaimResult, error) {
	home, _ := os.UserHomeDir()
	script := filepath.Join(home, ".config/opencode/skills/shave-yaks/scripts/yak-claim.sh")
	if _, err := exec.LookPath(script); err == nil {
		out, err := exec.CommandContext(ctx, script, name).Output()
		if err == nil && len(out) > 0 {
			return ClaimResult{Name: name, State: "wip", Claimed: true}, nil
		}
	}

	exec.CommandContext(ctx, "yx", "sync").Run()
	exec.CommandContext(ctx, "yx", "start", name).Run()
	exec.CommandContext(ctx, "yx", "sync").Run()
	return ClaimResult{Name: name, State: "wip", Claimed: true}, nil
}

func YakRemoveG2GTag(ctx context.Context, name string) error {
	return exec.CommandContext(ctx, "yx", "tag", "remove", name, G2GTag).Run()
}

func DispatchAgent(ctx context.Context, yakName string, cfg LLMConfig, repoRoot string, g2g bool) (AgentResult, error) {
	info, err := yakShow(ctx, yakName)
	if err != nil {
		return AgentResult{}, err
	}

	tagLine := ""
	if len(info.Tags) > 0 {
		tagLine = "Tags: " + strings.Join(info.Tags, ", ")
	}

	slug := strings.ToLower(strings.ReplaceAll(yakName, " ", "-"))
	workspaceName := "agent-" + slug

	fetch := exec.CommandContext(ctx, "jj", "git", "fetch")
	fetch.Dir = repoRoot
	fetch.Run()

	add := exec.CommandContext(ctx, "jj", "workspace", "add", "--name", workspaceName, ".workspaces/"+workspaceName)
	add.Dir = repoRoot
	if out, err := add.CombinedOutput(); err != nil {
		return AgentResult{}, fmt.Errorf("create workspace: %w\n%s", err, out)
	}

	workspacePath := filepath.Join(repoRoot, ".workspaces", workspaceName)
	defer exec.CommandContext(ctx, "jj", "workspace", "forget", workspaceName).Run()

	existingCode := readWorkspaceFiles(workspacePath)

	system := `You are a Go developer implementing a task (yak). Output ONLY valid JSON.
Format: {"files": [{"path": "relative/file/path", "content": "complete file content"}]}
Include ALL file contents — no placeholders.`

	user := fmt.Sprintf(`## Yak: %s
%s

%s

## Instructions
1. Write tests first (TDD)
2. Implement the minimal code to make them pass
3. Output all files that need to be created or modified as JSON`, yakName, tagLine, info.Context)

	if existingCode != "" {
		user += "\n\n## Existing code:\n" + existingCode
	}

	var result struct {
		Files []FileChange `json:"files"`
	}
	client := newLLM(cfg)
	if err := client.chatJSON(ctx, system, user, &result); err != nil {
		return AgentResult{}, fmt.Errorf("dispatch agent for %s: %w", yakName, err)
	}

	for _, f := range result.Files {
		path := filepath.Join(workspacePath, f.Path)
		os.MkdirAll(filepath.Dir(path), 0755)
		if err := os.WriteFile(path, []byte(f.Content), 0644); err != nil {
			return AgentResult{}, fmt.Errorf("write %s: %w", f.Path, err)
		}
	}

	commitCmd := exec.CommandContext(ctx, "jj", "commit", "-m", "Implement: "+yakName)
	commitCmd.Dir = workspacePath
	if out, err := commitCmd.CombinedOutput(); err != nil {
		return AgentResult{}, fmt.Errorf("jj commit: %w\n%s", err, out)
	}

	pushCmd := exec.CommandContext(ctx, "jj", "git", "push")
	pushCmd.Dir = workspacePath
	if out, err := pushCmd.CombinedOutput(); err != nil {
		return AgentResult{}, fmt.Errorf("jj git push: %w\n%s", err, out)
	}

	repoPath := repoRemoteToPath(repoRoot)
	createCmd := exec.CommandContext(ctx, "gh", "pr", "create", "--repo", repoPath, "--fill")
	createCmd.Dir = workspacePath
	out, err := createCmd.CombinedOutput()
	if err != nil {
		return AgentResult{}, fmt.Errorf("gh pr create: %w\n%s", err, out)
	}

	output := string(out)
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, "/pull/") {
			return AgentResult{PRURL: strings.TrimSpace(line)}, nil
		}
	}

	return AgentResult{}, fmt.Errorf("could not parse PR URL from: %s", output)
}

func WatchPRCI(ctx context.Context, prNumber int, repo string) (string, error) {
	pollInterval := 30 * time.Second
	for {
		out, err := exec.CommandContext(ctx, "gh", "pr", "checks",
			fmt.Sprint(prNumber), "--repo", repo,
			"--json", "name,state,link").Output()
		if err != nil {
			return "", fmt.Errorf("gh pr checks: %w", err)
		}

		var checks []struct {
			State string `json:"state"`
		}
		if err := json.Unmarshal(out, &checks); err != nil {
			return "", fmt.Errorf("parse checks: %w", err)
		}

		if len(checks) == 0 {
			time.Sleep(pollInterval)
			continue
		}

		states := map[string]bool{}
		for _, c := range checks {
			states[c.State] = true
		}

		if states["FAILURE"] || states["ERROR"] {
			return "failure", nil
		}
		if states["SUCCESS"] || (states["SUCCESS"] && states["NEUTRAL"]) {
			return "success", nil
		}

		time.Sleep(pollInterval)
	}
}

func MergePR(ctx context.Context, prNumber int, repo string) (bool, error) {
	err := exec.CommandContext(ctx, "gh", "pr", "merge",
		fmt.Sprint(prNumber), "--repo", repo,
		"--squash", "--delete-branch").Run()
	return err == nil, nil
}

func YakMarkDone(ctx context.Context, name string) error {
	exec.CommandContext(ctx, "yx", "done", name).Run()
	return exec.CommandContext(ctx, "yx", "sync").Run()
}

func YakMarkRefinement(ctx context.Context, name, reason string) error {
	home, _ := os.UserHomeDir()
	script := filepath.Join(home, ".config/opencode/skills/shave-yaks/scripts/yak-mark-refinement.sh")
	if _, err := exec.LookPath(script); err == nil {
		return exec.CommandContext(ctx, script, name, reason).Run()
	}
	exec.CommandContext(ctx, "yx", "tag", name, "@needs-human").Run()
	return exec.CommandContext(ctx, "yx", "sync").Run()
}

func CheckRefinement(ctx context.Context, name string) (*string, error) {
	home, _ := os.UserHomeDir()
	script := filepath.Join(home, ".config/opencode/skills/shave-yaks/scripts/yak-needs-refinement.sh")
	if _, err := exec.LookPath(script); err == nil {
		out, err := exec.CommandContext(ctx, script, name).Output()
		if err != nil {
			msg := strings.TrimSpace(string(out))
			if msg == "" {
				msg = "No context for autonomous implementation"
			}
			return &msg, nil
		}
	}
	return nil, nil
}

type yxEntry struct {
	Name     string    `json:"name"`
	State    string    `json:"state"`
	Tags     []string  `json:"tags"`
	Context  string    `json:"context"`
	Children []yxEntry `json:"children,omitempty"`
}

func flattenYaks(entries []yxEntry) []yxEntry {
	var result []yxEntry
	for _, e := range entries {
		result = append(result, e)
		result = append(result, flattenYaks(e.Children)...)
	}
	return result
}

func hasTag(tags []string, tag string) bool {
	for _, t := range tags {
		if t == tag {
			return true
		}
	}
	return false
}

type yakShowResult struct {
	Context string   `json:"context"`
	Tags    []string `json:"tags"`
}

func yakShow(ctx context.Context, name string) (*yakShowResult, error) {
	out, err := exec.CommandContext(ctx, "yx", "show", name, "--format", "json").Output()
	if err != nil {
		return nil, fmt.Errorf("yx show: %w", err)
	}
	var r yakShowResult
	if err := json.Unmarshal(out, &r); err != nil {
		return nil, fmt.Errorf("parse yx show: %w", err)
	}
	return &r, nil
}

func repoRemoteToPath(repoRoot string) string {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	url := strings.TrimSpace(string(out))
	// Strip protocol prefix
	if idx := strings.Index(url, "://"); idx >= 0 {
		url = url[idx+3:]
	}
	// Strip git@ prefix
	url = strings.TrimPrefix(url, "git@")
	// SSH format uses colon: git@github.com:owner/repo.git
	url = strings.Replace(url, ":", "/", 1)
	url = strings.TrimSuffix(url, ".git")
	// Remove leading slash if any
	url = strings.TrimPrefix(url, "/")
	return url
}
