package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// State is the persisted tag selection: workspace ID to the tags set on it.
// Herdr forgets workspace tokens over a server restart but keeps workspace
// IDs in its session file, so the ID is the stable key the startup hook
// re-reports from.
type State map[string][]string

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

// SaveState writes the state atomically. Workspaces without tags are
// dropped, so the file only ever lists what is set.
func SaveState(path string, st State) error {
	pruned := State{}
	for ws, tags := range st {
		if len(tags) > 0 {
			pruned[ws] = tags
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
