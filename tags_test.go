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

func TestPackLines(t *testing.T) {
	cfg := threeTags()
	cases := []struct {
		name     string
		selected []string
		columns  int
		want     [][]string
	}{
		{"everything fits on one line", []string{"in-review", "blocked", "wip"}, 28, [][]string{{"in-review", "blocked", "wip"}}},
		{"one column short wraps the last tag", []string{"in-review", "blocked", "wip"}, 27, [][]string{{"in-review", "blocked"}, {"wip"}}},
		{"config order, not selection order", []string{"wip", "in-review"}, 40, [][]string{{"in-review", "wip"}}},
		{"a tag wider than the line stands alone", []string{"in-review", "blocked", "wip"}, 5, [][]string{{"in-review"}, {"blocked"}, {"wip"}}},
		{"tags missing from the config are ignored", []string{"blocked", "removed-from-config"}, 40, [][]string{{"blocked"}}},
		{"nothing selected", nil, 40, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := PackLines(cfg, c.selected, c.columns); !reflect.DeepEqual(got, c.want) {
				t.Errorf("PackLines = %v, want %v", got, c.want)
			}
		})
	}
}

func TestPackLinesPutsTheOverflowOnTheLastLine(t *testing.T) {
	cfg := &Config{}
	var selected []string
	for i := 0; i < maxTagLines+3; i++ {
		name := fmt.Sprintf("tag%02d", i)
		cfg.Tags = append(cfg.Tags, Tag{Name: name, Color: "#fff"})
		selected = append(selected, name)
	}
	lines := PackLines(cfg, selected, 6)
	if len(lines) != maxTagLines {
		t.Fatalf("%d lines, want %d", len(lines), maxTagLines)
	}
	if got := len(lines[maxTagLines-1]); got != 4 {
		t.Errorf("last line holds %d tags, want 4", got)
	}
}

func TestTokenPatchesSetsSelectedAndClearsTheRest(t *testing.T) {
	got := TokenPatches(threeTags(), []string{"blocked", "wip", "removed-from-config"}, 8)
	want := []map[string]any{{
		"t1_in-review": nil,
		"t1_blocked":   "#blocked",
		"t1_wip":       nil,
		"t2_in-review": nil,
		"t2_blocked":   nil,
		"t2_wip":       "#wip",
		"t3_in-review": nil,
		"t3_blocked":   nil,
		"t3_wip":       nil,
	}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("TokenPatches = %v, want %v", got, want)
	}
}

func TestTokenPatchesChunksAtSixteenKeys(t *testing.T) {
	cfg := &Config{}
	for i := 0; i < 5; i++ {
		cfg.Tags = append(cfg.Tags, Tag{Name: fmt.Sprintf("x%d", i), Color: "#fff"})
	}
	got := TokenPatches(cfg, nil, 40)
	if len(got) != 2 || len(got[0]) != 16 || len(got[1]) != 9 {
		t.Fatalf("chunk sizes = %v", chunkSizes(got))
	}
	if _, ok := got[1]["t5_x4"]; !ok {
		t.Error("last token missing from the second chunk")
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
		`{ token = "$t1_in-review", fg = "#f9e2af" }`,
		`{ token = "$t1_blocked", fg = "#f38ba8" }`,
		`{ token = "$t1_wip", fg = "#89b4fa" }`,
		`{ token = "$t3_wip", fg = "#89b4fa" }`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("snippet lacks %q:\n%s", want, out)
		}
	}
	if strings.Index(out, "t1_in-review") > strings.Index(out, "t1_blocked") {
		t.Error("tokens are not in config order")
	}
	if strings.Index(out, "t1_wip") > strings.Index(out, "t2_in-review") {
		t.Error("lines are not in order")
	}
	if strings.Contains(out, "$t4_") {
		t.Error("more tag lines than tags")
	}
	if !strings.HasSuffix(out, "\n") {
		t.Error("snippet does not end with a newline")
	}
}
