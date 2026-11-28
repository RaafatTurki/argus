package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type index struct{}

func (v *index) update(a *App, msg tea.Msg) tea.Cmd { return nil }

func (v *index) view(a *App, w, h int) []string {
	return empty("Nothing indexed yet", w, h)
}

func (v *index) typing() bool { return false }

func (v *index) hints(a *App) string { return a.hints("help", "keys", "quit", "quit") }

func empty(text string, w, h int) []string {
	return strings.Split(lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, sDim.Render(text)), "\n")
}
