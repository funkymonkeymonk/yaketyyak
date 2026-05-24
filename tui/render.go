package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	todoColor  = lipgloss.Color("#FFFFFF")
	wipColor   = lipgloss.Color("#FFD700")
	doneColor  = lipgloss.Color("#666666")
	dimColor   = lipgloss.Color("#666666")
	cyanColor  = lipgloss.Color("#00FFFF")
	pinkColor  = lipgloss.Color("#FF69B4")
	greenColor = lipgloss.Color("#00FF00")

	dimStyle      = lipgloss.NewStyle().Foreground(dimColor)
	todoStyle     = lipgloss.NewStyle().Foreground(todoColor)
	wipStyle      = lipgloss.NewStyle().Foreground(wipColor)
	doneStyle     = lipgloss.NewStyle().Foreground(doneColor)
	cyanStyle     = lipgloss.NewStyle().Foreground(cyanColor)
	pinkBoldStyle = lipgloss.NewStyle().Foreground(pinkColor).Bold(true)
	greenStyle    = lipgloss.NewStyle().Foreground(greenColor)
	bgSelected    = lipgloss.NewStyle().Background(lipgloss.Color("#3A3A3A"))
	headerStyle   = lipgloss.NewStyle().Bold(true).Underline(true)
	strikeStyle   = lipgloss.NewStyle().Strikethrough(true)
	repoStyle     = lipgloss.NewStyle().Foreground(cyanColor).Bold(true)
)

func symbolFor(s YakState) string {
	switch s {
	case YakTodo:
		return "○"
	case YakWip:
		return "●"
	case YakDone:
		return "✓"
	}
	return "?"
}

func colorFor(s YakState) lipgloss.Style {
	switch s {
	case YakTodo:
		return todoStyle
	case YakWip:
		return wipStyle
	case YakDone:
		return doneStyle
	}
	return dimStyle
}

func treePrefixFor(tl *treeLine) string {
	if tl.depth == 0 {
		return ""
	}
	var b strings.Builder
	colCount := tl.depth - 1
	if colCount > len(tl.ancestorContinues) {
		colCount = len(tl.ancestorContinues)
	}
	cols := tl.ancestorContinues[:colCount]
	for i := len(cols) - 1; i >= 0; i-- {
		if cols[i] {
			b.WriteString(dimStyle.Render("│ "))
		} else {
			b.WriteString("  ")
		}
	}
	if tl.isLastSibling {
		b.WriteString(dimStyle.Render("╰─"))
	} else {
		b.WriteString(dimStyle.Render("├─"))
	}
	return b.String()
}

func renderTreeLine(tl *treeLine, selected bool, width int) string {
	prefix := treePrefixFor(tl)

	if tl.kind == treeRepo {
		name := repoStyle.Render("◆ " + tl.name)
		line := prefix + name
		if selected {
			line = bgSelected.Render(line)
		}
		return padWidth(line, width)
	}

	sym := symbolFor(tl.state)
	style := colorFor(tl.state)

	prefixWidth := lipgloss.Width(prefix)
	symWidth := lipgloss.Width(sym) + 1
	nameMax := width - prefixWidth - symWidth
	if nameMax < 4 {
		nameMax = 4
	}

	name := tl.name
	runes := []rune(name)
	if len(runes) > nameMax {
		name = string(runes[:nameMax-1]) + "…"
	}

	if tl.state == YakDone {
		name = strikeStyle.Render(name)
	}

	line := fmt.Sprintf("%s%s %s", prefix, style.Render(sym), name)

	out := line
	if selected {
		out = bgSelected.Render(line)
	}
	return padWidth(out, width)
}

func padWidth(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}

func truncateRunes(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	if max < 2 {
		return string(r[:max])
	}
	return string(r[:max-1]) + "…"
}

func wrapText(text string, width int) []string {
	var result []string
	paragraphs := strings.Split(text, "\n")
	for _, para := range paragraphs {
		words := strings.Fields(para)
		if len(words) == 0 {
			result = append(result, "")
			continue
		}
		var line strings.Builder
		for _, word := range words {
			if line.Len() > 0 {
				if line.Len()+1+len(word) > width {
					result = append(result, line.String())
					line.Reset()
				} else {
					line.WriteByte(' ')
				}
			}
			line.WriteString(word)
		}
		if line.Len() > 0 {
			result = append(result, line.String())
		}
	}
	return result
}
