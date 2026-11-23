package main

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
)

func TestParseKeys(t *testing.T) {
	cases := []struct {
		in   string
		want []Key
	}{
		{"\x1b[A", []Key{KeyUp}},
		{"\x1b[B", []Key{KeyDown}},
		{"\x1bOA", []Key{KeyUp}},
		{"k", []Key{KeyUp}},
		{"j", []Key{KeyDown}},
		{" ", []Key{KeySpace}},
		{"\r", []Key{KeyEnter}},
		{"\n", []Key{KeyEnter}},
		{"\x1b", []Key{KeyEscape}},
		{"q", []Key{KeyEscape}},
		{"\x03", []Key{KeyEscape}},
		{"x", []Key{KeyOther}},
		{"\x1b[B \r", []Key{KeyDown, KeySpace, KeyEnter}},
		{"\x1b[C", []Key{KeyOther}},
	}
	for _, c := range cases {
		if got := ParseKeys([]byte(c.in)); !reflect.DeepEqual(got, c.want) {
			t.Errorf("ParseKeys(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestPickerNavigationWrapsAndToggles(t *testing.T) {
	p := NewPicker(threeTags(), []string{"wip"})
	if got := p.SelectedNames(); !reflect.DeepEqual(got, []string{"wip"}) {
		t.Fatalf("initial selection = %v", got)
	}
	p.Handle(KeyUp)
	if p.Cursor != 2 {
		t.Errorf("up from top: cursor = %d, want 2 (wrap)", p.Cursor)
	}
	p.Handle(KeyDown)
	if p.Cursor != 0 {
		t.Errorf("down from bottom: cursor = %d, want 0 (wrap)", p.Cursor)
	}
	p.Handle(KeySpace)
	p.Handle(KeyDown)
	p.Handle(KeyDown)
	p.Handle(KeySpace)
	if got := p.SelectedNames(); !reflect.DeepEqual(got, []string{"in-review"}) {
		t.Errorf("after toggles: %v, want [in-review] (config order)", got)
	}
	if p.Done {
		t.Error("picker finished without enter or escape")
	}
}

func TestPickerEnterSavesEscapeDiscards(t *testing.T) {
	p := NewPicker(threeTags(), nil)
	p.Handle(KeyEnter)
	if !p.Done || !p.Saved {
		t.Errorf("after enter: done=%v saved=%v", p.Done, p.Saved)
	}
	p = NewPicker(threeTags(), nil)
	p.Handle(KeyEscape)
	if !p.Done || p.Saved {
		t.Errorf("after escape: done=%v saved=%v", p.Done, p.Saved)
	}
}

func TestRenderShowsCursorChecksAndColors(t *testing.T) {
	p := NewPicker(threeTags(), []string{"blocked"})
	p.Handle(KeyDown)
	var out bytes.Buffer
	Render(&out, p, "herdr-titles")
	s := out.String()
	for _, want := range []string{
		"herdr-titles",
		"  [ ] \x1b[38;2;249;226;175m#in-review\x1b[0m",
		"> [x] \x1b[38;2;243;139;168m#blocked\x1b[0m",
		"  [ ] \x1b[38;2;137;180;250m#wip\x1b[0m",
		"space", "enter", "esc",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("render lacks %q:\n%q", want, s)
		}
	}
}

func TestHexToRGBHandlesShortForm(t *testing.T) {
	r, g, b := hexToRGB("#fab")
	if r != 0xff || g != 0xaa || b != 0xbb {
		t.Errorf("hexToRGB(#fab) = %d,%d,%d", r, g, b)
	}
}

func TestRunPickerDrivesStateFromInput(t *testing.T) {
	p := NewPicker(threeTags(), nil)
	in := strings.NewReader("\x1b[B \r")
	var out bytes.Buffer
	saved, err := RunPicker(in, &out, p, "ws")
	if err != nil {
		t.Fatalf("RunPicker: %v", err)
	}
	if !saved {
		t.Error("enter did not save")
	}
	if got := p.SelectedNames(); !reflect.DeepEqual(got, []string{"blocked"}) {
		t.Errorf("selection = %v, want [blocked]", got)
	}
	if out.Len() == 0 {
		t.Error("nothing was rendered")
	}
}

func TestRunPickerEOFDiscards(t *testing.T) {
	p := NewPicker(threeTags(), nil)
	saved, err := RunPicker(strings.NewReader(""), &bytes.Buffer{}, p, "ws")
	if err != nil {
		t.Fatalf("RunPicker: %v", err)
	}
	if saved {
		t.Error("EOF should not save")
	}
}
