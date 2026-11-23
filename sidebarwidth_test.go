package main

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestClientPreferencesPathHashesTheClientSocket(t *testing.T) {
	got := ClientPreferencesPath("/state/herdr", "/Users/djo/.config/herdr/sessions/Private/herdr.sock")
	want := "/state/herdr/client-shell/local-533df154568f49f1.json"
	if got != want {
		t.Errorf("ClientPreferencesPath = %q, want %q", got, want)
	}
}

func TestSidebarWidth(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "nope")
	cases := []struct {
		name   string
		config string
		prefs  string
		want   int
	}{
		{"herdr's default without any file", "", "", 26},
		{"width from config.toml", "[ui]\nsidebar_width = 30\n", "", 30},
		{"dragged width wins over config.toml", "[ui]\nsidebar_width = 30\n", `{"sidebar_width": 22}`, 22},
		{"preferences without a width", "[ui]\nsidebar_width = 30\n", `{"sidebar_collapsed": false}`, 30},
		{"clamped to the default bounds", "", `{"sidebar_width": 50}`, 36},
		{"clamped to configured bounds", "[ui]\nsidebar_min_width = 24\nsidebar_max_width = 40\n", `{"sidebar_width": 20}`, 24},
		{"inverted bounds fall back to the defaults", "[ui]\nsidebar_min_width = 50\nsidebar_max_width = 30\n", `{"sidebar_width": 10}`, 18},
		{"unreadable config.toml", "[ui\nsidebar_width = ", "", 26},
		{"unreadable preferences", "[ui]\nsidebar_width = 30\n", "{", 30},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			configPath, prefsPath := missing, missing
			if c.config != "" {
				configPath = writeFile(t, "config.toml", c.config)
			}
			if c.prefs != "" {
				prefsPath = writeFile(t, "prefs.json", c.prefs)
			}
			if got := SidebarWidth(configPath, prefsPath); got != c.want {
				t.Errorf("SidebarWidth = %d, want %d", got, c.want)
			}
		})
	}
}

func TestTagColumns(t *testing.T) {
	cfg := threeTags()
	if got := TagColumns(cfg, 28, false); got != 21 {
		t.Errorf("top-level workspace at width 28: %d columns, want 21", got)
	}
	if got := TagColumns(cfg, 28, true); got != 16 {
		t.Errorf("indented workspace at width 28: %d columns, want 16", got)
	}
	cfg.NarrowestSidebarWidth = 24
	if got := TagColumns(cfg, 28, false); got != 17 {
		t.Errorf("narrowest_sidebar_width below the sidebar: %d columns, want 17", got)
	}
	cfg.NarrowestSidebarWidth = 34
	if got := TagColumns(cfg, 28, false); got != 21 {
		t.Errorf("narrowest_sidebar_width above the sidebar: %d columns, want 21", got)
	}
}
