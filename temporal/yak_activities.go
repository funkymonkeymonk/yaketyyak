package temporal

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"go.temporal.io/sdk/activity"
)

// YakClaim marks a yak as wip via yx, acting as a distributed lock.
func YakClaim(ctx context.Context, yakName string) error {
	if out, err := exec.CommandContext(ctx, "yx", "start", yakName).CombinedOutput(); err != nil {
		return fmt.Errorf("yx start %s: %w\n%s", yakName, err, out)
	}
	exec.CommandContext(ctx, "yx", "sync").Run()
	return nil
}

// YakRelease puts a yak back to todo state when a workflow exits without
// completing — e.g. cancelled or interrupted.
func YakRelease(ctx context.Context, yakName, reason string) error {
	exec.CommandContext(ctx, "yx", "state", yakName, "todo").Run()
	exec.CommandContext(ctx, "yx", "sync").Run()
	return nil
}

// YakMarkDone closes a yak after its PR merges.
func YakMarkDone(ctx context.Context, yakName string) error {
	if out, err := exec.CommandContext(ctx, "yx", "done", yakName).CombinedOutput(); err != nil {
		return fmt.Errorf("yx done %s: %w\n%s", yakName, err, out)
	}
	exec.CommandContext(ctx, "yx", "sync").Run()
	return nil
}

// WritePRToYak records the PR URL on the yak so it's visible in yx / TUI.
func WritePRToYak(ctx context.Context, yakName, prURL string) error {
	return exec.CommandContext(ctx, "yx", "field", yakName, "pr", prURL).Run()
}

// InitWorkspace creates a jj workspace for the yak and returns its name.
func InitWorkspace(ctx context.Context, repoRoot, yakName string) (string, error) {
	slug := sanitizeWorkflowID(strings.ToLower(yakName))
	workspaceName := "shave-" + slug
	workspacePath := ".workspaces/" + workspaceName

	// Ensure the parent directory exists before jj tries to access it.
	if err := os.MkdirAll(filepath.Join(repoRoot, ".workspaces"), 0755); err != nil {
		return "", fmt.Errorf("create .workspaces dir: %w", err)
	}

	// Remove any stale workspace directory from a previous failed run.
	fullWorkspacePath := filepath.Join(repoRoot, workspacePath)
	if err := os.RemoveAll(fullWorkspacePath); err != nil {
		return "", fmt.Errorf("clean stale workspace: %w", err)
	}

	fetch := exec.CommandContext(ctx, "jj", "git", "fetch")
	fetch.Dir = repoRoot
	if out, err := fetch.CombinedOutput(); err != nil {
		return "", fmt.Errorf("jj git fetch: %w\n%s", err, out)
	}

	add := exec.CommandContext(ctx, "jj", "workspace", "add", "--name", workspaceName, workspacePath)
	add.Dir = repoRoot
	if out, err := add.CombinedOutput(); err != nil {
		return "", fmt.Errorf("jj workspace add: %w\n%s", err, out)
	}

	return workspaceName, nil
}

// CleanupWorkspace removes the jj workspace after the shave completes.
func CleanupWorkspace(ctx context.Context, repoRoot, workspaceName string) error {
	cmd := exec.CommandContext(ctx, "jj", "workspace", "forget", workspaceName)
	cmd.Dir = repoRoot
	return cmd.Run()
}

// RunAgent invokes Pi in headless mode to implement the yak spec.
// Pi runs inside the jj workspace with access to the full repo.
// Provider is always LiteLLM — LITELLM_BASE_URL and LITELLM_API_KEY must be set in the environment.
// The activity heartbeats so Temporal knows it's still alive during long runs.
func RunAgent(ctx context.Context, yakName, repoRoot, workspaceName string, cfg PiConfig) error {
	workspacePath := filepath.Join(repoRoot, ".workspaces", workspaceName)

	contextFile, err := writeYakContextToFile(ctx, yakName, workspacePath)
	if err != nil {
		return err
	}
	defer os.Remove(contextFile)

	tools := cfg.Tools
	if len(tools) == 0 {
		tools = DefaultPiTools
	}

	args := []string{
		"--print",
		"--no-session",
		"--tools", strings.Join(tools, ","),
		"--extension", "npm:pi-provider-litellm",
		"--provider", "litellm",
	}

	if cfg.Model != "" {
		args = append(args, "--model", cfg.Model)
	}
	for _, skill := range cfg.Skills {
		args = append(args, "--skill", skill)
	}

	args = append(args,
		"@"+contextFile,
		"Implement this yak. Follow the spec exactly. When done, commit your changes with jj.",
	)

	cmd := exec.CommandContext(ctx, "pi", args...)
	cmd.Dir = workspacePath
	cmd.Env = os.Environ() // inherits LITELLM_BASE_URL + LITELLM_API_KEY from op run

	// Capture output so it appears in error messages; also heartbeat during long runs.
	pr, pw, err := os.Pipe()
	if err != nil {
		return fmt.Errorf("create pipe: %w", err)
	}
	cmd.Stdout = pw
	cmd.Stderr = pw

	if err := cmd.Start(); err != nil {
		pw.Close()
		pr.Close()
		return fmt.Errorf("start pi: %w", err)
	}
	pw.Close()

	// Collect output and heartbeat concurrently.
	var outputBuf strings.Builder
	readDone := make(chan struct{})
	go func() {
		defer close(readDone)
		buf := make([]byte, 4096)
		for {
			n, err := pr.Read(buf)
			if n > 0 {
				outputBuf.Write(buf[:n])
			}
			if err != nil {
				break
			}
		}
		pr.Close()
	}()

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case err := <-done:
			<-readDone
			if err != nil {
				output := outputBuf.String()
				if len(output) > 2000 {
					output = output[len(output)-2000:]
				}
				return fmt.Errorf("pi exited with error: %w\n--- pi output ---\n%s", err, output)
			}
			return nil
		case <-ticker.C:
			activity.RecordHeartbeat(ctx, "pi running")
		case <-ctx.Done():
			cmd.Process.Kill()
			<-readDone
			return ctx.Err()
		}
	}
}

// CreateDraftPR pushes the workspace branch and opens a draft PR.
func CreateDraftPR(ctx context.Context, repoRoot, workspaceName, yakName string) (PRResult, error) {
	workspacePath := filepath.Join(repoRoot, ".workspaces", workspaceName)

	pushCmd := exec.CommandContext(ctx, "jj", "git", "push", "--allow-new")
	pushCmd.Dir = workspacePath
	if out, err := pushCmd.CombinedOutput(); err != nil {
		return PRResult{}, fmt.Errorf("jj git push: %w\n%s", err, out)
	}

	repo := repoRemoteToPath(repoRoot)

	createArgs := []string{"pr", "create", "--draft", "--fill"}
	if repo != "" {
		createArgs = append(createArgs, "--repo", repo)
	}

	createCmd := exec.CommandContext(ctx, "gh", createArgs...)
	createCmd.Dir = workspacePath
	out, err := createCmd.CombinedOutput()
	if err != nil {
		return PRResult{}, fmt.Errorf("gh pr create: %w\n%s", err, out)
	}

	return parsePROutput(string(out))
}

// WatchPRMerged polls until the PR is merged or the context is cancelled.
// It heartbeats so Temporal knows the activity is alive during long waits.
func WatchPRMerged(ctx context.Context, prNumber int, repoRoot string) (bool, error) {
	repo := repoRemoteToPath(repoRoot)
	pollInterval := 60 * time.Second

	for {
		activity.RecordHeartbeat(ctx, fmt.Sprintf("watching PR #%d", prNumber))

		args := []string{"pr", "view", strconv.Itoa(prNumber), "--json", "state,merged"}
		if repo != "" {
			args = append(args, "--repo", repo)
		}

		out, err := exec.CommandContext(ctx, "gh", args...).Output()
		if err != nil {
			// Transient gh errors — log and retry next tick.
			select {
			case <-ctx.Done():
				return false, ctx.Err()
			case <-time.After(pollInterval):
				continue
			}
		}

		var pr struct {
			State  string `json:"state"`
			Merged bool   `json:"merged"`
		}
		if err := json.Unmarshal(out, &pr); err == nil && pr.Merged {
			return true, nil
		}

		select {
		case <-ctx.Done():
			return false, ctx.Err()
		case <-time.After(pollInterval):
		}
	}
}

// --- helpers ---

// writeYakContextToFile fetches the yak spec from yx and writes it to a temp
// file in the workspace so Pi can reference it via @path.
func writeYakContextToFile(ctx context.Context, yakName, workspacePath string) (string, error) {
	type yakShowResult struct {
		Context string `json:"context"`
	}

	out, err := exec.CommandContext(ctx, "yx", "show", yakName, "--format", "json").Output()
	if err != nil {
		return "", fmt.Errorf("yx show %s: %w", yakName, err)
	}

	var r struct {
		Context    string `json:"context"`
		HasContext bool   `json:"has_context"`
	}
	if err := json.Unmarshal(out, &r); err != nil {
		return "", fmt.Errorf("parse yx show: %w", err)
	}

	if !r.HasContext || r.Context == "" {
		return "", fmt.Errorf("yak %q has no context — add a spec with: yx context %s", yakName, yakName)
	}

	f, err := os.CreateTemp(workspacePath, ".yak-context-*.md")
	if err != nil {
		return "", fmt.Errorf("create context temp file: %w", err)
	}
	defer f.Close()

	if _, err := f.WriteString(r.Context); err != nil {
		return "", fmt.Errorf("write context: %w", err)
	}

	return f.Name(), nil
}

// repoRemoteToPath derives "owner/repo" from the git remote URL.
func repoRemoteToPath(repoRoot string) string {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	url := strings.TrimSpace(string(out))
	if idx := strings.Index(url, "://"); idx >= 0 {
		url = url[idx+3:]
	}
	url = strings.TrimPrefix(url, "git@")
	url = strings.Replace(url, ":", "/", 1)
	url = strings.TrimSuffix(url, ".git")
	url = strings.TrimPrefix(url, "/")
	return url
}

// parsePROutput extracts the PR URL and number from gh pr create output.
func parsePROutput(output string) (PRResult, error) {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "/pull/") {
			// Extract PR number from URL like https://github.com/owner/repo/pull/42
			parts := strings.Split(line, "/pull/")
			if len(parts) == 2 {
				numStr := strings.TrimSpace(parts[1])
				n, _ := strconv.Atoi(numStr)
				return PRResult{PRURL: line, PRNumber: n}, nil
			}
		}
	}
	return PRResult{}, fmt.Errorf("could not parse PR URL from: %s", output)
}
