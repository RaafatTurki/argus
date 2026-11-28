package keys

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/RaafatTurki/argus/internal/config"
)

type Action struct {
	ID      string
	Group   string
	Help    string
	Default []string
}

var Actions = []Action{
	{"up", "Move", "Up", []string{"up", "k"}},
	{"down", "Move", "Down", []string{"down", "j"}},
	{"next-view", "Views", "Next view", []string{"tab", "right", "l"}},
	{"prev-view", "Views", "Previous view", []string{"shift+tab", "left", "h"}},
	{"view-index", "Views", "Go to Index", []string{"1"}},
	{"view-review", "Views", "Go to Review", []string{"2"}},
	{"view-config", "Views", "Go to Config", []string{"3"}},
	{"confirm", "Everywhere", "Open / confirm", []string{"enter"}},
	{"back", "Everywhere", "Close popup / clear", []string{"esc"}},
	{"help", "Everywhere", "Show this help", []string{"?"}},
	{"quit", "Everywhere", "Quit argus", []string{"q"}},
	{"edit", "Config", "Edit the selected text setting", []string{"e"}},
	{"editor", "Config", "Open config.toml in your editor", []string{"E"}},
	{"reset", "Config", "Reset every setting to its default", []string{"d"}},
}

type Map struct {
	bind     map[string][]string
	Problems []string
}

func Path() string { return filepath.Join(config.Dir(), "keys.conf") }

func Load() *Map {
	m := &Map{bind: map[string][]string{}}
	known := map[string]bool{}
	for _, a := range Actions {
		m.bind[a.ID] = a.Default
		known[a.ID] = true
	}

	data, err := os.ReadFile(Path())
	if err != nil {
		return m
	}

	n := 0
	for line := range strings.SplitSeq(string(data), "\n") {
		n++
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		id, list, ok := strings.Cut(line, "=")
		id = strings.TrimSpace(id)
		if !ok || !known[id] {
			m.Problems = append(m.Problems, fmt.Sprintf("keys.conf line %d: unknown action %q", n, id))
			continue
		}

		var ks []string
		for k := range strings.SplitSeq(list, ",") {
			if k = strings.TrimSpace(k); k == "space" {
				k = " "
			} else if k == "" {
				continue
			}
			ks = append(ks, k)
		}
		m.bind[id] = ks
	}
	return m
}

func (m *Map) Is(id, key string) bool {
	return slices.Contains(m.bind[id], key)
}

func (m *Map) Any(key string, ids ...string) bool {
	for _, id := range ids {
		if m.Is(id, key) {
			return true
		}
	}
	return false
}

func (m *Map) Label(id string) string {
	ks := m.bind[id]
	if len(ks) == 0 {
		return ""
	}
	return Pretty(ks[0])
}

func (m *Map) Labels(id string) string {
	var out []string
	for _, k := range m.bind[id] {
		out = append(out, Pretty(k))
	}
	if len(out) == 0 {
		return "unbound"
	}
	return strings.Join(out, "  ")
}

var pretty = map[string]string{
	"enter": "↵", "up": "↑", "down": "↓", "left": "←", "right": "→", " ": "space", "pgup": "PgUp", "pgdown": "PgDn",
	"shift+tab": "⇧tab", "backspace": "⌫", "delete": "del",
}

func Pretty(k string) string {
	if p, ok := pretty[k]; ok {
		return p
	}
	if rest, ok := strings.CutPrefix(k, "ctrl+"); ok {
		return "^" + strings.ToUpper(rest)
	}
	return k
}

func (m *Map) Dump() string {
	var b strings.Builder
	b.WriteString("# argus key bindings: action = key, key, …   (use \"space\" for the space bar)\n")
	fmt.Fprintf(&b, "# Save as %s\n", Path())

	group := ""
	for _, a := range Actions {
		if a.Group != group {
			group = a.Group
			fmt.Fprintf(&b, "\n# %s\n", group)
		}

		ks := make([]string, len(m.bind[a.ID]))
		for i, k := range m.bind[a.ID] {
			ks[i] = k
			if k == " " {
				ks[i] = "space"
			}
		}
		fmt.Fprintf(&b, "%-14s = %-18s # %s\n", a.ID, strings.Join(ks, ", "), a.Help)
	}
	return b.String()
}
