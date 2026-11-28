package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

type review struct{}

func (v *review) update(a *App, msg tea.Msg) tea.Cmd { return nil }

func (v *review) view(a *App, w, h int) []string {
	return empty("Nothing to review", w, h)
}

func (v *review) hints(a *App) string { return a.hints("help", "keys", "quit", "quit") }
