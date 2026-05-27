package temporal

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// YakWorkflow is a single long-running workflow for one yak.
// It starts when the yak is ready to shave (@g2g), runs until the yak is
// closed (PR merged or won't-do signal), and uses Pi as the coding agent.
func YakWorkflow(ctx workflow.Context, yakName string, repoRoot string, cfg PiConfig) (string, error) {
	state := &YakWorkflowState{
		YakName: yakName,
		Phase:   "init",
	}

	if err := workflow.SetQueryHandler(ctx, "yak_status", func() (YakWorkflowState, error) {
		return *state, nil
	}); err != nil {
		return "", err
	}

	wontDoChan := workflow.GetSignalChannel(ctx, "wont-do")
	wontDo := false
	workflow.Go(ctx, func(ctx workflow.Context) {
		wontDoChan.Receive(ctx, nil)
		wontDo = true
	})

	actOpts := func(timeout time.Duration) workflow.Context {
		return workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			StartToCloseTimeout: timeout,
			// Activities that shell out are not retried automatically —
			// transient failures should be handled inside the activity.
			RetryPolicy: &temporal.RetryPolicy{MaximumAttempts: 1},
		})
	}
	retryOpts := func(timeout time.Duration, attempts int32) workflow.Context {
		return workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			StartToCloseTimeout: timeout,
			RetryPolicy:         &temporal.RetryPolicy{MaximumAttempts: attempts},
		})
	}

	// 1. Claim the yak — sets state to wip, acts as a distributed lock.
	state.Phase = "claiming"
	if err := workflow.ExecuteActivity(retryOpts(30*time.Second, 3), YakClaim, yakName).Get(ctx, nil); err != nil {
		return "", fmt.Errorf("claim yak %s: %w", yakName, err)
	}

	// Ensure we release the yak if we exit without completing normally.
	defer func() {
		if state.Phase != "done" && !wontDo {
			workflow.ExecuteActivity(retryOpts(30*time.Second, 3), YakRelease, yakName,
				fmt.Sprintf("workflow interrupted at phase %s", state.Phase)).Get(ctx, nil)
		}
	}()

	// 2. Init workspace — jj workspace add .workspaces/shave-<slug>
	state.Phase = "init-workspace"
	var workspaceName string
	if err := workflow.ExecuteActivity(actOpts(60*time.Second), InitWorkspace, repoRoot, yakName).Get(ctx, &workspaceName); err != nil {
		return "", fmt.Errorf("init workspace: %w", err)
	}
	state.Workspace = workspaceName

	defer workflow.ExecuteActivity(actOpts(30*time.Second), CleanupWorkspace, repoRoot, workspaceName).Get(ctx, nil)

	// 3. Run agent — Pi implements the yak spec in the workspace.
	state.Phase = "implementing"
	if err := workflow.ExecuteActivity(actOpts(4*time.Hour), RunAgent, yakName, repoRoot, workspaceName, cfg).Get(ctx, nil); err != nil {
		return "", fmt.Errorf("run agent: %w", err)
	}

	if wontDo {
		return fmt.Sprintf("yak %s marked won't-do during implementation", yakName), nil
	}

	// 4. Create draft PR — human reviews before it moves forward.
	state.Phase = "creating-pr"
	var pr PRResult
	if err := workflow.ExecuteActivity(actOpts(5*time.Minute), CreateDraftPR, repoRoot, workspaceName, yakName).Get(ctx, &pr); err != nil {
		return "", fmt.Errorf("create draft PR: %w", err)
	}
	state.PRURL = pr.PRURL
	state.PRNumber = pr.PRNumber

	// Write PR URL back to the yak so it's visible in yx list / TUI.
	workflow.ExecuteActivity(retryOpts(30*time.Second, 3), WritePRToYak, yakName, pr.PRURL).Get(ctx, nil)

	// 5. Wait for the PR to be merged (or won't-do).
	// The human drives: review, approve, merge. The workflow just observes.
	state.Phase = "waiting-for-merge"
	sel := workflow.NewSelector(ctx)

	mergedCh := workflow.NewChannel(ctx)

	// Poll for merge in a goroutine; signal the channel when done.
	workflow.Go(ctx, func(gCtx workflow.Context) {
		pollCtx := workflow.WithActivityOptions(gCtx, workflow.ActivityOptions{
			StartToCloseTimeout: 7 * 24 * time.Hour, // up to a week
			HeartbeatTimeout:    2 * time.Minute,
			RetryPolicy:         &temporal.RetryPolicy{MaximumAttempts: 1},
		})
		var merged bool
		err := workflow.ExecuteActivity(pollCtx, WatchPRMerged, pr.PRNumber, repoRoot).Get(gCtx, &merged)
		if err == nil && merged {
			mergedCh.Send(gCtx, true)
		}
	})

	sel.AddReceive(mergedCh, func(c workflow.ReceiveChannel, _ bool) {
		c.Receive(ctx, nil)
	})
	sel.AddReceive(wontDoChan, func(c workflow.ReceiveChannel, _ bool) {
		c.Receive(ctx, nil)
		wontDo = true
	})
	sel.Select(ctx)

	if wontDo {
		workflow.ExecuteActivity(retryOpts(30*time.Second, 3), YakRelease, yakName, "marked won't-do").Get(ctx, nil)
		return fmt.Sprintf("yak %s marked won't-do", yakName), nil
	}

	// 6. PR merged — close the yak.
	state.Phase = "done"
	if err := workflow.ExecuteActivity(retryOpts(30*time.Second, 3), YakMarkDone, yakName).Get(ctx, nil); err != nil {
		return "", fmt.Errorf("mark yak done: %w", err)
	}

	return fmt.Sprintf("yak %s done (PR %s)", yakName, pr.PRURL), nil
}
