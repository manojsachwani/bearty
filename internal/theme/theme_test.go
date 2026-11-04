package theme

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func sample() Options {
	return Options{Name: "Test", Paper: Papers[0].Hex, Ink: Inks[0].Hex, Font: Font{Name: "Inkina", PostScript: "Inkina-Regular", Heading: "Inkina-Regular"}, Size: 17}
}

// flatten turns nested JSON objects into "a.b c.d" paths.
func flatten(prefix string, v any, out map[string]any) {
	m, ok := v.(map[string]any)
	if !ok {
		out[prefix] = v
		return
	}
	for k, child := range m {
		p := k
		if prefix != "" {
			p = prefix + "." + k
		}
		flatten(p, child, out)
	}
}

func decode(t *testing.T, b []byte) map[string]any {
	t.Helper()
	var v any
	if err := json.Unmarshal(b, &v); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	flat := map[string]any{}
	flatten("", v, flat)
	return flat
}

func TestBuildRefsResolve(t *testing.T) {
	b, err := Build(sample()).JSON()
	if err != nil {
		t.Fatal(err)
	}
	flat := decode(t, b)
	for k, v := range flat {
		s, ok := v.(string)
		if !ok || !strings.HasPrefix(s, "$") {
			continue
		}
		if _, ok := flat[s[1:]]; !ok {
			t.Errorf("%s references missing key %q", k, s)
		}
	}
}

func TestBuildUsesChoices(t *testing.T) {
	o := sample()
	th := Build(o)
	if th.Base.BackgroundColor != o.Paper || th.Base.AccentColor != o.Ink || th.Editor.Headers.TextColor != o.Ink {
		t.Errorf("paper/ink not applied: %+v", th.Base)
	}
	if th.Editor.TextFont != "Inkina-Regular" || th.Editor.TextSize != 17 {
		t.Errorf("font/size not applied: %s %d", th.Editor.TextFont, th.Editor.TextSize)
	}
}

// TestKeysMatchBear compares against a built-in theme when Bear is installed.
func TestKeysMatchBear(t *testing.T) {
	ref, err := os.ReadFile("/Applications/Bear.app/Contents/Frameworks/BearCore.framework/Versions/A/Resources/Dark Graphite.theme")
	if err != nil {
		t.Skip("Bear not installed")
	}
	b, _ := Build(sample()).JSON()
	ours, theirs := decode(t, b), decode(t, ref)
	for k := range theirs {
		if strings.HasPrefix(k, "meta") {
			continue
		}
		if _, ok := ours[k]; !ok {
			t.Errorf("missing key %q", k)
		}
	}
}

func TestValidateHex(t *testing.T) {
	for _, s := range []string{"#D35A27", "#ffffff"} {
		if ValidateHex(s) != nil {
			t.Errorf("%s should be valid", s)
		}
	}
	for _, s := range []string{"D35A27", "#FFF", "#GGGGGG", ""} {
		if ValidateHex(s) == nil {
			t.Errorf("%s should be invalid", s)
		}
	}
}

func TestMix(t *testing.T) {
	if got := Mix("#000000", "#FFFFFF", 0.5); got != "#808080" {
		t.Errorf("Mix = %s", got)
	}
	if got := Shade("#FFFFFF", 0); got != "#FFFFFF" {
		t.Errorf("Shade = %s", got)
	}
}

func TestOverrides(t *testing.T) {
	o := sample()
	o.Headings = "#112233"
	o.Sidebar = SidebarColors{Background: "#EEEEEE", Icons: "#AA0000"}
	o.NoteList = NoteListColors{Title: "#000000", Selected: "#DDDDDD"}
	th := Build(o)
	if th.Editor.Headers.TextColor != "#112233" {
		t.Errorf("heading hex not applied: %s", th.Editor.Headers.TextColor)
	}
	if th.Sidebar.BackgroundColor != "#EEEEEE" || th.Sidebar.IconColor != "#AA0000" || th.Sidebar.TextColor != "#333333" {
		t.Errorf("sidebar overrides: %+v", th.Sidebar)
	}
	if th.Notes.TitleColor != "#000000" || th.Notes.SelectionBackground != "#DDDDDD" {
		t.Errorf("note list overrides: %+v", th.Notes)
	}
	// Unset note list fields stay derived.
	if th.Notes.DateColor != Mix(Rule, o.Paper, 0.45) {
		t.Errorf("date should be derived, got %s", th.Notes.DateColor)
	}
}

func TestAccents(t *testing.T) {
	o := sample()
	o.Accents = Accents{Links: "#3D4F74", Ticks: "#4D7374"}
	th := Build(o)
	if th.Editor.LinkColor != "#3D4F74" || th.Editor.Task.CheckColor != "#4D7374" {
		t.Errorf("accents not applied: link %s tick %s", th.Editor.LinkColor, th.Editor.Task.CheckColor)
	}
	if th.Editor.ListMarkerColor != "$base.accent color" {
		t.Errorf("unset markers should follow the ink, got %s", th.Editor.ListMarkerColor)
	}
}
