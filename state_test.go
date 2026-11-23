package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestStateRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tags.json")
	st := State{
		"w1": {Tags: []string{"blocked", "wip"}},
		"w2": {},
		"w3": {Note: "waiting on søren"},
		"w4": {Tags: []string{"wip"}, Note: "half done"},
	}
	if err := SaveState(path, st); err != nil {
		t.Fatalf("SaveState: %v", err)
	}
	got, err := LoadState(path)
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	want := State{
		"w1": {Tags: []string{"blocked", "wip"}},
		"w3": {Note: "waiting on søren"},
		"w4": {Tags: []string{"wip"}, Note: "half done"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("LoadState = %v, want %v (empty entries dropped)", got, want)
	}
	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0o600 {
		t.Errorf("state file mode = %o, want 600", info.Mode().Perm())
	}
}

func TestLoadStateReadsBareTagLists(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tags.json")
	os.WriteFile(path, []byte(`{"w1": ["blocked", "wip"], "w2": {"tags": ["ready"], "note": "ship it"}}`), 0o600)
	got, err := LoadState(path)
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	want := State{
		"w1": {Tags: []string{"blocked", "wip"}},
		"w2": {Tags: []string{"ready"}, Note: "ship it"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("LoadState = %v, want %v", got, want)
	}
}

func TestLoadStateMissingIsEmpty(t *testing.T) {
	got, err := LoadState(filepath.Join(t.TempDir(), "nope.json"))
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("LoadState = %v, want empty", got)
	}
}

func TestLoadStateCorruptIsError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tags.json")
	os.WriteFile(path, []byte("{nope"), 0o600)
	if _, err := LoadState(path); err == nil {
		t.Error("LoadState accepted corrupt JSON")
	}
}

func TestStatePathIsPerSession(t *testing.T) {
	if got := StatePath("/s", "work"); got != "/s/tags.work.json" {
		t.Errorf("StatePath = %q", got)
	}
}
