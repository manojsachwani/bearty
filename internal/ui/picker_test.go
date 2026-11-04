package ui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
)

func TestHSLRoundTrip(t *testing.T) {
	for _, hex := range []string{"#CF5624", "#3D4F74", "#F9F9F9", "#000000", "#FFFFFF", "#4D7374"} {
		h, s, l := toHSL(hex)
		if got := fromHSL(h, s, l); got != hex {
			t.Errorf("%s → %v %v %v → %s", hex, h, s, l, got)
		}
	}
}

func press(p *picker, keys ...string) {
	for _, k := range keys {
		var msg tea.KeyPressMsg
		switch k {
		case "left":
			msg = tea.KeyPressMsg{Code: tea.KeyLeft}
		case "right":
			msg = tea.KeyPressMsg{Code: tea.KeyRight}
		case "down":
			msg = tea.KeyPressMsg{Code: tea.KeyDown}
		case "backspace":
			msg = tea.KeyPressMsg{Code: tea.KeyBackspace}
		default:
			msg = tea.KeyPressMsg{Code: rune(k[0]), Text: k}
		}
		p.Update(msg)
	}
}

func TestPickerKeys(t *testing.T) {
	v := ""
	p := newOptionalPicker("k", "Links", &v, func() string { return "#CF5624" })
	p.WithKeyMap(huh.NewDefaultKeyMap())
	p.Focus()

	if p.current() != "#CF5624" || v != "" {
		t.Fatalf("empty optional picker should show the automatic colour, got %s / %q", p.current(), v)
	}
	press(p, "#", "3", "d", "4", "f", "7", "4")
	if v != "#3D4F74" {
		t.Errorf("typed hex = %q", v)
	}
	press(p, "2") // palette: Navy is the second ink
	if v != "#3D4F74" {
		t.Errorf("palette pick = %q", v)
	}
	press(p, "down", "down", "right", "right") // lightness up
	if _, _, l := toHSL(v); l <= 0.35 {
		t.Errorf("lightness should rise, got %v (%s)", l, v)
	}
	press(p, "x")
	if v != "" {
		t.Errorf("x should return to automatic, got %q", v)
	}
	p.Update(tea.PasteMsg{Content: " 4d7374 "})
	if v != "#4D7374" {
		t.Errorf("paste = %q", v)
	}
}
