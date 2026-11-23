package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.hcl")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadConfigKeepsFileOrder(t *testing.T) {
	cfg, err := LoadConfig(writeConfig(t, `
tag "blocked"   { color = "#f38ba8" }
tag "in-review" { color = "#fab" }
`))
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if len(cfg.Tags) != 2 {
		t.Fatalf("Tags = %v, want 2 entries", cfg.Tags)
	}
	if cfg.Tags[0].Name != "blocked" || cfg.Tags[0].Color != "#f38ba8" {
		t.Errorf("Tags[0] = %+v", cfg.Tags[0])
	}
	if cfg.Tags[1].Name != "in-review" || cfg.Tags[1].Color != "#fab" {
		t.Errorf("Tags[1] = %+v", cfg.Tags[1])
	}
}

func TestLoadConfigMissingFileUsesDefaults(t *testing.T) {
	cfg, err := LoadConfig(filepath.Join(t.TempDir(), "config.hcl"))
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if len(cfg.Tags) == 0 {
		t.Fatal("no default tags")
	}
	if cfg.Tags[0].Name != "in-review" {
		t.Errorf("first default tag = %q, want in-review", cfg.Tags[0].Name)
	}
}

func TestDefaultConfigParsesToDefaults(t *testing.T) {
	fromFile, err := LoadConfig(writeConfig(t, DefaultConfigHCL))
	if err != nil {
		t.Fatalf("default config does not parse: %v", err)
	}
	fromMissing, _ := LoadConfig(filepath.Join(t.TempDir(), "config.hcl"))
	if len(fromFile.Tags) != len(fromMissing.Tags) {
		t.Fatalf("file gives %d tags, missing file gives %d", len(fromFile.Tags), len(fromMissing.Tags))
	}
	for i := range fromFile.Tags {
		if fromFile.Tags[i] != fromMissing.Tags[i] {
			t.Errorf("tag %d: file %+v, missing %+v", i, fromFile.Tags[i], fromMissing.Tags[i])
		}
	}
}

func TestLoadConfigRejectsBadInput(t *testing.T) {
	cases := map[string]string{
		"bad color":      `tag "wip" { color = "red" }`,
		"short hex":      `tag "wip" { color = "#ab" }`,
		"space in name":  `tag "in review" { color = "#fff" }`,
		"empty name":     `tag "" { color = "#fff" }`,
		"too long":       `tag "` + strings.Repeat("x", 29) + `" { color = "#fff" }`,
		"duplicate":      "tag \"wip\" { color = \"#fff\" }\ntag \"wip\" { color = \"#000\" }",
		"no tags at all": `# nothing here`,
		"unknown attr":   `tag "wip" { color = "#fff" icon = "x" }`,
		"missing color":  `tag "wip" { }`,
		"syntax error":   `tag "wip" {`,
	}
	for name, content := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := LoadConfig(writeConfig(t, content)); err == nil {
				t.Errorf("LoadConfig accepted %q", content)
			}
		})
	}
}

func TestTagToken(t *testing.T) {
	if got := TokenName("in-review"); got != "tag_in-review" {
		t.Errorf("TokenName = %q", got)
	}
	if got := TagLabel("in-review"); got != "#in-review" {
		t.Errorf("TagLabel = %q", got)
	}
}
