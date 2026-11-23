package main

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Key is one decoded keypress the picker reacts to.
type Key int

const (
	KeyOther Key = iota
	KeyUp
	KeyDown
	KeySpace
	KeyEnter
	KeyEscape
)

// ParseKeys decodes the bytes of one terminal read into keys. Arrow keys
// arrive as CSI (ESC [ A) or SS3 (ESC O A) sequences; a lone ESC, or an ESC
// that starts no known sequence, is the Escape key.
func ParseKeys(buf []byte) []Key {
	var keys []Key
	for i := 0; i < len(buf); i++ {
		c := buf[i]
		switch {
		case c == 0x1b:
			if i+2 < len(buf) && (buf[i+1] == '[' || buf[i+1] == 'O') {
				switch buf[i+2] {
				case 'A':
					keys = append(keys, KeyUp)
				case 'B':
					keys = append(keys, KeyDown)
				default:
					keys = append(keys, KeyOther)
				}
				i += 2
				continue
			}
			keys = append(keys, KeyEscape)
		case c == 'k':
			keys = append(keys, KeyUp)
		case c == 'j':
			keys = append(keys, KeyDown)
		case c == ' ':
			keys = append(keys, KeySpace)
		case c == '\r' || c == '\n':
			keys = append(keys, KeyEnter)
		case c == 'q' || c == 0x03: // q, ctrl-c
			keys = append(keys, KeyEscape)
		default:
			keys = append(keys, KeyOther)
		}
	}
	return keys
}

// Picker is the selector's state: the configured tags, which are on, and
// where the cursor is. Done is set by enter (Saved) or escape (not Saved).
type Picker struct {
	Tags     []Tag
	Selected []bool
	Cursor   int
	Done     bool
	Saved    bool
}

// NewPicker starts a picker over cfg's tags with the named ones selected.
func NewPicker(cfg *Config, selected []string) *Picker {
	on := map[string]bool{}
	for _, name := range selected {
		on[name] = true
	}
	p := &Picker{Tags: cfg.Tags, Selected: make([]bool, len(cfg.Tags))}
	for i, tag := range cfg.Tags {
		p.Selected[i] = on[tag.Name]
	}
	return p
}

// Handle applies one key. The cursor wraps at both ends.
func (p *Picker) Handle(k Key) {
	n := len(p.Tags)
	switch k {
	case KeyUp:
		p.Cursor = (p.Cursor + n - 1) % n
	case KeyDown:
		p.Cursor = (p.Cursor + 1) % n
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

// Render draws the whole picker: a header naming the workspace, one line
// per tag in its own color so the preview matches the sidebar, and the key
// hints. The screen is cleared first, so every frame stands on its own.
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
	b.WriteString("\r\n\x1b[2m↑↓ move · space toggle · enter save · esc cancel\x1b[0m\r\n")
	io.WriteString(w, b.String())
}

// RunPicker drives p from keypresses read on in until it is done, redrawing
// after every read. The caller owns terminal modes. EOF on in ends the
// picker without saving.
func RunPicker(in io.Reader, out io.Writer, p *Picker, workspaceLabel string) (bool, error) {
	buf := make([]byte, 64)
	for !p.Done {
		Render(out, p, workspaceLabel)
		n, err := in.Read(buf)
		for _, k := range ParseKeys(buf[:n]) {
			p.Handle(k)
			if p.Done {
				break
			}
		}
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
