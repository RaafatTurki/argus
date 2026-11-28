package tui

import (
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func color(light, dark string) lipgloss.AdaptiveColor {
	return lipgloss.AdaptiveColor{Light: light, Dark: dark}
}

var (
	cAccent = color("#5B4BC4", "#9D8CFF")
	cGood   = color("#1F7A3D", "#5FD38D")
	cBad    = color("#B3261E", "#FF7B72")
	cDim    = color("#6B7280", "#7D8590")
	cEdge   = color("#B8BCC8", "#3D4152")
	cSel    = color("#E4E6F5", "#2A2D45")

	sDim    = lipgloss.NewStyle().Foreground(cDim)
	sAccent = lipgloss.NewStyle().Foreground(cAccent)
	sGood   = lipgloss.NewStyle().Foreground(cGood)
	sBad    = lipgloss.NewStyle().Foreground(cBad)
	sBold   = lipgloss.NewStyle().Bold(true)
	sEdge   = lipgloss.NewStyle().Foreground(cEdge)
	sSel    = lipgloss.NewStyle().Background(cSel)
)

func width(s string) int { return ansi.StringWidth(s) }

func clip(s string, w int) string {
	if w <= 0 {
		return ""
	}
	return ansi.Truncate(s, w, "…")
}

func fit(s string, w int) string {
	s = clip(s, w)
	return s + strings.Repeat(" ", max(0, w-width(s)))
}

func between(left, right string, w int) string {
	gap := w - width(left) - width(right)
	if gap < 1 {
		return clip(left, max(0, w-width(right)-1)) + " " + right
	}
	return left + strings.Repeat(" ", gap) + right
}

func heading(title string, w int) string {
	return sBold.Render(title) + " " + sEdge.Render(strings.Repeat("─", max(0, w-width(title)-1)))
}

func home(path string) string {
	if h, _ := os.UserHomeDir(); h != "" && strings.HasPrefix(path, h) {
		return "~" + strings.TrimPrefix(path, h)
	}
	return path
}

func wrap(text string, w int) []string {
	w = max(w, 8)

	var lines []string
	for para := range strings.SplitSeq(text, "\n") {
		line := ""
		for _, word := range strings.Fields(para) {
			switch {
			case line == "":
				line = word
			case width(line)+1+width(word) <= w:
				line += " " + word
			default:
				lines = append(lines, line)
				line = word
			}
		}
		lines = append(lines, line)
	}
	return lines
}

func windowRows(rows []string, sel, n int, off *int) []string {
	if len(rows) <= n {
		*off = 0
		return rows
	}
	*off = min(max(*off, sel-n+1), max(0, sel-1))
	*off = max(0, min(*off, len(rows)-n))
	return rows[*off : *off+n]
}
