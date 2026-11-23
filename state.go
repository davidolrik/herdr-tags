package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Entry is what one workspace carries: the tags set on it and its note.
type Entry struct {
	Tags []string `json:"tags,omitempty"`
	Note string   `json:"note,omitempty"`
}

// UnmarshalJSON also reads an entry stored as a bare list of tags, the form
// of a state file written before workspaces had notes.
func (e *Entry) UnmarshalJSON(data []byte) error {
	var tags []string
	if err := json.Unmarshal(data, &tags); err == nil {
		*e = Entry{Tags: tags}
		return nil
	}
	type plain Entry
	return json.Unmarshal(data, (*plain)(e))
}

// State is the persisted tags and notes: workspace ID to what is set on it.
// Herdr forgets workspace tokens over a server restart but keeps workspace
// IDs in its session file, so the ID is the stable key the startup hook
// re-reports from.
type State map[string]Entry

// StatePath is the state file for one herdr session; the state dir is
// shared by every session's server.
func StatePath(stateDir, session string) string {
	return filepath.Join(stateDir, "tags."+session+".json")
}

// LoadState reads the state file. A missing file is an empty state;
// unreadable content is an error so a corrupt file is noticed, not wiped.
func LoadState(path string) (State, error) {
	st := State{}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return st, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, &st); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return st, nil
}

// SaveState writes the state atomically. Workspaces with neither tags nor a
// note are dropped, so the file only ever lists what is set.
func SaveState(path string, st State) error {
	pruned := State{}
	for ws, entry := range st {
		if len(entry.Tags) > 0 || entry.Note != "" {
			pruned[ws] = entry
		}
	}
	data, err := json.MarshalIndent(pruned, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(data, '\n'), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
