package theme

import (
	"testing"
)

// Every preset paper/ink pair must give a readable dark version.
func TestDarkVariantIsReadable(t *testing.T) {
	for _, p := range Papers {
		for _, i := range Inks {
			o := Options{Paper: p.Hex, Ink: i.Hex, Size: 17, Headings: HeadingsBlack}
			d := o.DarkVariant(Dark{Mode: DarkAuto})
			v, err := Build(d).Values()
			if err != nil {
				t.Fatal(err)
			}
			bg := v["base.background color"].(string)
			if !IsDark(bg) {
				t.Fatalf("%s/%s: dark paper %s isn't dark", p.Name, i.Name, bg)
			}
			for key, min := range map[string]float64{
				"base.text color":               4.5,
				"editor.headers.text color":     4.5,
				"base.text secondary color":     3,
				"editor.code.text color":        4.5,
				"editor.link color":             4.5,
				"notes.title color":             4.5,
				"editor.separator.border color": 3,
			} {
				if c := Contrast(v[key].(string), bg); c < min {
					t.Errorf("%s/%s dark: %s %s on %s is %.1f:1, want ≥ %.1f", p.Name, i.Name, key, v[key], bg, c, min)
				}
			}
			side := v["sidebar.background color"].(string)
			if c := Contrast(v["sidebar.text color"].(string), side); c < 4.5 {
				t.Errorf("%s/%s dark: sidebar text %.1f:1", p.Name, i.Name, c)
			}
		}
	}
}

func TestDarkCustomColours(t *testing.T) {
	o := Options{Paper: "#F9F9F9", Ink: "#CF5624", Size: 17, Accents: Accents{Links: "#3D4F74"}}
	d := o.DarkVariant(Dark{Mode: DarkCustom, Paper: "#101820", Ink: "#F2C14E"})
	if d.Paper != "#101820" || d.Ink != "#F2C14E" {
		t.Errorf("custom dark paper/ink ignored: %s %s", d.Paper, d.Ink)
	}
	if _, _, l := ToHSL(d.Accents.Links); l < 0.6 {
		t.Errorf("navy link should be lifted for dark paper, got %s", d.Accents.Links)
	}
}

func TestDetailsAndExtra(t *testing.T) {
	o := Options{Paper: "#F9F9F9", Ink: "#CF5624", Size: 17,
		Highlights:   Highlights{Yellow: "#FFE066"},
		Code:         CodeColors{Font: "Menlo-Regular", Keyword: "#AA00AA"},
		Tag:          TagColors{Background: "#DDEEFF"},
		HeadingScale: 1.3,
		Extra:        map[string]any{"editor.toolbar.text color": "#123456", "notes.selection inactive color": "#EEEEEE"},
	}
	v, err := Build(o).Values()
	if err != nil {
		t.Fatal(err)
	}
	for k, want := range map[string]any{
		"editor.highlighter.yellow.background color": "#FFE066",
		"editor.highlighter.yellow.text color":       Rule,
		"editor.code.font":                           "Menlo-Regular",
		"editor.code.syntax highlight.keyword":       "#AA00AA",
		"editor.tag.background color":                "#DDEEFF",
		"editor.headers.modular scale":               1.3,
		"editor.toolbar.text color":                  "#123456",
		"notes.selection inactive color":             "#EEEEEE",
	} {
		if v[k] != want {
			t.Errorf("%s = %v, want %v", k, v[k], want)
		}
	}
}

func TestHSLRoundTrip(t *testing.T) {
	for _, hex := range []string{"#CF5624", "#3D4F74", "#F9F9F9", "#000000", "#FFFFFF", "#4D7374"} {
		h, s, l := ToHSL(hex)
		if got := FromHSL(h, s, l); got != hex {
			t.Errorf("%s → %s", hex, got)
		}
	}
}
