package main

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func threeTags() *Config {
	return &Config{Tags: []Tag{
		{Name: "in-review", Color: "#f9e2af"},
		{Name: "blocked", Color: "#f38ba8"},
		{Name: "wip", Color: "#89b4fa"},
	}}
}

func TestTokenPatchesSetsSelectedAndClearsTheRest(t *testing.T) {
	got := TokenPatches(threeTags(), []string{"blocked", "removed-from-config"})
	want := []map[string]any{{
		"tag_in-review": nil,
		"tag_blocked":   "#blocked",
		"tag_wip":       nil,
	}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("TokenPatches = %v, want %v", got, want)
	}
}

func TestTokenPatchesChunksAtSixteenKeys(t *testing.T) {
	cfg := &Config{}
	for i := 0; i < 20; i++ {
		cfg.Tags = append(cfg.Tags, Tag{Name: fmt.Sprintf("t%d", i), Color: "#fff"})
	}
	got := TokenPatches(cfg, nil)
	if len(got) != 2 || len(got[0]) != 16 || len(got[1]) != 4 {
		t.Fatalf("chunk sizes = %v", chunkSizes(got))
	}
	if _, ok := got[1]["tag_t19"]; !ok {
		t.Error("last tag missing from the second chunk")
	}
}

func chunkSizes(chunks []map[string]any) []int {
	var sizes []int
	for _, c := range chunks {
		sizes = append(sizes, len(c))
	}
	return sizes
}

func TestSidebarRowsTOML(t *testing.T) {
	out := SidebarRowsTOML(threeTags())
	for _, want := range []string{
		"[ui.sidebar.spaces]",
		`["state_icon", "workspace"]`,
		`["branch", "git_status"]`,
		`{ token = "$tag_in-review", fg = "#f9e2af" }`,
		`{ token = "$tag_blocked", fg = "#f38ba8" }`,
		`{ token = "$tag_wip", fg = "#89b4fa" }`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("snippet lacks %q:\n%s", want, out)
		}
	}
	if strings.Index(out, "tag_in-review") > strings.Index(out, "tag_blocked") {
		t.Error("tokens are not in config order")
	}
	if !strings.HasSuffix(out, "\n") {
		t.Error("snippet does not end with a newline")
	}
}
