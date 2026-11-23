package main

import (
	"reflect"
	"strings"
	"testing"
)

func TestWrapNote(t *testing.T) {
	cases := []struct {
		name    string
		note    string
		columns int
		lines   int
		want    []string
	}{
		{"fits on one line", "waiting on review", 20, 2, []string{"waiting on review"}},
		{"wraps at a word boundary", "waiting on review", 12, 2, []string{"waiting on", "review"}},
		{"overflow ends in an ellipsis", "waiting on review from the platform team", 12, 2, []string{"waiting on", "review from…"}},
		{"the ellipsis takes the last cell of a full line", "aaaa bbbb cccc", 4, 2, []string{"aaaa", "bbb…"}},
		{"no space is left before the ellipsis", "abc d ef gh", 5, 1, []string{"abc…"}},
		{"a word wider than the line is broken", "abcdefghij", 4, 3, []string{"abcd", "efgh", "ij"}},
		{"counts characters, not bytes", "søren får ærter", 9, 2, []string{"søren får", "ærter"}},
		{"whitespace is collapsed", "  a \t b\n", 10, 2, []string{"a b"}},
		{"blank note", "   ", 10, 2, nil},
		{"no lines to fill", "note", 10, 0, nil},
		{"no room at all still shows something", "abc", 0, 2, []string{"a", "…"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := WrapNote(c.note, c.columns, c.lines); !reflect.DeepEqual(got, c.want) {
				t.Errorf("WrapNote = %q, want %q", got, c.want)
			}
		})
	}
}

func TestNoteTokensSetsUsedLinesAndClearsTheRest(t *testing.T) {
	cfg := threeTags()
	cfg.NoteLines = 2
	got := NoteTokens(cfg, "waiting on review", 12)
	want := map[string]any{"note1": "waiting on", "note2": "review"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("NoteTokens = %v, want %v", got, want)
	}
	got = NoteTokens(cfg, "short", 12)
	want = map[string]any{"note1": "short", "note2": nil}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("NoteTokens = %v, want %v", got, want)
	}
	got = NoteTokens(cfg, "", 12)
	want = map[string]any{"note1": nil, "note2": nil}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("NoteTokens = %v, want %v", got, want)
	}
}

func TestNoteTokensWithoutNoteLines(t *testing.T) {
	if got := NoteTokens(threeTags(), "anything", 12); len(got) != 0 {
		t.Errorf("NoteTokens = %v, want no tokens when the note is off", got)
	}
}

func TestSidebarRowsTOMLEndsWithTheNoteRows(t *testing.T) {
	cfg := threeTags()
	cfg.NoteLines = 2
	cfg.NoteColor = "#9399b2"
	out := SidebarRowsTOML(cfg)
	for _, want := range []string{
		`[{ token = "$note1", fg = "#9399b2" }],`,
		`[{ token = "$note2", fg = "#9399b2" }],`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("snippet lacks %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "$note3") {
		t.Error("more note rows than note_lines")
	}
	if strings.Index(out, "$note1") < strings.Index(out, "$t3_wip") {
		t.Error("note rows come before the tag rows")
	}
	if strings.Index(out, "$note1") > strings.Index(out, "$note2") {
		t.Error("note rows are not in order")
	}
}

func TestNoteLinesComeOutOfTheTagLines(t *testing.T) {
	cfg := &Config{NoteLines: 2}
	for i := 0; i < maxSidebarRows; i++ {
		cfg.Tags = append(cfg.Tags, Tag{Name: strings.Repeat("x", i+1), Color: "#fff"})
	}
	if got := tagLineCount(cfg); got != maxSidebarRows-2 {
		t.Errorf("tagLineCount = %d, want %d", got, maxSidebarRows-2)
	}
}
