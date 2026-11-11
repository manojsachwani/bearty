package ui

import (
	"strings"
	"testing"

	"charm.land/bubbles/v2/help"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"

	"github.com/manojsachwani/bearty/internal/fonts"
	"github.com/manojsachwani/bearty/internal/library"
	"github.com/manojsachwani/bearty/internal/theme"
)

func TestPreviewRowsFillPage(t *testing.T) {
	for _, style := range []string{"default"} {
		c := defaults()
		lines := strings.Split(renderPreview(c, c.Options()), "\n")
		// title, blank, page…, note, blank, label, window rows
		page := lines[2 : len(lines)-len(sampleNav)-3]
		window := lines[len(lines)-len(sampleNav):]
		for i, l := range append(page, window...) {
			if w := lipgloss.Width(l); w != previewWidth {
				t.Errorf("%s row %d width %d, want %d", style, i, w, previewWidth)
			}
		}
	}
}

func TestCustomColourFallback(t *testing.T) {
	c := defaults()
	c.InkSel, c.InkHex = custom, "#12"
	if c.Ink() != theme.Inks[0].Hex {
		t.Errorf("invalid hex should fall back, got %s", c.Ink())
	}
	c.InkHex = "#3D4E7A"
	if c.Ink() != "#3D4E7A" {
		t.Errorf("Ink = %s", c.Ink())
	}
}

func TestDefaultSavePath(t *testing.T) {
	c := defaults()
	c.Name = "Pumpkin Paper!"
	c.SavePath = ""
	if got := c.DefaultSavePath(); got != "./pumpkin-paper.theme" {
		t.Errorf("DefaultSavePath = %s", got)
	}
}

func TestFormsBuildForEveryEnv(t *testing.T) {
	saved := []library.Saved{library.Builtins[0]}
	for name, env := range map[string]Env{
		"no bear":     {Saved: saved},
		"bear":        {Slots: []string{"Notes"}, Saved: saved},
		"with backup": {Slots: []string{"Notes"}, Backups: []string{"Notes"}, AddedFonts: []string{"Inkina-Regular.otf"}, Saved: saved},
	} {
		for _, f := range []*huh.Form{newStartForm(defaults(), env), newEditForm(defaults(), env)} {
			f.Init()
			if f.View() == "" {
				t.Errorf("%s: empty form view", name)
			}
		}
	}
}

// Regression: a select with a description must still show its options.
func TestOutputOptionsVisible(t *testing.T) {
	c := defaults()
	f := newEditForm(c, Env{Slots: []string{"Notes"}})
	f.Init()
	for i := 0; i < 20 && f.GetFocusedField().GetKey() != "output"; i++ {
		f.NextGroup()
	}
	if v := f.View(); !strings.Contains(v, "Install into Bear") || !strings.Contains(v, "Just save it") {
		t.Errorf("output options missing:\n%s", v)
	}
}

func TestFromSavedRoundTrip(t *testing.T) {
	s := library.Builtins[0]
	c := fromSaved(s, nil)
	got := c.Saved()
	if got.Paper != s.Paper || got.Ink != s.Ink || got.Font != s.Font || got.Size != s.Size || got.Lines != s.Lines {
		t.Errorf("round trip: %+v != %+v", got, s)
	}
	// E-Paper's #F9F9F9 is the White preset, so the picker lands on it.
	if c.PaperSel != "#F9F9F9" {
		t.Errorf("PaperSel = %s", c.PaperSel)
	}
}

func TestRegionsAndHeadingHexRoundTrip(t *testing.T) {
	s := library.Builtins[0]
	s.Headings = "#112233"
	s.Sidebar = theme.SidebarColors{Background: "#EEEEEE"}
	s.NoteList = theme.NoteListColors{Title: "#000000"}

	c := fromSaved(s, nil)
	if c.Headings != custom || c.HeadingsHex != "#112233" || c.Regions != RegionsCustom {
		t.Fatalf("fromSaved: %+v", c)
	}
	got := c.Saved()
	if got.Headings != "#112233" || got.Sidebar.Background != "#EEEEEE" || got.NoteList.Title != "#000000" {
		t.Errorf("Saved: %+v", got)
	}

	// Switching back to automatic drops the overrides from the theme.
	c.Regions = RegionsAuto
	if o := c.Options(); !o.Sidebar.IsZero() || !o.NoteList.IsZero() {
		t.Errorf("automatic should ignore overrides: %+v", o)
	}
}

// next moves to the next field, crossing into the next visible group.
func next(f *huh.Form) {
	before := f.GetFocusedField().GetKey()
	f.NextField()
	if f.GetFocusedField().GetKey() == before {
		f.NextGroup()
	}
}

// advanceTo moves through the form until the field with key is focused.
func advanceTo(t *testing.T, f *huh.Form, key string) {
	t.Helper()
	for i := 0; i < 40 && f.GetFocusedField().GetKey() != key; i++ {
		next(f)
	}
	if got := f.GetFocusedField().GetKey(); got != key {
		t.Fatalf("never reached %q, stuck on %q", key, got)
	}
}

// Regression: choosing a custom heading colour must go straight to its hex input.
func TestCustomHeadingAsksForHex(t *testing.T) {
	c := defaults()
	f := newEditForm(c, Env{Slots: []string{"Notes"}})
	f.Init()
	advanceTo(t, f, "headings")
	c.Headings = custom
	next(f)
	if got := f.GetFocusedField().GetKey(); got != "headinghex" {
		t.Errorf("after Custom hex… focus is %q, want headinghex", got)
	}
}

func TestFontSourcesRoundTrip(t *testing.T) {
	faces := []fonts.Face{{Family: "Avenir Next", Style: "Regular", PostScript: "AvenirNext-Regular"}}
	for name, s := range map[string]library.Saved{
		"pick":      {Name: "A", Paper: "#F9F9F9", Ink: "#CF5624", Size: 17, Font: "Inkina-Regular", FontSource: "inkina"},
		"google":    {Name: "B", Paper: "#F9F9F9", Ink: "#CF5624", Size: 17, Font: "ReenieBeanie", FontSource: "google:Reenie Beanie"},
		"installed": {Name: "C", Paper: "#F9F9F9", Ink: "#CF5624", Size: 17, Font: "AvenirNext-Regular"},
	} {
		c := fromSaved(s, faces)
		got := c.Saved()
		if got.Font != s.Font || got.FontSource != s.FontSource {
			t.Errorf("%s: got font %q source %q, want %q %q", name, got.Font, got.FontSource, s.Font, s.FontSource)
		}
	}
}

func TestBuiltinsSurviveEditor(t *testing.T) {
	for _, b := range library.Builtins {
		got := fromSaved(b, nil).Saved()
		got.Builtin = b.Builtin
		if got != b {
			t.Errorf("%s changed in the editor:\n got %+v\nwant %+v", b.Name, got, b)
		}
	}
}

func TestPreviewFitsEveryBuiltin(t *testing.T) {
	for _, b := range library.Builtins {
		c := fromSaved(b, nil)
		lines := strings.Split(renderPreview(c, c.Options()), "\n")
		for i, l := range lines[2 : len(lines)-len(sampleNav)-3] {
			if w := lipgloss.Width(l); w != previewWidth {
				t.Errorf("%s: row %d width %d, want %d", b.Name, i, w, previewWidth)
			}
		}
	}
}

func TestQuitKey(t *testing.T) {
	q := tea.KeyPressMsg{Code: 'q', Text: "q"}

	m := model{choices: defaults(), help: help.New(), motion: &motion{off: true}}
	m.form = newStartForm(m.choices, Env{})
	m.form.Init()
	if _, cmd := m.Update(q); cmd == nil || cmd() != tea.Quit() {
		t.Error("q on a list should quit")
	}

	m = model{choices: defaults(), help: help.New(), motion: &motion{off: true}}
	m.form = newEditForm(m.choices, Env{})
	m.form.Init() // focuses the Name input
	mm, _ := m.Update(q)
	if mm.(model).aborted {
		t.Error("q in a text field should type, not quit")
	}
}

// Regression: options were near-black (colour 235) when the terminal didn't
// report a dark background. The theme must not depend on that guess.
func TestFormThemeIgnoresTerminalBackground(t *testing.T) {
	light, dark := formTheme(false), formTheme(true)
	for name, pair := range map[string][2]lipgloss.Style{
		"option":            {light.Focused.UnselectedOption, dark.Focused.UnselectedOption},
		"selected":          {light.Focused.SelectedOption, dark.Focused.SelectedOption},
		"description":       {light.Focused.Description, dark.Focused.Description},
		"blurred option":    {light.Blurred.UnselectedOption, dark.Blurred.UnselectedOption},
		"help key":          {light.Help.ShortKey, dark.Help.ShortKey},
		"input placeholder": {light.Focused.TextInput.Placeholder, dark.Focused.TextInput.Placeholder},
	} {
		if pair[0].GetForeground() != pair[1].GetForeground() {
			t.Errorf("%s differs between light and dark terminals", name)
		}
	}
	if light.Focused.UnselectedOption.GetForeground() != uiText {
		t.Errorf("unselected options should use uiText, got %v", light.Focused.UnselectedOption.GetForeground())
	}
}
