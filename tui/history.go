package tui

import (
	"fmt"
	"strings"
	"time"

	historypb "go.temporal.io/api/history/v1"
)

// activitySpan represents a single activity execution extracted from workflow history.
type activitySpan struct {
	name     string
	offset   time.Duration // time from workflow start to activity scheduled
	duration time.Duration // from scheduled to completed/failed
	status   string        // "completed", "failed", "running"
	errMsg   string
}

// extractSpans walks the history events and pairs ActivityTaskScheduled →
// ActivityTaskCompleted / ActivityTaskFailed to produce a slice of activitySpans.
// Returns an empty (non-nil) slice if there are no activity events.
func extractSpans(events []*historypb.HistoryEvent) []activitySpan {
	spans := []activitySpan{}
	if len(events) == 0 {
		return spans
	}

	// Determine workflow start time from first event.
	var workflowStart time.Time
	if t := events[0].GetEventTime(); t != nil {
		workflowStart = t.AsTime()
	}

	// Map from scheduledEventId → partially built span.
	type partial struct {
		name      string
		scheduled time.Time
		offset    time.Duration
	}
	pending := make(map[int64]partial)

	for _, ev := range events {
		if ev == nil {
			continue
		}
		switch ev.GetEventType().Number() {
		case 10: // EVENT_TYPE_ACTIVITY_TASK_SCHEDULED
			attrs := ev.GetActivityTaskScheduledEventAttributes()
			if attrs == nil {
				continue
			}
			name := ""
			if at := attrs.GetActivityType(); at != nil {
				name = at.GetName()
			}
			if name == "" {
				name = attrs.GetActivityId()
			}
			var t time.Time
			if et := ev.GetEventTime(); et != nil {
				t = et.AsTime()
			}
			pending[ev.GetEventId()] = partial{
				name:      name,
				scheduled: t,
				offset:    t.Sub(workflowStart),
			}

		case 12: // EVENT_TYPE_ACTIVITY_TASK_COMPLETED
			attrs := ev.GetActivityTaskCompletedEventAttributes()
			if attrs == nil {
				continue
			}
			p, ok := pending[attrs.GetScheduledEventId()]
			if !ok {
				continue
			}
			delete(pending, attrs.GetScheduledEventId())
			var completedAt time.Time
			if et := ev.GetEventTime(); et != nil {
				completedAt = et.AsTime()
			}
			spans = append(spans, activitySpan{
				name:     p.name,
				offset:   p.offset,
				duration: completedAt.Sub(p.scheduled),
				status:   "completed",
			})

		case 13: // EVENT_TYPE_ACTIVITY_TASK_FAILED
			attrs := ev.GetActivityTaskFailedEventAttributes()
			if attrs == nil {
				continue
			}
			p, ok := pending[attrs.GetScheduledEventId()]
			if !ok {
				continue
			}
			delete(pending, attrs.GetScheduledEventId())
			errMsg := ""
			if f := attrs.GetFailure(); f != nil {
				errMsg = firstLine(f.GetMessage())
			}
			var failedAt time.Time
			if et := ev.GetEventTime(); et != nil {
				failedAt = et.AsTime()
			}
			spans = append(spans, activitySpan{
				name:     p.name,
				offset:   p.offset,
				duration: failedAt.Sub(p.scheduled),
				status:   "failed",
				errMsg:   errMsg,
			})
		}
	}

	// Any still-pending scheduled events are "running".
	for id, p := range pending {
		_ = id
		spans = append(spans, activitySpan{
			name:   p.name,
			offset: p.offset,
			status: "running",
		})
	}

	return spans
}

// formatSpanBar renders one timeline row for the mini Gantt chart.
// Format: Name    ░░░▓▓▓▓▓░ ✓ 500ms (or ✗ for failed, → for running)
func formatSpanBar(s activitySpan, width int, total time.Duration) string {
	if total <= 0 {
		total = time.Second
	}

	marker := "✓"
	switch s.status {
	case "failed":
		marker = "✗"
	case "running":
		marker = "→"
	}

	// Name column: 12 chars wide, right-padded.
	nameCol := 12
	name := s.name
	runes := []rune(name)
	if len(runes) > nameCol {
		name = string(runes[:nameCol-1]) + "…"
	} else {
		name = name + strings.Repeat(" ", nameCol-len([]rune(name)))
	}

	// Bar: remaining width minus name, space, marker, space, duration.
	durStr := durLabel(s.duration)
	// Reserve: name(12) + " "(1) + bar + " "(1) + marker(1 or more) + " "(1) + dur
	markerWidth := len([]rune(marker))
	reserved := nameCol + 1 + 1 + markerWidth + 1 + len(durStr)
	barWidth := width - reserved
	if barWidth < 4 {
		barWidth = 4
	}

	bar := renderBar(s.offset, s.duration, total, barWidth)

	result := name + " " + bar + " " + marker + " " + durStr

	if s.status == "failed" && s.errMsg != "" {
		result += "  " + dimStyle.Render(firstLine(s.errMsg))
	}

	return result
}

// renderBar builds the ASCII bar for offset/duration within total width.
// Offset area = '░', active area = '▓', remainder = '░'.
func renderBar(offset, duration, total time.Duration, width int) string {
	if width <= 0 {
		return ""
	}
	if total <= 0 {
		return strings.Repeat("░", width)
	}

	offsetCells := int(float64(width) * float64(offset) / float64(total))
	activeCells := int(float64(width) * float64(duration) / float64(total))
	if activeCells < 1 && duration > 0 {
		activeCells = 1
	}
	if offsetCells+activeCells > width {
		activeCells = width - offsetCells
	}
	if offsetCells > width {
		offsetCells = width
		activeCells = 0
	}
	trailCells := width - offsetCells - activeCells

	return strings.Repeat("░", offsetCells) +
		strings.Repeat("▓", activeCells) +
		strings.Repeat("░", trailCells)
}

// formatEventDetail returns a human-readable detail string for known event attribute types.
// Returns "" for unknown/nil attrs.
func formatEventDetail(ev *historypb.HistoryEvent, attrs interface{}) string {
	if ev == nil || attrs == nil {
		return ""
	}
	switch a := attrs.(type) {
	case *historypb.ActivityTaskScheduledEventAttributes:
		name := ""
		if at := a.GetActivityType(); at != nil {
			name = at.GetName()
		}
		return fmt.Sprintf("Activity: %s (id=%s)", name, a.GetActivityId())
	case *historypb.ActivityTaskCompletedEventAttributes:
		return fmt.Sprintf("Completed (scheduledEventId=%d)", a.GetScheduledEventId())
	case *historypb.ActivityTaskFailedEventAttributes:
		msg := ""
		if f := a.GetFailure(); f != nil {
			msg = firstLine(f.GetMessage())
		}
		return fmt.Sprintf("Failed: %s", msg)
	}
	return ""
}

// durLabel formats a duration into a compact human-readable string.
//   - d < 1s  → "500ms" (integer ms)
//   - d < 60s → "2.5s" (one decimal)
//   - d >= 60s → "2m5s"
func durLabel(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < 60*time.Second {
		secs := float64(d) / float64(time.Second)
		return fmt.Sprintf("%.1fs", secs)
	}
	mins := int(d.Minutes())
	secs := int(d.Seconds()) % 60
	return fmt.Sprintf("%dm%ds", mins, secs)
}

// firstLine returns the text before the first \n or \r.
// Returns the full string if there is no newline.
func firstLine(s string) string {
	for i, c := range s {
		if c == '\n' || c == '\r' {
			return s[:i]
		}
	}
	return s
}
