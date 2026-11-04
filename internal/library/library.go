// Package library stores the user's themes as editable JSON files in
// ~/.config/bearty/themes. Built-in themes ship in code; saving one under
// the same name writes a user copy that replaces it.
package library

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/manojsachwani/bearty/internal/fonts"
	"github.com/manojsachwani/bearty/internal/theme"
)

// Saved is one theme's settings. Every field can be changed in the CLI or by
// editing the file.
type Saved struct {
	Name       string `json:"name"`
	Paper      string `json:"paper"`
	Ink        string `json:"ink"`
	Font       string `json:"font"`                  // PostScript name
	FontSource string `json:"font_source,omitempty"` // "inkina" or "google:<Family>" when it must be downloaded

	// Headings use the text font unless these are set.
	HeadingFont       string `json:"heading_font,omitempty"`
	HeadingFontSource string `json:"heading_font_source,omitempty"`
	Size              int    `json:"size"`
	Headings          string `json:"headings,omitempty"` // "ink", "black" or #RRGGBB
	Lines             string `json:"lines,omitempty"`    // #RRGGBB; derived from paper when empty

	Sidebar  theme.SidebarColors  `json:"sidebar,omitzero"`
	NoteList theme.NoteListColors `json:"notelist,omitzero"`
	Accents  theme.Accents        `json:"accents,omitzero"`
	Type     theme.Typography     `json:"typography,omitzero"`

	Slot string `json:"slot,omitempty"` // Bear theme it was last installed over

	Builtin bool `json:"-"`
}

// Builtins are always available, even before anything is saved.
var Builtins = []Saved{
	{
		// Sampled from the reference screenshots: grey-white e-paper, pumpkin
		// ink, faint rules and the large handwriting size.
		Name:       "E-Paper",
		Paper:      "#F9F9F9",
		Ink:        "#CF5624",
		Font:       fonts.InkinaPostScript,
		FontSource: fonts.SourceInkina,
		Size:       19,
		Headings:   theme.HeadingsInk,
		Lines:      "#EAEAEA",
		Builtin:    true,
	},
	{
		// Warm paper, pumpkin Inkina ink and black headings, with a navy
		// sidebar selection and navy note previews.
		Name:       "Paper Pro",
		Paper:      "#F7F7F5",
		Ink:        "#CF5624",
		Font:       fonts.InkinaPostScript,
		FontSource: fonts.SourceInkina,
		Size:       19,
		Headings:   theme.HeadingsBlack,
		Lines:      "#EAEAEA",
		Sidebar:    theme.SidebarColors{Background: "#F7F7F5", Text: "#3D4F74", Selected: "#3D4F74"},
		NoteList:   theme.NoteListColors{Preview: "#3D4F74"},
		Builtin:    true,
	},
	{
		// Blush paper and navy ink with green headings, a different colour
		// for every detail, and Indie Flower text under Patrick Hand headings.
		Name:              "All these things",
		Paper:             "#E4C3C3",
		Ink:               "#3D4F74",
		Font:              "IndieFlower-Regular",
		FontSource:        fonts.GoogleSource("Indie Flower"),
		HeadingFont:       "PatrickHand-Regular",
		HeadingFontSource: fonts.GoogleSource("Patrick Hand"),
		Size:              24,
		Headings:          "#1E7115",
		Sidebar:           theme.SidebarColors{Background: "#C7ACAC", Text: "#3D4F74", Selected: "#AB744C"},
		NoteList:          theme.NoteListColors{Selected: "#D1B3B3"},
		Accents:           theme.Accents{Links: "#307C36", Markers: "#6913B7", Ticks: "#4B9120"},
		Type:              theme.Typography{LineHeight: 1.4, LineWidth: 64, ParagraphSpacing: 0.4, ParagraphIndent: 0.2},
		Builtin:           true,
	},
}

// Options converts the saved settings for the theme builder.
func (s Saved) Options() theme.Options {
	f := theme.Font{Name: s.Font, PostScript: s.Font, Heading: s.Font}
	if p, ok := fonts.PickByPostScript(s.Font); ok {
		f = theme.Font{Name: p.Name, PostScript: p.PostScript, Heading: p.HeadingFont()}
	}
	if s.HeadingFont != "" {
		f.Heading = s.HeadingFont
	}
	return theme.Options{
		Name: s.Name, Paper: s.Paper, Ink: s.Ink, Font: f, Size: s.Size,
		Headings: s.Headings, Lines: s.Lines,
		Sidebar: s.Sidebar, NoteList: s.NoteList, Accents: s.Accents, Type: s.Type,
	}
}

// Dir is where user themes live. XDG_CONFIG_HOME is honoured.
func Dir() (string, error) {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "bearty", "themes"), nil
}

var nonSlug = regexp.MustCompile(`[^a-z0-9]+`)

// TypographyBackupPath holds the user's own Bear typography from before
// bearty first changed it.
func TypographyBackupPath() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(dir), "bear-typography.original.json"), nil
}

// Slug turns a theme name into a file name stem.
func Slug(name string) string {
	s := strings.Trim(nonSlug.ReplaceAllString(strings.ToLower(name), "-"), "-")
	if s == "" {
		return "bearty"
	}
	return s
}

// Path is the file a theme with this name is saved to.
func Path(name string) (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, Slug(name)+".json"), nil
}

// List returns the built-ins followed by the user's themes, sorted by name.
// A user theme with a built-in's name replaces it.
func List() ([]Saved, error) {
	bySlug := map[string]Saved{}
	for _, b := range Builtins {
		bySlug[Slug(b.Name)] = b
	}

	dir, err := Dir()
	if err != nil {
		return nil, err
	}
	paths, _ := filepath.Glob(filepath.Join(dir, "*.json"))
	var bad []string
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			bad = append(bad, p)
			continue
		}
		var s Saved
		if err := json.Unmarshal(b, &s); err != nil || s.Name == "" ||
			theme.ValidateHex(s.Paper) != nil || theme.ValidateHex(s.Ink) != nil {
			bad = append(bad, filepath.Base(p))
			continue
		}
		bySlug[Slug(s.Name)] = s
	}

	list := make([]Saved, 0, len(bySlug))
	for _, s := range bySlug {
		list = append(list, s)
	}
	// Built-ins first, in their own order, even once edited; then the rest by name.
	rank := func(s Saved) int {
		for i, b := range Builtins {
			if Slug(b.Name) == Slug(s.Name) {
				return i
			}
		}
		return len(Builtins)
	}
	sort.Slice(list, func(i, j int) bool {
		if ri, rj := rank(list[i]), rank(list[j]); ri != rj {
			return ri < rj
		}
		return strings.ToLower(list[i].Name) < strings.ToLower(list[j].Name)
	})
	if len(bad) > 0 {
		return list, errors.New("skipped unreadable theme files: " + strings.Join(bad, ", "))
	}
	return list, nil
}

// Save writes the theme to its file and returns the path.
func Save(s Saved) (string, error) {
	path, err := Path(s.Name)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return "", err
	}
	return path, os.WriteFile(path, append(b, '\n'), 0o644)
}
