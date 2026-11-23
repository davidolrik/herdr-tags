package main

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsimple"
)

// DefaultConfigHCL is the documented config file written by `init-config`.
// Its tags are also what the plugin uses while no config file exists, so the
// two must agree: a test parses this text and compares it to the defaults.
const DefaultConfigHCL = `# herdr-tags configuration
#
# One block per tag, in the order the picker lists them and the sidebar
# shows them. Add, remove, or recolor freely; the plugin reads this file on
# every invocation, so changes apply the next time you open the picker.
#
# Names become herdr sidebar tokens ("in-review" is shown as "#in-review"
# through the $t1_in-review token, $t2_in-review on the second line, and so
# on), so they may use ASCII letters, digits, underscores and hyphens only,
# up to 28 characters.
#
# Colors are strict #RGB or #RRGGBB hex, the only form herdr's sidebar
# accepts. After editing, regenerate the sidebar rows for config.toml:
#
#   herdr plugin action invoke davidolrik.tags.sidebar-rows
#
# The defaults are Catppuccin Mocha accents.
#
# Tags that do not fit on one sidebar line wrap onto the next. The plugin
# reads the sidebar width from the herdr client on this machine; a client
# attached with "herdr --remote" keeps its width where the plugin cannot see
# it. If such a client has a narrower sidebar, give its width here and tags
# are wrapped to fit the narrower of the two:
#
# narrowest_sidebar_width = 24
#
# A workspace can also carry a free-text note, typed into the field below
# the tags in the picker and shown below the tags in the sidebar. It wraps
# onto at most note_lines sidebar lines and anything longer ends in "…";
# 0 turns the note off. Regenerate the sidebar rows after changing either:
#
# note_lines = 2
# note_color = "#9399b2"

tag "in-review" { color = "#f9e2af" }
tag "blocked"   { color = "#f38ba8" }
tag "wip"       { color = "#89b4fa" }
tag "ready"     { color = "#a6e3a1" }
tag "on-hold"   { color = "#9399b2" }
`

// Tag is one configured tag: its name, and the color it is drawn in.
type Tag struct {
	Name  string
	Color string
}

// Config is the fully-resolved plugin configuration.
type Config struct {
	Tags []Tag
	// NarrowestSidebarWidth caps the sidebar width tags are wrapped for,
	// covering clients whose width the plugin cannot read. Zero means unset.
	NarrowestSidebarWidth int
	// NoteLines is how many sidebar lines a workspace's note may fill; zero
	// turns the note off.
	NoteLines int
	// NoteColor is the color the note is drawn in.
	NoteColor string
}

// What the note gets when the config does not say.
const (
	defaultNoteLines = 2
	defaultNoteColor = "#9399b2"
)

type rawConfig struct {
	NarrowestSidebarWidth *int     `hcl:"narrowest_sidebar_width,optional"`
	NoteLines             *int     `hcl:"note_lines,optional"`
	NoteColor             *string  `hcl:"note_color,optional"`
	Tags                  []rawTag `hcl:"tag,block"`
}

type rawTag struct {
	Name  string `hcl:"name,label"`
	Color string `hcl:"color"`
}

// Token names are limited to 32 characters of ASCII letters, digits,
// underscores and hyphens by herdr; the line prefix ("t1_" to "t14_") takes
// up to four of them.

var (
	tagNameRe = regexp.MustCompile(`^[A-Za-z0-9_-]{1,28}$`)
	colorRe   = regexp.MustCompile(`^#([0-9a-fA-F]{3}|[0-9a-fA-F]{6})$`)
)

// TokenName is the herdr workspace metadata token that carries a tag on one
// sidebar line, counted from 1. Herdr lays tokens out in fixed rows, so a tag
// has one token per line it may wrap onto and only that line's token is set.
func TokenName(line int, tag string) string {
	return fmt.Sprintf("t%d_%s", line, tag)
}

// TagLabel is the text shown in the sidebar for a set tag.
func TagLabel(tag string) string {
	return "#" + tag
}

// LoadConfig reads and validates the HCL config at path. A missing file
// yields the default tag set; anything else that goes wrong is an error, so
// a typo never silently empties the picker.
func LoadConfig(path string) (*Config, error) {
	src, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		src = []byte(DefaultConfigHCL)
	} else if err != nil {
		return nil, err
	}
	return parseConfig(path, src)
}

func parseConfig(filename string, src []byte) (*Config, error) {
	var raw rawConfig
	if err := hclsimple.Decode(filename, src, nil, &raw); err != nil {
		var diags hcl.Diagnostics
		if errors.As(err, &diags) {
			return nil, fmt.Errorf("%s: %s", filename, strings.Join(diagMessages(diags), "; "))
		}
		return nil, err
	}
	if len(raw.Tags) == 0 {
		return nil, fmt.Errorf("%s: no tag blocks defined", filename)
	}

	cfg := &Config{NoteLines: defaultNoteLines, NoteColor: defaultNoteColor}
	if raw.NoteLines != nil {
		// The tags keep at least one of the sidebar rows.
		if *raw.NoteLines < 0 || *raw.NoteLines >= maxSidebarRows {
			return nil, fmt.Errorf("%s: note_lines must be between 0 and %d", filename, maxSidebarRows-1)
		}
		cfg.NoteLines = *raw.NoteLines
	}
	if raw.NoteColor != nil {
		if !colorRe.MatchString(*raw.NoteColor) {
			return nil, fmt.Errorf("%s: note_color %q must be #RGB or #RRGGBB", filename, *raw.NoteColor)
		}
		cfg.NoteColor = *raw.NoteColor
	}
	if raw.NarrowestSidebarWidth != nil {
		if *raw.NarrowestSidebarWidth < 1 {
			return nil, fmt.Errorf("%s: narrowest_sidebar_width must be a positive number of columns", filename)
		}
		cfg.NarrowestSidebarWidth = *raw.NarrowestSidebarWidth
	}
	seen := map[string]bool{}
	for _, t := range raw.Tags {
		if !tagNameRe.MatchString(t.Name) {
			return nil, fmt.Errorf("%s: tag %q: name must be 1-28 ASCII letters, digits, underscores or hyphens", filename, t.Name)
		}
		if !colorRe.MatchString(t.Color) {
			return nil, fmt.Errorf("%s: tag %q: color %q must be #RGB or #RRGGBB", filename, t.Name, t.Color)
		}
		if seen[t.Name] {
			return nil, fmt.Errorf("%s: tag %q defined twice", filename, t.Name)
		}
		seen[t.Name] = true
		cfg.Tags = append(cfg.Tags, Tag{Name: t.Name, Color: t.Color})
	}
	return cfg, nil
}

func diagMessages(diags hcl.Diagnostics) []string {
	var msgs []string
	for _, d := range diags {
		if d.Severity != hcl.DiagError {
			continue
		}
		msg := d.Summary
		if d.Detail != "" {
			msg += ": " + d.Detail
		}
		if d.Subject != nil {
			msg = fmt.Sprintf("%d:%d: %s", d.Subject.Start.Line, d.Subject.Start.Column, msg)
		}
		msgs = append(msgs, msg)
	}
	return msgs
}
