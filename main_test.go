package main

import (
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
