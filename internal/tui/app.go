package tui

import (
	"cmp"
	"fmt"
	"os"
	"os/signal"
	"slices"
	"strings"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/RaafatTurki/argus/internal/config"
	"github.com/RaafatTurki/argus/internal/keys"
)

type tabID int

var (
	tabNames = [...]string{"Index", "Review", "Config"}
	tabKeys  = [...]string{"view-index", "view-review", "view-config"}
)

type page interface {
	update(a *App, msg tea.Msg) tea.Cmd
	view(a *App, w, h int) []string
	hints(a *App) string
	typing() bool
}

type App struct {
	keys *keys.Map
	cfg  *config.Config

	w, h   int
	tab    tabID
	pages  [len(tabNames)]page
	popups []popup

	status status
}

type status struct {
	text string
	bad  bool
	at   time.Time
}

type expireMsg time.Time

func Run(km *keys.Map, cfg *config.Config) error {
	a := &App{keys: km, cfg: cfg, pages: [...]page{&index{}, &review{}, newConfigView()}}

	p := tea.NewProgram(a, tea.WithAltScreen())

	hup := make(chan os.Signal, 1)
	signal.Notify(hup, syscall.SIGHUP)
	go func() {
		<-hup
		p.Quit()
	}()

	_, err := p.Run()
	signal.Stop(hup)
	return err
}

func (a *App) Init() tea.Cmd {
	if problems := slices.Concat(a.keys.Problems, a.cfg.Problems); len(problems) > 0 {
		return a.complain("%s", problems[0])
	}
	return nil
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return a, a.update(msg) }

func (a *App) say(format string, args ...any) tea.Cmd {
	return a.show(status{text: fmt.Sprintf(format, args...), at: time.Now()})
}

func (a *App) complain(format string, args ...any) tea.Cmd {
	return a.show(status{text: fmt.Sprintf(format, args...), bad: true, at: time.Now()})
}

func (a *App) show(s status) tea.Cmd {
	a.status = s
	return tea.Tick(6*time.Second, func(time.Time) tea.Msg { return expireMsg(s.at) })
}

func (a *App) push(p popup) { a.popups = append(a.popups, p) }

func (a *App) is(msg tea.KeyMsg, actions ...string) bool { return a.keys.Any(msg.String(), actions...) }

func (a *App) hints(pairs ...string) string {
	var parts []string
	for i := 0; i+1 < len(pairs); i += 2 {
		key := pairs[i]
		if l := a.keys.Label(key); l != "" {
			key = l
		}
		parts = append(parts, sBold.Render(key)+" "+sDim.Render(pairs[i+1]))
	}
	return strings.Join(parts, sDim.Render("  ·  "))
}

func (a *App) page() page { return a.pages[a.tab] }

func (a *App) update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.w, a.h = msg.Width, msg.Height
		return nil
	case expireMsg:
		if a.status.at.Equal(time.Time(msg)) {
			a.status = status{}
		}
		return nil
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return tea.Quit
		}
	}

	if len(a.popups) > 0 {
		return a.updatePopups(msg)
	}

	key, ok := msg.(tea.KeyMsg)
	if !ok {
		var cmds []tea.Cmd
		for _, p := range a.pages {
			cmds = append(cmds, p.update(a, msg))
		}
		return tea.Batch(cmds...)
	}

	p := a.page()
	switch {
	case a.is(key, "next-view") && !(p.typing() && key.String() != "tab"):
		a.tab = (a.tab + 1) % tabID(len(tabNames))
		return nil
	case a.is(key, "prev-view") && !(p.typing() && key.String() != "shift+tab"):
		a.tab = (a.tab + tabID(len(tabNames)) - 1) % tabID(len(tabNames))
		return nil
	case p.typing():
		return p.update(a, msg)
	case a.is(key, "quit"):
		return tea.Quit
	case a.is(key, "help"):
		a.push(&helpView{})
		return nil
	}

	for i, id := range tabKeys {
		if a.is(key, id) {
			a.tab = tabID(i)
			return nil
		}
	}
	return p.update(a, msg)
}

func (a *App) updatePopups(msg tea.Msg) tea.Cmd {
	n := len(a.popups)
	top, cmd := a.popups[n-1].update(a, msg)
	if top == nil {
		a.popups = a.popups[:n-1]
	} else {
		a.popups[n-1] = top
	}
	return cmd
}

const minWidth, minHeight = 40, 10

func (a *App) View() string {
	switch {
	case a.w == 0:
		return ""
	case a.w < minWidth || a.h < minHeight:
		return a.tooSmall()
	}

	bodyH := a.h - 3
	lines := a.page().view(a, a.w, bodyH)
	for len(lines) < bodyH {
		lines = append(lines, "")
	}

	header, rule := a.tabBar()
	screen := append([]string{header, rule}, lines[:bodyH]...)
	screen = append(screen, a.footer())

	if n := len(a.popups); n > 0 {
		p := a.popups[n-1]
		pw, ph := a.popupSize(p)
		screen = composite(screen, box(p.title(a), p.hints(a), p.body(a, pw-4, ph-2), pw, ph), a.w)
	}
	return strings.Join(screen, "\n")
}

func (a *App) popupSize(p popup) (int, int) {
	w, h := p.size(a)
	return min(w, a.w-2), min(h, a.h-4)
}

func (a *App) popupBody(p popup) (int, int) {
	w, h := a.popupSize(p)
	return w - 4, h - 2
}

func (a *App) tooSmall() string {
	msg := []string{clip(sBold.Render("Not enough screen space"), a.w)}
	if a.h >= 3 {
		msg = append(msg, clip(sDim.Render(fmt.Sprintf("make it at least %d×%d", minWidth, minHeight)), a.w))
	}
	return lipgloss.Place(a.w, a.h, lipgloss.Center, lipgloss.Center, lipgloss.JoinVertical(lipgloss.Center, msg...))
}

func (a *App) tabBar() (header, rule string) {
	layouts := []struct {
		brand, short bool
		gap          int
	}{{true, false, 3}, {false, false, 2}, {false, true, 2}}

	for n, l := range layouts {
		x, start, end := 1, 0, 0
		head := " "
		if l.brand {
			brand := sAccent.Bold(true).Render("argus") + "   "
			head += brand
			x += width(brand)
		}

		for i, name := range tabNames {
			if i > 0 {
				head += strings.Repeat(" ", l.gap)
				x += l.gap
			}

			if l.short && tabID(i) != a.tab {
				name = cmp.Or(a.keys.Label(tabKeys[i]), name[:1])
			}

			st := sDim
			if tabID(i) == a.tab {
				st, start, end = sBold, x, x+width(name)
			}
			head += st.Render(name)
			x += width(name)
		}

		if x <= a.w || n == len(layouts)-1 {
			end = min(end, a.w)
			rule = sEdge.Render(strings.Repeat("─", start)) + sAccent.Render(strings.Repeat("━", max(0, end-start))) + sEdge.Render(strings.Repeat("─", max(0, a.w-end)))
			return clip(head, a.w), rule
		}
	}
	return "", ""
}

func (a *App) footer() string {
	hints := " " + a.page().hints(a)
	if len(a.popups) > 0 || !a.cfg.KeyHints {
		hints = ""
	}
	if a.status.text == "" {
		return hints
	}

	st := sGood
	if a.status.bad {
		st = sBad
	}
	return between(hints, st.Render(a.status.text)+" ", a.w)
}
