package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/RaafatTurki/argus/internal/keys"
)

type helpView struct{ off int }

func (v *helpView) title(a *App) string { return "Keys" }

func (v *helpView) hints(a *App) string {
	return a.hints("up", "scroll", "back", "close") + sDim.Render("  ·  customize: argus keys")
}

func (v *helpView) size(a *App) (int, int) { return min(a.w-8, 100), min(a.h-6, 30) }

func (v *helpView) update(a *App, msg tea.Msg) (popup, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch {
		case a.is(key, "back", "quit", "help"):
			return nil, nil
		case a.is(key, "up"):
			v.off = max(0, v.off-1)
		case a.is(key, "down"):
			w, h := a.popupBody(v)
			v.off = min(v.off+1, max(0, len(v.lines(a, w))-h))
		}
	}
	return v, nil
}

func (v *helpView) body(a *App, w, h int) []string {
	lines := v.lines(a, w)
	return lines[min(v.off, len(lines)-1):]
}

func (v *helpView) lines(a *App, w int) []string {
	lines := []string{""}
	group := ""
	for _, act := range keys.Actions {
		if act.Group != group {
			if group != "" {
				lines = append(lines, "")
			}
			group = act.Group
			lines = append(lines, " "+heading(group, w-2))
		}
		lines = append(lines, " "+sBold.Render(fit(a.keys.Labels(act.ID), 14))+sDim.Render(act.Help))
	}
	return append(lines, "", " "+sDim.Render("Change any of these in "+home(keys.Path())+", or run `argus keys` to see the format."))
}
