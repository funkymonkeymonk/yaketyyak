package temporal

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

func BarberWorkflow(ctx workflow.Context, repo, repoRoot string, cfg LLMConfig, g2gMode bool, g2gScanIntervalMinutes int) (string, error) {
	state := &WorkflowState{
		Repo:     repo,
		RepoRoot: repoRoot,
		G2GMode:  g2gMode,
	}

	scanInterval := time.Duration(g2gScanIntervalMinutes) * time.Minute

	if err := workflow.SetQueryHandler(ctx, "status", func() (WorkflowState, error) {
		return *state, nil
	}); err != nil {
		return "", err
	}

	ciSignalChan := workflow.GetSignalChannel(ctx, "ci_signal")
	g2gSignalChan := workflow.GetSignalChannel(ctx, "g2g_signal")
	feedbackChan := workflow.GetSignalChannel(ctx, "pr_feedback")
	pauseChan := workflow.GetSignalChannel(ctx, "pause")
	resumeChan := workflow.GetSignalChannel(ctx, "resume")

	for {
		if state.Phase == "paused" {
			workflow.Await(ctx, func() bool { return state.Phase != "paused" })
		}

		hasCI := len(state.PendingCISignals) > 0
		hasG2G := state.PendingG2GScans > 0

		if hasCI || hasG2G || g2gMode {
			if hasCI {
				state.PendingCISignals = state.PendingCISignals[1:]
			}
			processed, err := triageAndImplement(ctx, repo, repoRoot, cfg, g2gMode, state)
			if err != nil {
				return "", err
			}
			if !processed {
				state.Phase = "idle"
			}
			continue
		}

		state.Phase = "idle"
		sel := workflow.NewSelector(ctx)

		sel.AddReceive(ciSignalChan, func(c workflow.ReceiveChannel, _ bool) {
			var val []interface{}
			c.Receive(ctx, &val)
			if len(val) >= 4 {
				state.PendingCISignals = append(state.PendingCISignals, CISignal{
					Conclusion: toString(val[0]),
					Branch:     toString(val[1]),
					SHA:        toString(val[2]),
					Details:    toString(val[3]),
				})
			}
		})
		sel.AddReceive(g2gSignalChan, func(c workflow.ReceiveChannel, _ bool) {
			c.Receive(ctx, nil)
			state.PendingG2GScans++
		})
		sel.AddReceive(feedbackChan, func(c workflow.ReceiveChannel, _ bool) {
			var val []interface{}
			c.Receive(ctx, &val)
			if len(val) >= 3 {
				state.PendingPRFeedback = append(state.PendingPRFeedback, PRFeedback{
					PRNumber: toInt(val[0]),
					Comment:  toString(val[1]),
					Author:   toString(val[2]),
				})
			}
		})
		sel.AddReceive(pauseChan, func(c workflow.ReceiveChannel, _ bool) {
			c.Receive(ctx, nil)
			state.Phase = "paused"
		})
		sel.AddReceive(resumeChan, func(c workflow.ReceiveChannel, _ bool) {
			c.Receive(ctx, nil)
			if state.Phase == "paused" {
				state.Phase = "idle"
			}
		})

		timer := workflow.NewTimer(ctx, scanInterval)
		sel.AddFuture(timer, func(f workflow.Future) {
			state.PendingG2GScans++
		})

		sel.Select(ctx)
	}
}

func triageAndImplement(ctx workflow.Context, repo, repoRoot string, cfg LLMConfig, g2gMode bool, state *WorkflowState) (bool, error) {
	state.Phase = "triaging"

	actOpts := func(timeout time.Duration, attempts int32) workflow.Context {
		return workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			StartToCloseTimeout: timeout,
			RetryPolicy:         &temporal.RetryPolicy{MaximumAttempts: attempts},
		})
	}

	err := workflow.ExecuteActivity(
		actOpts(30*time.Second, 3), YxSync).Get(ctx, nil)
	if err != nil {
		return false, err
	}

	var actionable []G2GYak
	if g2gMode || state.PendingG2GScans > 0 {
		state.PendingG2GScans = 0
		var g2gYaks []G2GYak
		err = workflow.ExecuteActivity(
			actOpts(30*time.Second, 2), YakTriageG2G).Get(ctx, &g2gYaks)
		if err != nil {
			return false, err
		}
		actionable = g2gYaks
	}

	if len(actionable) == 0 {
		var triageYaks []G2GYak
		err = workflow.ExecuteActivity(
			actOpts(30*time.Second, 3), YakTriage).Get(ctx, &triageYaks)
		if err != nil {
			return false, err
		}
		actionable = triageYaks
	}

	if len(actionable) == 0 {
		workflow.GetLogger(ctx).Info("No actionable yaks found")
		state.Phase = "idle"
		return false, nil
	}

	for _, yak := range actionable {
		name := yak.Name
		isG2G := yak.G2G

		var needsRefinement *string
		err = workflow.ExecuteActivity(
			actOpts(30*time.Second, 1), CheckRefinement, name).Get(ctx, &needsRefinement)
		if err != nil {
			return false, err
		}
		if needsRefinement != nil {
			workflow.ExecuteActivity(
				actOpts(30*time.Second, 1), YakMarkRefinement, name, "Flagged by yaketyyak workflow: "+*needsRefinement).Get(ctx, nil)
			continue
		}

		var claim ClaimResult
		err = workflow.ExecuteActivity(
			actOpts(30*time.Second, 3), YakClaim, name).Get(ctx, &claim)
		if err != nil {
			return false, err
		}
		if !claim.Claimed {
			continue
		}

		if isG2G {
			workflow.ExecuteActivity(
				actOpts(15*time.Second, 1), YakRemoveG2GTag, name).Get(ctx, nil)
		}

		state.CurrentYak = &YakInfo{
			Name: name,
			Tags: yak.Tags,
			G2G:  isG2G,
		}

		state.Phase = "implementing"
		var agentResult AgentResult
		err = workflow.ExecuteActivity(
			actOpts(2*time.Hour, 2), DispatchAgent, name, cfg, repoRoot, isG2G).Get(ctx, &agentResult)
		if err != nil {
			return false, err
		}

		if agentResult.PRNumber == 0 {
			workflow.GetLogger(ctx).Error("Agent dispatch failed", "yak", name)
			state.FailedYaks++
			continue
		}

		state.CurrentYak.PRNumber = agentResult.PRNumber
		state.CurrentYak.PRURL = agentResult.PRURL

		state.Phase = "watching-ci"
		var ciResult string
		err = workflow.ExecuteActivity(
			actOpts(4*time.Hour, 1), WatchPRCI, agentResult.PRNumber, repo).Get(ctx, &ciResult)
		if err != nil {
			return false, err
		}

		for ciResult == "failure" || len(state.PendingPRFeedback) > 0 {
			var feedback *PRFeedback
			if len(state.PendingPRFeedback) > 0 {
				fb := state.PendingPRFeedback[0]
				state.PendingPRFeedback = state.PendingPRFeedback[1:]
				feedback = &fb
			}

			state.Phase = "reviewing"
			reworkPrompt := fmt.Sprintf("PR #%d needs changes.\n\n", agentResult.PRNumber)
			if feedback != nil {
				reworkPrompt += fmt.Sprintf("Feedback from %s: %s\n\n", feedback.Author, feedback.Comment)
			}
			reworkPrompt += fmt.Sprintf("Update the PR at %s to address all feedback.", agentResult.PRURL)

			workflow.ExecuteActivity(
				actOpts(1*time.Hour, 2), DispatchAgent, name+" (rework)", cfg, repoRoot, isG2G).Get(ctx, nil)

			ciResult = ""
			err = workflow.ExecuteActivity(
				actOpts(2*time.Hour, 1), WatchPRCI, agentResult.PRNumber, repo).Get(ctx, &ciResult)
			if err != nil {
				return false, err
			}
		}

		if ciResult == "success" {
			var merged bool
			err = workflow.ExecuteActivity(
				actOpts(5*time.Minute, 3), MergePR, agentResult.PRNumber, repo).Get(ctx, &merged)
			if err != nil {
				return false, err
			}
			if merged {
				workflow.ExecuteActivity(
					actOpts(30*time.Second, 1), YakMarkDone, name).Get(ctx, nil)
				state.CompletedYaks++
				state.CurrentYak = nil
			}
		}
	}

	state.Phase = "idle"
	return true, nil
}

func toString(v interface{}) string {
	s, _ := v.(string)
	return s
}

func toInt(v interface{}) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	}
	return 0
}
