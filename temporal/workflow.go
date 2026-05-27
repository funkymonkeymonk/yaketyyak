package temporal

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// Activity stubs — implementations live in worker/src/activities.ts and worker/src/run-agent.ts.
// Names must match the keys registered in the TypeScript worker exactly.
var (
	yakClaim         = "YakClaim"
	yakRelease       = "YakRelease"
	yakMarkDone      = "YakMarkDone"
	writePRToYak     = "WritePRToYak"
	initWorkspace    = "InitWorkspace"
	cleanupWorkspace = "CleanupWorkspace"
	runAgent         = "RunAgent"
	createDraftPR    = "CreateDraftPR"
	watchPRMerged    = "WatchPRMerged"
)

// YakWorkflow is a single long-running workflow for one yak.
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

	noRetry := func(timeout time.Duration) workflow.Context {
		return workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			StartToCloseTimeout: timeout,
			RetryPolicy:         &temporal.RetryPolicy{MaximumAttempts: 1},
		})
	}
	withRetry := func(timeout time.Duration, attempts int32) workflow.Context {
		return workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			StartToCloseTimeout: timeout,
			RetryPolicy:         &temporal.RetryPolicy{MaximumAttempts: attempts},
		})
	}

	// 1. Claim the yak.
	state.Phase = "claiming"
	if err := workflow.ExecuteActivity(withRetry(30*time.Second, 3), yakClaim, yakName).Get(ctx, nil); err != nil {
		return "", fmt.Errorf("claim yak %s: %w", yakName, err)
	}

	defer func() {
		if state.Phase != "done" && !wontDo {
			workflow.ExecuteActivity(withRetry(30*time.Second, 3), yakRelease, yakName,
				fmt.Sprintf("workflow interrupted at phase %s", state.Phase)).Get(ctx, nil)
		}
	}()

	// 2. Init workspace.
	state.Phase = "init-workspace"
	var workspaceName string
	if err := workflow.ExecuteActivity(noRetry(60*time.Second), initWorkspace, repoRoot, yakName).Get(ctx, &workspaceName); err != nil {
		return "", fmt.Errorf("init workspace: %w", err)
	}
	state.Workspace = workspaceName

	defer workflow.ExecuteActivity(noRetry(30*time.Second), cleanupWorkspace, repoRoot, workspaceName).Get(ctx, nil)

	// 3. Run Pi agent.
	state.Phase = "implementing"
	if err := workflow.ExecuteActivity(noRetry(4*time.Hour), runAgent, yakName, repoRoot, workspaceName, cfg).Get(ctx, nil); err != nil {
		return "", fmt.Errorf("run agent: %w", err)
	}

	if wontDo {
		return fmt.Sprintf("yak %s marked won't-do during implementation", yakName), nil
	}

	// 4. Create draft PR.
	state.Phase = "creating-pr"
	var pr PRResult
	if err := workflow.ExecuteActivity(noRetry(5*time.Minute), createDraftPR, repoRoot, workspaceName, yakName).Get(ctx, &pr); err != nil {
		return "", fmt.Errorf("create draft PR: %w", err)
	}
	state.PRURL = pr.PRURL
	state.PRNumber = pr.PRNumber

	workflow.ExecuteActivity(withRetry(30*time.Second, 3), writePRToYak, yakName, pr.PRURL).Get(ctx, nil)

	// 5. Wait for merge.
	state.Phase = "waiting-for-merge"
	sel := workflow.NewSelector(ctx)
	mergedCh := workflow.NewChannel(ctx)

	workflow.Go(ctx, func(gCtx workflow.Context) {
		pollCtx := workflow.WithActivityOptions(gCtx, workflow.ActivityOptions{
			StartToCloseTimeout: 7 * 24 * time.Hour,
			HeartbeatTimeout:    2 * time.Minute,
			RetryPolicy:         &temporal.RetryPolicy{MaximumAttempts: 1},
		})
		var merged bool
		if err := workflow.ExecuteActivity(pollCtx, watchPRMerged, pr.PRNumber, repoRoot).Get(gCtx, &merged); err == nil && merged {
			mergedCh.Send(gCtx, true)
		}
	})

	sel.AddReceive(mergedCh, func(c workflow.ReceiveChannel, _ bool) { c.Receive(ctx, nil) })
	sel.AddReceive(wontDoChan, func(c workflow.ReceiveChannel, _ bool) {
		c.Receive(ctx, nil)
		wontDo = true
	})
	sel.Select(ctx)

	if wontDo {
		workflow.ExecuteActivity(withRetry(30*time.Second, 3), yakRelease, yakName, "marked won't-do").Get(ctx, nil)
		return fmt.Sprintf("yak %s marked won't-do", yakName), nil
	}

	// 6. Close the yak.
	state.Phase = "done"
	if err := workflow.ExecuteActivity(withRetry(30*time.Second, 3), yakMarkDone, yakName).Get(ctx, nil); err != nil {
		return "", fmt.Errorf("mark yak done: %w", err)
	}

	return fmt.Sprintf("yak %s done (PR %s)", yakName, pr.PRURL), nil
}
