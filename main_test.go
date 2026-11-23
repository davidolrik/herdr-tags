package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestPruneStateDropsClosedWorkspaces(t *testing.T) {
	st := State{"w1": {"wip"}, "w2": {"blocked"}, "w3": {"ready"}}
	pruned := pruneState(st, []string{"w1", "w3", "w9"})
	want := State{"w1": {"wip"}, "w3": {"ready"}}
	if !reflect.DeepEqual(pruned, want) {
		t.Errorf("pruneState = %v, want %v", pruned, want)
	}
}

func TestWorkspaceIndentedForLinkedWorktrees(t *testing.T) {
	cases := map[string]bool{
		`{"workspace_id":"w1"}`: false,
		`{"workspace_id":"w2","worktree":{"is_linked_worktree":false}}`: false,
		`{"workspace_id":"w3","worktree":{"is_linked_worktree":true}}`:  true,
	}
	for record, want := range cases {
		var ws workspace
		if err := json.Unmarshal([]byte(record), &ws); err != nil {
			t.Fatal(err)
		}
		if got := ws.indented(); got != want {
			t.Errorf("%s: indented = %v, want %v", record, got, want)
		}
	}
}

func TestHerdrPathsFollowHerdrsLookup(t *testing.T) {
	t.Setenv("HOME", "/home/me")
	t.Setenv("HERDR_CONFIG_PATH", "")
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("XDG_STATE_HOME", "")
	if got := herdrConfigPath(); got != "/home/me/.config/herdr/config.toml" {
		t.Errorf("herdrConfigPath = %q", got)
	}
	if got := herdrStateDir(); got != "/home/me/.local/state/herdr" {
		t.Errorf("herdrStateDir = %q", got)
	}
	t.Setenv("XDG_CONFIG_HOME", "/xdg/config")
	t.Setenv("XDG_STATE_HOME", "/xdg/state")
	if got := herdrConfigPath(); got != "/xdg/config/herdr/config.toml" {
		t.Errorf("herdrConfigPath = %q", got)
	}
	if got := herdrStateDir(); got != "/xdg/state/herdr" {
		t.Errorf("herdrStateDir = %q", got)
	}
	t.Setenv("HERDR_CONFIG_PATH", "/elsewhere/herdr.toml")
	if got := herdrConfigPath(); got != "/elsewhere/herdr.toml" {
		t.Errorf("herdrConfigPath = %q", got)
	}
}

func TestPopupSizeFitsTheTagList(t *testing.T) {
	w, h := popupSize(threeTags())
	if h != len(threeTags().Tags)+popupChromeRows {
		t.Errorf("height = %d", h)
	}
	if w < utf8.RuneCountInString("↑↓ move · space toggle · enter save · esc cancel")+4 {
		t.Errorf("width %d is narrower than the hint line", w)
	}
}

func TestWorkspaceIDFromContextJSON(t *testing.T) {
	t.Setenv("HERDR_TAGS_WORKSPACE_ID", "")
	t.Setenv("HERDR_WORKSPACE_ID", "")
	t.Setenv("HERDR_PLUGIN_CONTEXT_JSON", `{"workspace_id":"w7","tab_id":"w7:t1"}`)
	if got := workspaceIDFromEnv(); got != "w7" {
		t.Errorf("workspaceIDFromEnv = %q, want w7 (from context JSON)", got)
	}
	t.Setenv("HERDR_WORKSPACE_ID", "w8")
	if got := workspaceIDFromEnv(); got != "w8" {
		t.Errorf("workspaceIDFromEnv = %q, want w8 (env var wins)", got)
	}
	t.Setenv("HERDR_TAGS_WORKSPACE_ID", "w9")
	if got := workspaceIDFromEnv(); got != "w9" {
		t.Errorf("workspaceIDFromEnv = %q, want w9 (popup hand-off wins)", got)
	}
}

func TestInitConfigNeverOverwrites(t *testing.T) {
	dir := t.TempDir()
	if err := initConfig(dir); err != nil {
		t.Fatalf("initConfig: %v", err)
	}
	path := filepath.Join(dir, "config.hcl")
	data, _ := os.ReadFile(path)
	if string(data) != DefaultConfigHCL {
		t.Error("written config differs from DefaultConfigHCL")
	}
	os.WriteFile(path, []byte("tag \"mine\" { color = \"#000\" }\n"), 0o644)
	if err := initConfig(dir); err == nil {
		t.Error("initConfig overwrote an existing config without complaint")
	}
	data, _ = os.ReadFile(path)
	if !strings.Contains(string(data), "mine") {
		t.Error("existing config was clobbered")
	}
}

func TestExampleConfigMatchesDefault(t *testing.T) {
	data, err := os.ReadFile("config.example.hcl")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != DefaultConfigHCL {
		t.Error("config.example.hcl is out of sync with DefaultConfigHCL")
	}
}
