package config

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Editor   string `toml:"editor"`
	KeyHints bool   `toml:"key_hints"`
	LLMURL   string `toml:"llm_url"`

	Problems []string `toml:"-"`

	broken error
}

func Defaults() *Config {
	return &Config{KeyHints: true, LLMURL: "http://localhost:8080"}
}

func Dir() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "argus")
}

func Path() string { return filepath.Join(Dir(), "config.toml") }

func Load() *Config {
	c := Defaults()

	meta, err := toml.DecodeFile(Path(), c)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return c
	case err != nil:
		c = Defaults()
		c.broken = fmt.Errorf("config.toml couldn't be read, so the defaults are used: %w", err)
		c.Problems = []string{c.broken.Error()}
		return c
	}

	for _, key := range meta.Undecoded() {
		c.Problems = append(c.Problems, fmt.Sprintf("config.toml: unknown setting %q", key.String()))
	}
	return c
}

func (c *Config) Broken() bool { return c.broken != nil }

func (c *Config) Save() error {
	if c.broken != nil {
		return c.broken
	}

	var b bytes.Buffer
	if err := toml.NewEncoder(&b).Encode(c); err != nil {
		return err
	}

	path := Path()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	f, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())

	_, err = f.Write(b.Bytes())
	if err := errors.Join(err, f.Close()); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}

func (c *Config) Reset() error {
	*c = *Defaults()
	return c.Save()
}

func (c *Config) EditorCmd() []string {
	for _, e := range []string{c.Editor, os.Getenv("VISUAL"), os.Getenv("EDITOR"), "vi"} {
		if f := strings.Fields(e); len(f) > 0 {
			return f
		}
	}
	return nil
}
