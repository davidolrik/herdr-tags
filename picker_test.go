package main

import (
	"bytes"
	"io"
	"reflect"
	"regexp"
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
		{"k", []Key{'k'}},
		{"j", []Key{'j'}},
		{" ", []Key{KeySpace}},
		{"\r", []Key{KeyEnter}},
		{"\n", []Key{KeyEnter}},
		{"\x1b", []Key{KeyEscape}},
		{"q", []Key{'q'}},
		{"\x03", []Key{KeyEscape}},
		{"x", []Key{'x'}},
		{"\t", []Key{KeyTab}},
		{"\x7f", []Key{KeyBackspace}},
		{"\x08", []Key{KeyBackspace}},
		{"\x02", []Key{KeyOther}},
		{"\x1b[D", []Key{KeyLeft}},
		{"\x1bOC", []Key{KeyRight}},
		{"\x1b[H", []Key{KeyHome}},
		{"\x1b[1~", []Key{KeyHome}},
		{"\x1bOH", []Key{KeyHome}},
		{"\x01", []Key{KeyHome}},
		{"\x1b[F", []Key{KeyEnd}},
		{"\x1b[4~", []Key{KeyEnd}},
		{"\x05", []Key{KeyEnd}},
		{"\x15", []Key{KeyEraseToStart}},
		{"\x0b", []Key{KeyEraseToEnd}},
		{"\x1b[3~", []Key{KeyDelete}},
		{"\x1b[5~x", []Key{KeyOther, 'x'}},
		{"æ✓", []Key{'æ', '✓'}},
		{"\xff", []Key{KeyOther}},
		{"\x1b[1;5Cx", []Key{KeyRight, 'x'}},
		{"\x1b[B \r", []Key{KeyDown, KeySpace, KeyEnter}},
		{"\x1b[C", []Key{KeyRight}},
		{"\x1b[Z", []Key{KeyOther}},
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

func TestPickerListLettersActAsKeys(t *testing.T) {
	p := NewPicker(threeTags(), nil)
	p.Handle('j')
	if p.Cursor != 1 {
		t.Errorf("j: cursor = %d, want 1", p.Cursor)
	}
	p.Handle('k')
	if p.Cursor != 0 {
		t.Errorf("k: cursor = %d, want 0", p.Cursor)
	}
	p.Handle('q')
	if !p.Done || p.Saved {
		t.Errorf("after q: done=%v saved=%v", p.Done, p.Saved)
	}
}

func noteConfig() *Config {
	cfg := threeTags()
	cfg.NoteLines = 2
	cfg.NoteColor = "#9399b2"
	return cfg
}

func TestPickerNoteFieldSitsBelowTheTags(t *testing.T) {
	p := NewPicker(noteConfig(), nil)
	if p.OnNote() {
		t.Fatal("picker starts on the note field")
	}
	p.Handle(KeyUp)
	if !p.OnNote() {
		t.Errorf("up from the top: cursor = %d, want the note field", p.Cursor)
	}
	p.Handle(KeyDown)
	if p.Cursor != 0 {
		t.Errorf("down from the note field: cursor = %d, want 0 (wrap)", p.Cursor)
	}
	p.Handle(KeyTab)
	if !p.OnNote() {
		t.Error("tab did not jump to the note field")
	}
	p.Handle(KeyTab)
	if p.Cursor != 0 {
		t.Errorf("tab from the note field: cursor = %d, want 0", p.Cursor)
	}
}

func TestPickerWithoutNoteLinesHasNoNoteField(t *testing.T) {
	p := NewPicker(threeTags(), nil)
	p.Handle(KeyTab)
	if p.OnNote() || p.Cursor != 0 {
		t.Errorf("tab moved the cursor to %d without a note field", p.Cursor)
	}
}

func TestPickerNoteFieldTakesTyping(t *testing.T) {
	p := NewPicker(noteConfig(), []string{"wip"})
	p.SetNote("ask")
	p.Handle(KeyTab)
	for _, k := range []Key{' ', 'j', 'k', 'q', 'æ', 'x', KeyBackspace, KeyOther} {
		p.Handle(k)
	}
	if p.Note != "ask jkqæ" {
		t.Errorf("Note = %q, want %q", p.Note, "ask jkqæ")
	}
	if p.Done {
		t.Error("typing q in the note field closed the picker")
	}
	if got := p.SelectedNames(); !reflect.DeepEqual(got, []string{"wip"}) {
		t.Errorf("typing a space toggled a tag: %v", got)
	}
	for i := 0; i < 20; i++ {
		p.Handle(KeyBackspace)
	}
	if p.Note != "" {
		t.Errorf("Note = %q after erasing everything", p.Note)
	}
	p.Handle(KeyEnter)
	if !p.Done || !p.Saved {
		t.Errorf("enter in the note field: done=%v saved=%v", p.Done, p.Saved)
	}
}

func TestPickerNoteCursorMovesAndEditsInPlace(t *testing.T) {
	p := NewPicker(noteConfig(), nil)
	p.SetNote("søren")
	p.Handle(KeyTab)
	steps := []struct {
		keys []Key
		want string
		at   int
	}{
		{[]Key{KeyLeft, KeyLeft, 'X'}, "sørXen", 4},
		{[]Key{KeyBackspace, KeyBackspace}, "søen", 2},
		{[]Key{KeyDelete}, "søn", 2},
		{[]Key{KeyHome, KeyLeft, KeyBackspace, '>'}, ">søn", 1},
		{[]Key{KeyEnd, KeyRight, KeyDelete, '!'}, ">søn!", 5},
		{[]Key{KeyHome, KeyRight, KeyRight, KeyDelete}, ">sn!", 2},
	}
	for _, step := range steps {
		for _, k := range step.keys {
			p.Handle(k)
		}
		if p.Note != step.want || p.NoteCursor != step.at {
			t.Fatalf("after %v: Note = %q with the cursor at %d, want %q at %d", step.keys, p.Note, p.NoteCursor, step.want, step.at)
		}
	}
}

func TestPickerNoteErasesToEitherEnd(t *testing.T) {
	p := NewPicker(noteConfig(), nil)
	p.SetNote("waiting on søren")
	p.Handle(KeyTab)
	p.Handle(KeyEraseToEnd)
	if p.Note != "waiting on søren" || p.NoteCursor != 16 {
		t.Errorf("erase to end at the end: Note = %q, cursor at %d", p.Note, p.NoteCursor)
	}
	for i := 0; i < len([]rune("søren")); i++ {
		p.Handle(KeyLeft)
	}
	p.Handle(KeyEraseToEnd)
	if p.Note != "waiting on " || p.NoteCursor != 11 {
		t.Errorf("erase to end: Note = %q, cursor at %d", p.Note, p.NoteCursor)
	}
	for i := 0; i < len("on "); i++ {
		p.Handle(KeyLeft)
	}
	p.Handle(KeyEraseToStart)
	if p.Note != "on " || p.NoteCursor != 0 {
		t.Errorf("erase to start: Note = %q, cursor at %d", p.Note, p.NoteCursor)
	}
	p.Handle(KeyEnd)
	p.Handle(KeyEraseToStart)
	if p.Note != "" || p.NoteCursor != 0 {
		t.Errorf("erase to start from the end: Note = %q, cursor at %d", p.Note, p.NoteCursor)
	}
}

func TestRenderAfterClearingALongNote(t *testing.T) {
	p := NewPicker(noteConfig(), nil)
	p.SetNote(strings.Repeat("a", noteFieldWidth*2))
	p.Handle(KeyTab)
	p.Handle(KeyEraseToStart)
	if got := renderedNote(t, p); got != " " {
		t.Errorf("cleared note renders as %q, want only the cursor", got)
	}
}

func TestPickerNoteKeysDoNothingInTheTagList(t *testing.T) {
	p := NewPicker(noteConfig(), []string{"wip"})
	p.SetNote("keep")
	for _, k := range []Key{KeyLeft, KeyRight, KeyHome, KeyEnd, KeyDelete, KeyBackspace, KeyEraseToStart, KeyEraseToEnd} {
		p.Handle(k)
	}
	if p.Note != "keep" || p.NoteCursor != 4 || p.Cursor != 0 || p.Done {
		t.Errorf("Note = %q, NoteCursor = %d, Cursor = %d, Done = %v", p.Note, p.NoteCursor, p.Cursor, p.Done)
	}
}

func TestRenderMarksTheNoteCursor(t *testing.T) {
	p := NewPicker(noteConfig(), nil)
	p.SetNote("abc")
	p.Handle(KeyTab)
	var out bytes.Buffer
	Render(&out, p, "ws")
	if !strings.Contains(out.String(), "abc\x1b[0m\x1b[7m \x1b[0m") {
		t.Errorf("render lacks a cursor after the note:\n%q", out.String())
	}

	p.Handle(KeyLeft)
	p.Handle(KeyLeft)
	out.Reset()
	Render(&out, p, "ws")
	if !strings.Contains(out.String(), "a\x1b[0m\x1b[7mb\x1b[0m\x1b[38;2;147;153;178mc\x1b[0m") {
		t.Errorf("render lacks a cursor on the b:\n%q", out.String())
	}

	p.Handle(KeyDown)
	out.Reset()
	Render(&out, p, "ws")
	if strings.Contains(out.String(), "\x1b[7m") {
		t.Errorf("render shows a note cursor while the tag list has the focus:\n%q", out.String())
	}
}

// renderedNote is the note field's text as drawn, without styling.
func renderedNote(t *testing.T, p *Picker) string {
	t.Helper()
	var out bytes.Buffer
	Render(&out, p, "ws")
	_, field, found := strings.Cut(out.String(), "Note: ")
	if !found {
		t.Fatalf("render has no note field:\n%q", out.String())
	}
	field, _, _ = strings.Cut(field, "\r\n")
	return regexp.MustCompile("\x1b\\[[0-9;]*m").ReplaceAllString(field, "")
}

func TestRenderScrollsALongNoteWithItsCursor(t *testing.T) {
	head := "START" + strings.Repeat("a", noteFieldWidth-5)
	p := NewPicker(noteConfig(), nil)
	p.SetNote(head + "END")
	p.Handle(KeyTab)
	if got, want := renderedNote(t, p), "…"+head[4:]+"END "; got != want {
		t.Errorf("cursor at the end:\n got %q\nwant %q", got, want)
	}
	p.Handle(KeyLeft)
	if got, want := renderedNote(t, p), "…"+head[4:]+"END"; got != want {
		t.Errorf("one step left keeps the view:\n got %q\nwant %q", got, want)
	}
	p.Handle(KeyHome)
	if got, want := renderedNote(t, p), head[:noteFieldWidth-1]+"…"; got != want {
		t.Errorf("cursor at the start:\n got %q\nwant %q", got, want)
	}
	for i := 0; i < noteFieldWidth; i++ {
		p.Handle(KeyRight)
	}
	if got, want := renderedNote(t, p), head+"E"; got != want {
		t.Errorf("cursor just past the field:\n got %q\nwant %q", got, want)
	}
	for i := 0; i < len("END"); i++ {
		p.Handle(KeyDelete)
	}
	if got, want := renderedNote(t, p), head+" "; got != want {
		t.Errorf("after deleting the end:\n got %q\nwant %q", got, want)
	}
}

func TestPickerNoteStopsAtItsLimit(t *testing.T) {
	p := NewPicker(noteConfig(), nil)
	p.Handle(KeyTab)
	for i := 0; i < maxNoteLength+10; i++ {
		p.Handle('å')
	}
	if got := len([]rune(p.Note)); got != maxNoteLength {
		t.Errorf("note holds %d characters, want %d", got, maxNoteLength)
	}
}

func TestRenderShowsTheNoteField(t *testing.T) {
	p := NewPicker(noteConfig(), nil)
	p.SetNote("waiting on søren")
	var out bytes.Buffer
	Render(&out, p, "ws")
	s := out.String()
	if !strings.Contains(s, "  Note: \x1b[38;2;147;153;178mwaiting on søren\x1b[0m") {
		t.Errorf("render lacks the note in its color:\n%q", s)
	}
	if !strings.Contains(s, "space toggle") {
		t.Errorf("render lacks the list hints:\n%q", s)
	}

	p.Handle(KeyTab)
	out.Reset()
	Render(&out, p, "ws")
	s = out.String()
	if !strings.Contains(s, "> Note: ") {
		t.Errorf("render does not mark the focused note field:\n%q", s)
	}
	if !strings.Contains(s, "type to edit") {
		t.Errorf("render lacks the note hints:\n%q", s)
	}
	if strings.Contains(s, "space toggle") {
		t.Errorf("render offers space toggle while typing a note:\n%q", s)
	}
}

func TestRenderShowsTheEndOfALongNote(t *testing.T) {
	p := NewPicker(noteConfig(), nil)
	p.SetNote(strings.Repeat("a", noteFieldWidth) + "END")
	p.Handle(KeyTab)
	var out bytes.Buffer
	Render(&out, p, "ws")
	s := out.String()
	if !strings.Contains(s, "…"+strings.Repeat("a", noteFieldWidth-4)+"END") {
		t.Errorf("render does not show the tail of a long note:\n%q", s)
	}
	if strings.Contains(s, strings.Repeat("a", noteFieldWidth)) {
		t.Errorf("render overflows the note field:\n%q", s)
	}
}

func TestRenderWithoutNoteLinesHasNoNoteField(t *testing.T) {
	var out bytes.Buffer
	Render(&out, NewPicker(threeTags(), nil), "ws")
	if strings.Contains(out.String(), "Note:") {
		t.Errorf("render shows a note field although the note is off:\n%q", out.String())
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

// oneByteReader hands out its input a byte at a time, the way a slow
// terminal may split a multi-byte character over two reads.
type oneByteReader struct{ data []byte }

func (r *oneByteReader) Read(p []byte) (int, error) {
	if len(r.data) == 0 {
		return 0, io.EOF
	}
	p[0] = r.data[0]
	r.data = r.data[1:]
	return 1, nil
}

func TestRunPickerTypesANote(t *testing.T) {
	p := NewPicker(noteConfig(), nil)
	saved, err := RunPicker(strings.NewReader(" \tjust a note\r"), &bytes.Buffer{}, p, "ws")
	if err != nil {
		t.Fatalf("RunPicker: %v", err)
	}
	if !saved || p.Note != "just a note" {
		t.Errorf("saved = %v, Note = %q", saved, p.Note)
	}
	if got := p.SelectedNames(); !reflect.DeepEqual(got, []string{"in-review"}) {
		t.Errorf("selection = %v, want [in-review]", got)
	}
}

func TestRunPickerJoinsCharactersSplitOverReads(t *testing.T) {
	p := NewPicker(noteConfig(), nil)
	saved, err := RunPicker(&oneByteReader{data: []byte("\tsøren ✓\r")}, &bytes.Buffer{}, p, "ws")
	if err != nil {
		t.Fatalf("RunPicker: %v", err)
	}
	if !saved || p.Note != "søren ✓" {
		t.Errorf("saved = %v, Note = %q", saved, p.Note)
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
