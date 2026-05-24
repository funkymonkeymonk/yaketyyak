package temporal

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

func ShaveWorkflow(ctx workflow.Context, yakName string, cfg LLMConfig, repo, repoRoot string, maxRetries int) (string, error) {
	if maxRetries <= 0 {
		maxRetries = 3
	}

	state := &ShaveState{
		YakName:    yakName,
		LLMConfig:  cfg,
		Repo:       repo,
		RepoRoot:   repoRoot,
		MaxRetries: maxRetries,
		Iteration:  0,
		Phase:      "init",
	}

	if err := workflow.SetQueryHandler(ctx, "shave_status", func() (ShaveState, error) {
		return *state, nil
	}); err != nil {
		return "", err
	}

	cancelChan := workflow.GetSignalChannel(ctx, "shave_cancel")
	cancelled := false
	workflow.Go(ctx, func(ctx workflow.Context) {
		cancelChan.Receive(ctx, nil)
		cancelled = true
	})

	actOpts := func(timeout time.Duration, attempts int32) workflow.Context {
		return workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			StartToCloseTimeout: timeout,
			RetryPolicy:         &temporal.RetryPolicy{MaximumAttempts: attempts},
		})
	}

	// 1. Sync
	state.Phase = "syncing"
	if err := workflow.ExecuteActivity(actOpts(30*time.Second, 3), YxSync).Get(ctx, nil); err != nil {
		return "", err
	}

	// 2. Claim yak
	state.Phase = "claiming"
	var claim ClaimResult
	if err := workflow.ExecuteActivity(actOpts(30*time.Second, 3), YakClaim, yakName).Get(ctx, &claim); err != nil {
		return "", fmt.Errorf("failed to claim yak: %w", err)
	}
	if !claim.Claimed {
		return "", fmt.Errorf("yak %s is already claimed or does not exist", yakName)
	}

	defer func() {
		if state.Phase != "done" && state.Phase != "needs-human" {
			workflow.ExecuteActivity(actOpts(30*time.Second, 1), YakMarkRefinement, yakName,
				fmt.Sprintf("Shave loop interrupted at phase %s", state.Phase)).Get(ctx, nil)
		}
	}()

	// 3. Init workspace
	state.Phase = "init-workspace"
	var workspaceName string
	if err := workflow.ExecuteActivity(actOpts(60*time.Second, 2), ShaveInitWorkspace, repoRoot, yakName).Get(ctx, &workspaceName); err != nil {
		return "", err
	}
	state.Workspace = workspaceName

	defer workflow.ExecuteActivity(actOpts(30*time.Second, 1), ShaveCleanup, repoRoot, workspaceName).Get(ctx, nil)

	// 4. Shave loop: implement -> validate -> review -> fix -> loop
	var lastFailure string
	var reviewIssues string
	accepted := false

	for state.Iteration = 1; state.Iteration <= maxRetries; state.Iteration++ {
		if cancelled {
			state.Phase = "cancelled"
			return fmt.Sprintf("ralph shave of %s cancelled at iteration %d", yakName, state.Iteration), nil
		}

		// Implement or fix
		state.Phase = "implementing"
		if err := workflow.ExecuteActivity(actOpts(2*time.Hour, 1), ShaveImplement,
			yakName, cfg, repoRoot, workspaceName, lastFailure, reviewIssues, state.Iteration).Get(ctx, nil); err != nil {
			return "", err
		}

		if cancelled {
			state.Phase = "cancelled"
			return fmt.Sprintf("ralph shave of %s cancelled at iteration %d", yakName, state.Iteration), nil
		}

		// Validate
		state.Phase = "validating"
		var valResult ShaveValidationResult
		if err := workflow.ExecuteActivity(actOpts(15*time.Minute, 1), ShaveValidate, repoRoot, workspaceName).Get(ctx, &valResult); err != nil {
			return "", err
		}

		if !valResult.Passed {
			lastFailure = valResult.Output
			reviewIssues = ""
			continue
		}

		// Adversarial review
		state.Phase = "reviewing"
		var reviewResult ShaveReviewResult
		if err := workflow.ExecuteActivity(actOpts(1*time.Hour, 1), ShaveAdversarialReview,
			yakName, cfg, repoRoot, workspaceName).Get(ctx, &reviewResult); err != nil {
			return "", err
		}

		if !reviewResult.Passed {
			lastFailure = ""
			reviewIssues = reviewResult.Issues
			for _, item := range reviewResult.Items {
				reviewIssues += "\n- " + item
			}
			continue
		}

		accepted = true
		state.Phase = "accepted"
		break
	}

	if !accepted {
		state.Phase = "needs-human"
		reason := fmt.Sprintf("Shave loop exhausted %d retries", maxRetries)
		if lastFailure != "" {
			reason += "\n\nLast validation failure:\n" + lastFailure
		}
		if reviewIssues != "" {
			reason += "\n\nLast review issues:\n" + reviewIssues
		}
		workflow.ExecuteActivity(actOpts(30*time.Second, 1), YakMarkRefinement, yakName, reason).Get(ctx, nil)
		return fmt.Sprintf("yak %s needs human attention after %d retries", yakName, maxRetries), nil
	}

	// 5. Create PR
	state.Phase = "creating-pr"
	var prResult AgentResult
	if err := workflow.ExecuteActivity(actOpts(5*time.Minute, 2), ShaveCreatePR, repoRoot, workspaceName, repo).Get(ctx, &prResult); err != nil {
		return "", err
	}
	state.PRURL = prResult.PRURL
	state.PRNumber = prResult.PRNumber

	// 6. Watch CI
	state.Phase = "watching-ci"
	var ciResult string
	if prResult.PRNumber > 0 {
		if err := workflow.ExecuteActivity(actOpts(4*time.Hour, 1), WatchPRCI, prResult.PRNumber, repo).Get(ctx, &ciResult); err != nil {
			return "", err
		}
	}

	// 7. Merge if CI passed
	if ciResult == "success" {
		state.Phase = "merging"
		var merged bool
		if err := workflow.ExecuteActivity(actOpts(5*time.Minute, 3), MergePR, prResult.PRNumber, repo).Get(ctx, &merged); err != nil {
			return "", err
		}
		if merged {
			state.Phase = "done"
			workflow.ExecuteActivity(actOpts(30*time.Second, 1), YakMarkDone, yakName).Get(ctx, nil)
			return fmt.Sprintf("yak %s shaved successfully (PR #%d)", yakName, prResult.PRNumber), nil
		}
	}

	state.Phase = "needs-human"
	workflow.ExecuteActivity(actOpts(30*time.Second, 1), YakMarkRefinement, yakName,
		fmt.Sprintf("PR #%d created but CI did not pass. Result: %s", prResult.PRNumber, ciResult)).Get(ctx, nil)
	return fmt.Sprintf("yak %s: PR #%d created, CI: %s", yakName, prResult.PRNumber, ciResult), nil
}
