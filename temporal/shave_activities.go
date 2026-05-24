package temporal

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func ShaveInitWorkspace(ctx context.Context, repoRoot, yakName string) (string, error) {
	slug := strings.ToLower(strings.ReplaceAll(yakName, " ", "-"))
	workspaceName := "shave-" + slug

	fetch := exec.CommandContext(ctx, "jj", "git", "fetch")
	fetch.Dir = repoRoot
	if out, err := fetch.CombinedOutput(); err != nil {
		return "", fmt.Errorf("jj git fetch: %w\n%s", err, out)
	}

	add := exec.CommandContext(ctx, "jj", "workspace", "add", "--name", workspaceName, ".workspaces/"+workspaceName)
	add.Dir = repoRoot
	if out, err := add.CombinedOutput(); err != nil {
		return "", fmt.Errorf("jj workspace add: %w\n%s", err, out)
	}

	return workspaceName, nil
}

func ShaveImplement(ctx context.Context, yakName string, cfg LLMConfig, repoRoot, workspaceName string, lastFailure, reviewIssues string, iteration int) error {
	info, err := yakShow(ctx, yakName)
	if err != nil {
		return err
	}

	workspacePath := filepath.Join(repoRoot, ".workspaces", workspaceName)

	existingCode := readWorkspaceFiles(workspacePath)

	system := `You are a Go developer implementing changes for a task.
Output ONLY valid JSON with the files you create or modify.
Use the format: {"files": [{"path": "relative/file/path", "content": "file content here"}]}
Include the complete file contents for every file. Do not use placeholders or "rest of code remains the same".`

	user := fmt.Sprintf("## Yak: %s\n\n%s\n\n", yakName, info.Context)

	if iteration > 1 {
		var feedback string
		if lastFailure != "" {
			feedback += "## Validation failures from previous attempt\n" + lastFailure + "\n\n"
		}
		if reviewIssues != "" {
			feedback += "## Adversarial review found these issues\n" + reviewIssues + "\n\n"
		}
		feedback += "Fix all the issues listed above."
		user += feedback + "\n\n"
	}

	if existingCode != "" {
		user += "## Existing code in workspace:\n" + existingCode + "\n\n"
	}

	user += `## Instructions
1. Write failing tests first (TDD)
2. Implement the minimal code to make tests pass
3. Read the existing code carefully — preserve existing logic unless it conflicts with the yak requirements
4. Output ALL files that need to be created or modified as a JSON object with a "files" array`

	var result struct {
		Files []FileChange `json:"files"`
	}

	client := newLLM(cfg)
	if err := client.chatJSON(ctx, system, user, &result); err != nil {
		return fmt.Errorf("implement yak %s: %w", yakName, err)
	}

	for _, f := range result.Files {
		path := filepath.Join(workspacePath, f.Path)
		os.MkdirAll(filepath.Dir(path), 0755)
		if err := os.WriteFile(path, []byte(f.Content), 0644); err != nil {
			return fmt.Errorf("write %s: %w", f.Path, err)
		}
	}

	return nil
}

func ShaveValidate(ctx context.Context, repoRoot, workspaceName string) (ShaveValidationResult, error) {
	workspacePath := filepath.Join(repoRoot, ".workspaces", workspaceName)

	cmds := []struct {
		name string
		args []string
	}{
		{"go", []string{"vet", "./..."}},
		{"go", []string{"test", "./..."}},
		{"go", []string{"build", "./..."}},
	}

	for _, c := range cmds {
		cmd := exec.CommandContext(ctx, c.name, c.args...)
		cmd.Dir = workspacePath
		out, err := cmd.CombinedOutput()
		if err != nil {
			return ShaveValidationResult{
				Passed: false,
				Output: fmt.Sprintf("%s %s failed:\n%s", c.name, strings.Join(c.args, " "), string(out)),
			}, nil
		}
	}

	return ShaveValidationResult{Passed: true, Output: "all checks passed"}, nil
}

func ShaveAdversarialReview(ctx context.Context, yakName string, cfg LLMConfig, repoRoot, workspaceName string) (ShaveReviewResult, error) {
	info, err := yakShow(ctx, yakName)
	if err != nil {
		return ShaveReviewResult{}, err
	}

	workspacePath := filepath.Join(repoRoot, ".workspaces", workspaceName)

	diffCmd := exec.CommandContext(ctx, "jj", "diff")
	diffCmd.Dir = workspacePath
	diffOut, err := diffCmd.CombinedOutput()
	if err != nil {
		diffOut = []byte("(could not get diff: " + err.Error() + ")")
	}

	system := `You are an adversarial code reviewer. Your job is to find problems.
Output ONLY valid JSON in the format:
{"passed": true/false, "issues": "summary", "items": ["specific issue 1", "specific issue 2"]}

If the code is correct and complete, output {"passed": true, "issues": "", "items": []}.`

	user := fmt.Sprintf(`## Original Requirements
%s

## Current Code Changes (diff)
%s

## Review Instructions
Critically examine the code changes against the original requirements. Look for:
- Bugs and logic errors
- Security vulnerabilities
- Missing edge cases and error handling
- Insufficient test coverage (tests should exist for all new behavior)
- Code that does not match the acceptance criteria
- Design flaws or maintainability issues

Be adversarial: assume nothing, verify everything, find every flaw.`, info.Context, string(diffOut))

	var result ShaveReviewResult
	client := newLLM(cfg)
	if err := client.chatJSON(ctx, system, user, &result); err != nil {
		return ShaveReviewResult{
			Passed: false,
			Issues: "Review agent failed to produce valid output",
			Items:  []string{fmt.Sprintf("review error: %v", err)},
		}, nil
	}

	return result, nil
}

func ShaveCreatePR(ctx context.Context, repoRoot, workspaceName, repo string) (AgentResult, error) {
	workspacePath := filepath.Join(repoRoot, ".workspaces", workspaceName)

	pushCmd := exec.CommandContext(ctx, "jj", "git", "push")
	pushCmd.Dir = workspacePath
	if out, err := pushCmd.CombinedOutput(); err != nil {
		return AgentResult{}, fmt.Errorf("jj git push: %w\n%s", err, out)
	}

	createCmd := exec.CommandContext(ctx, "gh", "pr", "create", "--repo", repo, "--fill")
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

func ShaveCleanup(ctx context.Context, repoRoot, workspaceName string) error {
	forgetCmd := exec.CommandContext(ctx, "jj", "workspace", "forget", workspaceName)
	forgetCmd.Dir = repoRoot
	return forgetCmd.Run()
}

func readWorkspaceFiles(root string) string {
	var b strings.Builder
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		if len(data) > 50000 {
			data = data[:50000]
		}
		b.WriteString(fmt.Sprintf("--- %s ---\n%s\n", rel, string(data)))
		return nil
	})
	if b.Len() > 200000 {
		return b.String()[:200000] + "\n... (truncated)"
	}
	return b.String()
}
