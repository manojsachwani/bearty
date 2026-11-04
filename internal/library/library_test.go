package library

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/manojsachwani/bearty/internal/theme"
)

func TestListHasBuiltinsWhenEmpty(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	list, err := List()
	if err != nil || len(list) != len(Builtins) || list[0].Name != "E-Paper" || list[1].Name != "Paper Pro" || list[2].Name != "All these things" {
		t.Fatalf("List = %v, %v", list, err)
	}
}

func TestSaveOverridesBuiltinAndRoundTrips(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	edited := Builtins[0]
	edited.Ink = "#3D4F74"
	edited.Builtin = false
	if _, err := Save(edited); err != nil {
		t.Fatal(err)
	}
	Save(Saved{Name: "Night Notes", Paper: "#EFDEC4", Ink: "#2F2F2F", Font: "Noteworthy-Light", Size: 17})

	list, err := List()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != len(Builtins)+1 {
		t.Fatalf("want built-ins + Night Notes, got %v", list)
	}
	if list[0].Name != "E-Paper" || list[0].Ink != "#3D4F74" || list[0].Builtin {
		t.Errorf("override not applied: %+v", list[0])
	}
	if last := list[len(list)-1]; last.Name != "Night Notes" {
		t.Errorf("last = %+v", last)
	}
}

func TestListSkipsBadFiles(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	dir, _ := Dir()
	os.MkdirAll(dir, 0o755)
	os.WriteFile(filepath.Join(dir, "broken.json"), []byte("{"), 0o644)
	os.WriteFile(filepath.Join(dir, "badhex.json"), []byte(`{"name":"X","paper":"red","ink":"#000000"}`), 0o644)

	list, err := List()
	if err == nil {
		t.Error("expected a warning about skipped files")
	}
	if len(list) != len(Builtins) {
		t.Errorf("bad files should be skipped, got %v", list)
	}
}

func TestSlug(t *testing.T) {
	if got := Slug("E-Paper (mine)!"); got != "e-paper-mine" {
		t.Errorf("Slug = %s", got)
	}
}

func TestRegionColoursRoundTrip(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	in := Saved{Name: "Mine", Paper: "#F9F9F9", Ink: "#CF5624", Font: "Inkina-Regular", Size: 19,
		Headings: "#123456"}
	in.Sidebar.Background = "#EEEEEE"
	in.NoteList.Title = "#000000"
	path, _ := Save(in)
	b, _ := os.ReadFile(path)
	if !strings.Contains(string(b), `"sidebar"`) || strings.Contains(string(b), `"icons"`) {
		t.Errorf("only set fields should be written:\n%s", b)
	}
	list, _ := List()
	var got Saved
	for _, s := range list {
		if s.Name == "Mine" {
			got = s
		}
	}
	if got.Sidebar.Background != "#EEEEEE" || got.NoteList.Title != "#000000" || got.Headings != "#123456" {
		t.Errorf("round trip: %+v", got)
	}
	// A theme with no overrides writes no region blocks.
	path, _ = Save(Saved{Name: "Plain", Paper: "#F9F9F9", Ink: "#CF5624", Size: 17})
	if b, _ := os.ReadFile(path); strings.Contains(string(b), "sidebar") {
		t.Errorf("empty overrides should be omitted:\n%s", b)
	}
}

// Paper Pro was designed in Bear as a Red Graphite override; the built-in
// must rebuild that file exactly.
func TestPaperProMatchesOriginal(t *testing.T) {
	want, err := os.ReadFile("testdata/paper-pro.theme")
	if err != nil {
		t.Fatal(err)
	}
	var pro Saved
	for _, b := range Builtins {
		if b.Name == "Paper Pro" {
			pro = b
		}
	}
	got, err := theme.Build(pro.Options()).JSON()
	if err != nil {
		t.Fatal(err)
	}
	var a, b any
	json.Unmarshal(want, &a)
	json.Unmarshal(got, &b)
	if !reflect.DeepEqual(a, b) {
		t.Errorf("Paper Pro differs from the original:\n%s", got)
	}
}

// All these things was designed in Bear as a High Contrast override, with
// its typography set in Bear's preferences. Colours must match the file;
// the type settings come from those preferences.
func TestAllTheseThingsMatchesOriginal(t *testing.T) {
	raw, err := os.ReadFile("testdata/all-these-things.theme")
	if err != nil {
		t.Fatal(err)
	}
	var want map[string]any
	json.Unmarshal(raw, &want)
	editor := want["editor"].(map[string]any)
	editor["text size"] = 24.0
	editor["line height multiplier"] = 1.4
	editor["line width multiplier"] = 64.0
	editor["paragraph spacing multiplier"] = 0.4
	editor["paragraph indent multiplier"] = 0.2
	editor["headers"].(map[string]any)["font"] = "PatrickHand-Regular"

	var b Saved
	for _, s := range Builtins {
		if s.Name == "All these things" {
			b = s
		}
	}
	out, err := theme.Build(b.Options()).JSON()
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	json.Unmarshal(out, &got)
	if !reflect.DeepEqual(want, got) {
		for _, sec := range []string{"base", "notes", "sidebar", "placeholder", "editor"} {
			if !reflect.DeepEqual(want[sec], got[sec]) {
				w, _ := json.MarshalIndent(want[sec], "", " ")
				g, _ := json.MarshalIndent(got[sec], "", " ")
				t.Errorf("section %s differs:\nwant %s\n got %s", sec, w, g)
			}
		}
	}
}
