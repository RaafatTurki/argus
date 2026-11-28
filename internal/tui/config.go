package tui

import (
	"cmp"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/RaafatTurki/argus/internal/config"
)

type choice struct{ value, label string }

type setting struct {
	group, label, help string
	text               bool
	choices            []choice
	get                func(*config.Config) string
	set                func(*config.Config, string)
}

var shownHidden = []choice{{"true", "Shown"}, {"false", "Hidden"}}

func toggle(group, label, help string, field func(*config.Config) *bool) setting {
	return setting{
		group: group, label: label, help: help, choices: shownHidden,
		get: func(c *config.Config) string { return strconv.FormatBool(*field(c)) },
		set: func(c *config.Config, v string) { *field(c) = v == "true" },
	}
}

var settings = []setting{
	{
		group: "General", label: "Editor", text: true,
		help: "The command that opens config.toml (E). It takes over this terminal while you edit, e.g. nvim. Empty uses $VISUAL, then $EDITOR, then vi.",
		get:  func(c *config.Config) string { return c.Editor },
		set:  func(c *config.Config, v string) { c.Editor = strings.TrimSpace(v) },
	},

	toggle("Show", "Key hints", "The line of keys at the bottom of the screen. ? still lists every key.", func(c *config.Config) *bool { return &c.KeyHints }),
}

type configView struct {
	cur, off int
	editing  bool
	input    textinput.Model
}

type editedMsg struct{ err error }

func newConfigView() *configView {
	ti := textinput.New()
	ti.Prompt = ""
	return &configView{input: ti}
}

func (v *configView) typing() bool { return v.editing }

func (v *configView) hints(a *App) string {
	if v.editing {
		return a.hints("confirm", "accept", "back", "cancel")
	}
	return a.hints("confirm", "change", "editor", "open in editor", "reset", "reset all", "help", "keys")
}

func (v *configView) display(a *App, s setting) string {
	cur := s.get(a.cfg)
	if s.text {
		return cmp.Or(cur, "default")
	}
	for _, c := range s.choices {
		if c.value == cur {
			return c.label
		}
	}
	return cur
}

func (a *App) saveConfig() tea.Cmd {
	if err := a.cfg.Save(); err != nil {
		return a.complain("Couldn't save the config: %v", err)
	}
	return nil
}

func (v *configView) cycle(a *App, s setting) tea.Cmd {
	cur := 0
	for i, c := range s.choices {
		if c.value == s.get(a.cfg) {
			cur = i
		}
	}
	s.set(a.cfg, s.choices[(cur+1)%len(s.choices)].value)
	return a.saveConfig()
}

func (a *App) openEditor() tea.Cmd {
	if _, err := os.Stat(config.Path()); errors.Is(err, fs.ErrNotExist) {
		if cmd := a.saveConfig(); cmd != nil {
			return cmd
		}
	}

	argv := append(a.cfg.EditorCmd(), config.Path())
	return tea.ExecProcess(exec.Command(argv[0], argv[1:]...), func(err error) tea.Msg { return editedMsg{err} })
}

func (v *configView) update(a *App, msg tea.Msg) tea.Cmd {
	if msg, ok := msg.(editedMsg); ok {
		if msg.err != nil {
			return a.complain("Couldn't run the editor: %v", msg.err)
		}
		a.cfg = config.Load()
		if len(a.cfg.Problems) > 0 {
			return a.complain("%s", a.cfg.Problems[0])
		}
		return a.say("Config reloaded")
	}

	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return nil
	}

	s := settings[v.cur]
	if v.editing {
		switch {
		case a.is(key, "confirm"):
			s.set(a.cfg, v.input.Value())
			v.editing = false
			return a.saveConfig()
		case a.is(key, "back"):
			v.editing = false
			return nil
		}
		var cmd tea.Cmd
		v.input, cmd = v.input.Update(msg)
		return cmd
	}

	switch {
	case a.is(key, "up"):
		v.cur = max(0, v.cur-1)
	case a.is(key, "down"):
		v.cur = min(len(settings)-1, v.cur+1)
	case a.is(key, "editor"):
		return a.openEditor()
	case a.is(key, "reset"):
		a.push(&confirm{
			heading: "Reset the config?",
			text:    "Every setting goes back to its default and config.toml is rewritten, even if it couldn't be read.",
			yes: func(a *App) tea.Cmd {
				if err := a.cfg.Reset(); err != nil {
					return a.complain("Couldn't save the config: %v", err)
				}
				return a.say("Config reset to defaults")
			},
		})
	case a.is(key, "confirm"), a.is(key, "edit") && s.text:
		if !s.text {
			return v.cycle(a, s)
		}
		v.editing = true
		v.input.SetValue(s.get(a.cfg))
		v.input.CursorEnd()
		return v.input.Focus()
	}
	return nil
}

func (v *configView) view(a *App, w, h int) []string {
	const labelW, valW = 22, 16

	var rows []string
	selRow, group := 0, ""
	for i, s := range settings {
		if s.group != group {
			group = s.group
			if len(rows) > 0 {
				rows = append(rows, "")
			}
			rows = append(rows, " "+heading(group, w-2))
		}

		if i == v.cur {
			selRow = len(rows)
		}

		val := v.display(a, s)
		switch {
		case i == v.cur && v.editing:
			v.input.Width = max(1, w-labelW-4)
			rows = append(rows, " "+sBold.Render(fit(s.label, labelW))+" "+v.input.View())
		case i == v.cur:
			rows = append(rows, sSel.Bold(true).Render(" "+fit(s.label, labelW)+" "+fit(val, valW))+sSel.Render(strings.Repeat(" ", max(0, w-labelW-valW-3))))
		default:
			rows = append(rows, " "+sDim.Render(fit(s.label, labelW))+" "+fit(val, valW))
		}
	}

	cur := settings[v.cur]
	foot := []string{"", sEdge.Render(strings.Repeat("─", w)), " " + sBold.Render(cur.label)}
	for _, l := range wrap(cur.help, w-4) {
		foot = append(foot, "  "+sDim.Render(l))
	}

	if len(a.cfg.Problems) > 0 {
		foot = append(foot, "")
		for _, p := range a.cfg.Problems {
			foot = append(foot, " "+sBad.Render(clip(p, w-2)))
		}
	}

	saved := "Saved to " + home(config.Path()) + " as you change it."
	if a.cfg.Broken() {
		saved = "Not saved until " + home(config.Path()) + " is fixed (E) or reset (d)."
	}
	foot = append(foot, "  "+sDim.Render(saved))

	room := max(3, h-1-len(foot))
	return append(append([]string{""}, windowRows(rows, selRow, room, &v.off)...), foot...)
}
