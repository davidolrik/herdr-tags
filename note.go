package main

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// noteEllipsis ends a note that has more to say than its lines hold.
const noteEllipsis = "…"

// NoteTokenName is the herdr workspace metadata token that carries one
// sidebar line of the note, counted from 1.
func NoteTokenName(line int) string {
	return fmt.Sprintf("note%d", line)
}

// WrapNote breaks a note into at most maxLines sidebar lines of at most
// columns characters, at word boundaries where it can; a word wider than a
// line is cut. A note that needs more lines ends in an ellipsis on the last
// one. Width is counted in characters, which is the cell count for
// everything but double-width characters such as CJK and emoji.
func WrapNote(note string, columns, maxLines int) []string {
	if maxLines < 1 {
		return nil
	}
	columns = max(columns, 1)
	var lines []string
	for _, word := range strings.Fields(note) {
		if n := len(lines); n > 0 && utf8.RuneCountInString(lines[n-1])+1+utf8.RuneCountInString(word) <= columns {
			lines[n-1] += " " + word
			continue
		}
		runes := []rune(word)
		for len(runes) > columns {
			lines = append(lines, string(runes[:columns]))
			runes = runes[columns:]
		}
		lines = append(lines, string(runes))
	}
	if len(lines) > maxLines {
		last := []rune(lines[maxLines-1])
		if len(last) >= columns {
			last = last[:columns-1]
		}
		lines = lines[:maxLines]
		lines[maxLines-1] = strings.TrimRight(string(last), " ") + noteEllipsis
	}
	return lines
}

// NoteTokens builds the workspace.report_metadata token map that makes herdr
// show the note wrapped to columns cells: every note line's token is
// mentioned, set to its text where the note reaches and null (cleared)
// below, so a shortened note leaves nothing behind.
func NoteTokens(cfg *Config, note string, columns int) map[string]any {
	tokens := map[string]any{}
	lines := WrapNote(note, columns, cfg.NoteLines)
	for line := 1; line <= cfg.NoteLines; line++ {
		var value any
		if line <= len(lines) {
			value = lines[line-1]
		}
		tokens[NoteTokenName(line)] = value
	}
	return tokens
}

// ReportNote pushes the note for one workspace to herdr, wrapped to columns
// cells per sidebar line.
func ReportNote(sockPath, workspaceID string, cfg *Config, note string, columns int) error {
	tokens := NoteTokens(cfg, note, columns)
	if len(tokens) == 0 {
		return nil
	}
	return reportTokens(sockPath, workspaceID, tokens)
}
