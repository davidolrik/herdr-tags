package main

import (
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Key is one decoded keypress: a typed character, or one of the named keys
// below, which are negative so they never collide with a character. What a
// character means is up to the picker: in the tag list "j" moves down, in
// the note field it is a letter.
type Key rune

const (
	KeyOther Key = -1 - iota
	KeyUp
	KeyDown
	KeyEnter
	KeyEscape
	KeyTab
	KeyBackspace
	KeyLeft
	KeyRight
	KeyHome
	KeyEnd
	KeyDelete
	KeyEraseToStart
	KeyEraseToEnd
)

// KeySpace toggles the tag under the cursor.
const KeySpace Key = ' '

// noteFieldWidth is how many characters of the note the picker shows; a
// longer note scrolls to keep its cursor in view.
const noteFieldWidth = 38

// The key hints Render shows, by where the cursor is.
const (
	listHints = "↑↓ move · space toggle · enter save · esc cancel"
	noteHints = "↑↓←→ move · type to edit · enter save · esc cancel"
)

// maxNoteLength is how many characters a note may hold, far more than the
// sidebar has room to show.
const maxNoteLength = 200

// ParseKeys decodes the bytes of one terminal read into keys. Arrow keys,
// home and end arrive as CSI (ESC [ A) or SS3 (ESC O A) sequences; a lone
// ESC, or an ESC that starts no known sequence, is the Escape key. A CSI
// sequence with parameters (ESC [ 1 ; 5 C) is consumed up to its final byte,
// so none of it is taken for typing; home, end and delete also come as a
// number before a tilde (ESC [ 3 ~).
func ParseKeys(buf []byte) []Key {
	var keys []Key
	for i := 0; i < len(buf); i++ {
		c := buf[i]
		switch {
		case c == 0x1b:
			if i+2 < len(buf) && (buf[i+1] == '[' || buf[i+1] == 'O') {
				final := i + 2
				for buf[i+1] == '[' && buf[final] < 0x40 && final+1 < len(buf) {
					final++
				}
				switch buf[final] {
				case 'A':
					keys = append(keys, KeyUp)
				case 'B':
					keys = append(keys, KeyDown)
				case 'C':
					keys = append(keys, KeyRight)
				case 'D':
					keys = append(keys, KeyLeft)
				case 'H':
					keys = append(keys, KeyHome)
				case 'F':
					keys = append(keys, KeyEnd)
				case '~':
					switch string(buf[i+2 : final]) {
					case "1", "7":
						keys = append(keys, KeyHome)
					case "4", "8":
						keys = append(keys, KeyEnd)
					case "3":
						keys = append(keys, KeyDelete)
					default:
						keys = append(keys, KeyOther)
					}
				default:
					keys = append(keys, KeyOther)
				}
				i = final
				continue
			}
			keys = append(keys, KeyEscape)
		case c == '\t':
			keys = append(keys, KeyTab)
		case c == 0x7f || c == 0x08:
			keys = append(keys, KeyBackspace)
		case c == '\r' || c == '\n':
			keys = append(keys, KeyEnter)
		case c == 0x03: // ctrl-c
			keys = append(keys, KeyEscape)
		case c == 0x01: // ctrl-a
			keys = append(keys, KeyHome)
		case c == 0x05: // ctrl-e
			keys = append(keys, KeyEnd)
		case c == 0x15: // ctrl-u
			keys = append(keys, KeyEraseToStart)
		case c == 0x0b: // ctrl-k
			keys = append(keys, KeyEraseToEnd)
		case c < 0x20:
			keys = append(keys, KeyOther)
		default:
			r, size := utf8.DecodeRune(buf[i:])
			if r == utf8.RuneError {
				keys = append(keys, KeyOther)
			} else {
				keys = append(keys, Key(r))
			}
			i += size - 1
		}
	}
	return keys
}

// Picker is the selector's state: the configured tags, which are on, the
// note, and where the cursor is. With HasNote the cursor has one position
// past the last tag, the note field. NoteCursor is the character of the note
// that typing goes in front of; at the note's length it is at the end. Done
// is set by enter (Saved) or escape (not Saved).
type Picker struct {
	Tags       []Tag
	Selected   []bool
	HasNote    bool
	Note       string
	NoteCursor int
	NoteColor  string
	Cursor     int
	Done       bool
	Saved      bool

	// noteScroll is the first character of the note the field shows.
	noteScroll int
}

// NewPicker starts a picker over cfg's tags with the named ones selected.
func NewPicker(cfg *Config, selected []string) *Picker {
	on := map[string]bool{}
	for _, name := range selected {
		on[name] = true
	}
	p := &Picker{
		Tags:      cfg.Tags,
		Selected:  make([]bool, len(cfg.Tags)),
		HasNote:   cfg.NoteLines > 0,
		NoteColor: cfg.NoteColor,
	}
	for i, tag := range cfg.Tags {
		p.Selected[i] = on[tag.Name]
	}
	return p
}

// SetNote replaces the note and puts its cursor at the end.
func (p *Picker) SetNote(note string) {
	p.Note = note
	p.NoteCursor = utf8.RuneCountInString(note)
	p.scrollNote()
}

// scrollNote moves the note field's view as little as it takes to show the
// cursor, without scrolling past the end of the note.
func (p *Picker) scrollNote() {
	length := utf8.RuneCountInString(p.Note)
	p.noteScroll = min(p.noteScroll, max(length-noteFieldWidth, 0), p.NoteCursor)
	p.noteScroll = max(p.noteScroll, p.NoteCursor-noteFieldWidth)
}

// editNote applies a key the note field keeps to itself: cursor movement,
// erasing, and typing at the cursor. Erasing to the start or the end takes
// everything on that side of the cursor, the way a shell's ctrl-u and ctrl-k
// do. It reports whether k was one.
func (p *Picker) editNote(k Key) bool {
	note := []rune(p.Note)
	at := p.NoteCursor
	switch {
	case k == KeyLeft:
		at = max(at-1, 0)
	case k == KeyRight:
		at = min(at+1, len(note))
	case k == KeyHome:
		at = 0
	case k == KeyEnd:
		at = len(note)
	case k == KeyBackspace:
		if at > 0 {
			note = slices.Delete(note, at-1, at)
			at--
		}
	case k == KeyDelete:
		if at < len(note) {
			note = slices.Delete(note, at, at+1)
		}
	case k == KeyEraseToStart:
		note = note[at:]
		at = 0
	case k == KeyEraseToEnd:
		note = note[:at]
	case k >= 0:
		if len(note) < maxNoteLength {
			note = slices.Insert(note, at, rune(k))
			at++
		}
	default:
		return false
	}
	p.Note, p.NoteCursor = string(note), at
	p.scrollNote()
	return true
}

// OnNote reports whether the cursor is on the note field.
func (p *Picker) OnNote() bool {
	return p.HasNote && p.Cursor == len(p.Tags)
}

// Handle applies one key. The cursor wraps at both ends, and tab jumps
// between the tag list and the note field. On the note field characters are
// typed at the note's cursor, which left, right, home and end move, and
// backspace, delete, ctrl-u and ctrl-k erase; in the tag list j, k and q
// stand in for down, up and escape.
func (p *Picker) Handle(k Key) {
	if p.OnNote() {
		if p.editNote(k) {
			return
		}
	} else {
		switch k {
		case 'k':
			k = KeyUp
		case 'j':
			k = KeyDown
		case 'q':
			k = KeyEscape
		}
	}
	n := len(p.Tags)
	if p.HasNote {
		n++
	}
	switch k {
	case KeyUp:
		p.Cursor = (p.Cursor + n - 1) % n
	case KeyDown:
		p.Cursor = (p.Cursor + 1) % n
	case KeyTab:
		if p.OnNote() {
			p.Cursor = 0
		} else if p.HasNote {
			p.Cursor = len(p.Tags)
		}
	case KeySpace:
		p.Selected[p.Cursor] = !p.Selected[p.Cursor]
	case KeyEnter:
		p.Done, p.Saved = true, true
	case KeyEscape:
		p.Done, p.Saved = true, false
	}
}

// SelectedNames lists the selected tags in config order.
func (p *Picker) SelectedNames() []string {
	var names []string
	for i, on := range p.Selected {
		if on {
			names = append(names, p.Tags[i].Name)
		}
	}
	return names
}

// hexToRGB expands a validated #RGB or #RRGGBB color.
func hexToRGB(color string) (r, g, b uint8) {
	hex := strings.TrimPrefix(color, "#")
	if len(hex) == 3 {
		hex = string([]byte{hex[0], hex[0], hex[1], hex[1], hex[2], hex[2]})
	}
	v, _ := strconv.ParseUint(hex, 16, 32)
	return uint8(v >> 16), uint8(v >> 8), uint8(v)
}

func colored(color, text string) string {
	r, g, b := hexToRGB(color)
	return fmt.Sprintf("\x1b[38;2;%d;%d;%dm%s\x1b[0m", r, g, b, text)
}

// renderNote draws the note field's text: the stretch of the note in view,
// with an ellipsis on a side that has more, and the cursor, while the field
// has the focus, in reverse video on the character typing goes in front of
// or on a blank after the last one.
func renderNote(p *Picker) string {
	note := []rune(p.Note)
	cursor := -1
	if p.OnNote() {
		cursor = p.NoteCursor - p.noteScroll
	}
	// A cursor one step past the view gets a cell of its own, the one a
	// cursor at the end of the note stands on.
	end := min(len(note), p.noteScroll+noteFieldWidth)
	if cursor == noteFieldWidth && end < len(note) {
		end++
	}
	cells := note[p.noteScroll:end]
	if p.noteScroll > 0 && cursor != 0 {
		cells[0] = []rune(noteEllipsis)[0]
	}
	if end < len(note) && cursor != len(cells)-1 {
		cells[len(cells)-1] = []rune(noteEllipsis)[0]
	}
	if cursor < 0 {
		return colored(p.NoteColor, string(cells))
	}
	if cursor == len(cells) {
		cells = append(cells, ' ')
	}
	var b strings.Builder
	if cursor > 0 {
		b.WriteString(colored(p.NoteColor, string(cells[:cursor])))
	}
	fmt.Fprintf(&b, "\x1b[7m%c\x1b[0m", cells[cursor])
	if cursor+1 < len(cells) {
		b.WriteString(colored(p.NoteColor, string(cells[cursor+1:])))
	}
	return b.String()
}

// Render draws the whole picker: a header naming the workspace, one line
// per tag in its own color so the preview matches the sidebar, the note
// field when there is one, and the key hints for where the cursor is. The
// screen is cleared first, so every frame stands on its own.
func Render(w io.Writer, p *Picker, workspaceLabel string) {
	var b strings.Builder
	b.WriteString("\x1b[2J\x1b[H")
	fmt.Fprintf(&b, "\x1b[1mTags for %s\x1b[0m\r\n\r\n", workspaceLabel)
	for i, tag := range p.Tags {
		cursor, check := " ", " "
		if i == p.Cursor {
			cursor = ">"
		}
		if p.Selected[i] {
			check = "x"
		}
		fmt.Fprintf(&b, "%s [%s] %s\r\n", cursor, check, colored(tag.Color, TagLabel(tag.Name)))
	}
	hints := listHints
	if p.HasNote {
		cursor := " "
		if p.OnNote() {
			cursor = ">"
			hints = noteHints
		}
		fmt.Fprintf(&b, "\r\n%s Note: %s\r\n", cursor, renderNote(p))
	}
	fmt.Fprintf(&b, "\r\n\x1b[2m%s\x1b[0m\r\n", hints)
	io.WriteString(w, b.String())
}

// partialRuneTail is how many bytes at the end of buf start a character
// whose remaining bytes have not arrived yet.
func partialRuneTail(buf []byte) int {
	for back := 1; back <= len(buf) && back < utf8.UTFMax; back++ {
		if tail := buf[len(buf)-back:]; utf8.RuneStart(tail[0]) {
			if utf8.FullRune(tail) {
				return 0
			}
			return back
		}
	}
	return 0
}

// RunPicker drives p from keypresses read on in until it is done, redrawing
// after every read. A character cut off by the end of a read is held back
// until the rest of it arrives. The caller owns terminal modes. EOF on in
// ends the picker without saving.
func RunPicker(in io.Reader, out io.Writer, p *Picker, workspaceLabel string) (bool, error) {
	buf := make([]byte, 64)
	held := 0
	for !p.Done {
		Render(out, p, workspaceLabel)
		n, err := in.Read(buf[held:])
		end := held + n
		whole := end - partialRuneTail(buf[:end])
		for _, k := range ParseKeys(buf[:whole]) {
			p.Handle(k)
			if p.Done {
				break
			}
		}
		held = copy(buf, buf[whole:end])
		if err == io.EOF {
			break
		}
		if err != nil {
			return false, err
		}
	}
	io.WriteString(out, "\x1b[2J\x1b[H")
	return p.Saved, nil
}
