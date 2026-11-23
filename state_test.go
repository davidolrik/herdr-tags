package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestStateRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tags.json")
	st := State{"w1": {"blocked", "wip"}, "w2": {}}
	if err := SaveState(path, st); err != nil {
		t.Fatalf("SaveState: %v", err)
	}
	got, err := LoadState(path)
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	want := State{"w1": {"blocked", "wip"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("LoadState = %v, want %v (empty entries dropped)", got, want)
	}
	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0o600 {
		t.Errorf("state file mode = %o, want 600", info.Mode().Perm())
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
