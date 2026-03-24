package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
)

// TextInput is a simple inline text input with cursor movement.
type TextInput struct {
	Value  string
	cursor int
}

// NewTextInput creates a TextInput with an initial value and cursor at end.
func NewTextInput(value string) TextInput {
	return TextInput{Value: value, cursor: len(value)}
}

// Update handles a key press. Returns true if the key was consumed.
func (t *TextInput) Update(msg tea.KeyPressMsg) bool {
	switch msg.String() {
	case "left":
		if t.cursor > 0 {
			t.cursor--
		}
	case "right":
		if t.cursor < len(t.Value) {
			t.cursor++
		}
	case "home", "ctrl+a":
		t.cursor = 0
	case "end", "ctrl+e":
		t.cursor = len(t.Value)
	case "backspace":
		if t.cursor > 0 {
			t.Value = t.Value[:t.cursor-1] + t.Value[t.cursor:]
			t.cursor--
		}
	case "delete":
		if t.cursor < len(t.Value) {
			t.Value = t.Value[:t.cursor] + t.Value[t.cursor+1:]
		}
	case "ctrl+u":
		t.Value = t.Value[t.cursor:]
		t.cursor = 0
	case "ctrl+k":
		t.Value = t.Value[:t.cursor]
	default:
		if msg.Text != "" {
			t.Value = t.Value[:t.cursor] + msg.Text + t.Value[t.cursor:]
			t.cursor += len(msg.Text)
		} else {
			return false
		}
	}
	return true
}

// Render returns the input value with a cursor indicator.
func (t TextInput) Render() string {
	if t.cursor >= len(t.Value) {
		return t.Value + "█"
	}
	var b strings.Builder
	b.WriteString(t.Value[:t.cursor])
	b.WriteString("█")
	b.WriteString(t.Value[t.cursor:])
	return b.String()
}
