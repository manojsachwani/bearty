// Package capture turns an existing Bear theme into an editable bearty theme.
//
// Everything bearty models (paper, ink, fonts, typography, sidebar, note
// list, accents, highlighters, code, tags) becomes a normal setting, kept
// only where it differs from what bearty would derive. Whatever is left over
// goes into Extra, so building the captured theme reproduces the original's
// values exactly.
package capture

import (
	"fmt"
	"strings"

	"github.com/manojsachwani/bearty/internal/fonts"
	"github.com/manojsachwani/bearty/internal/library"
	"github.com/manojsachwani/bearty/internal/theme"
)

// File captures a .theme file. dir is Bear's themes folder, used when the
// theme builds on another one; it may be empty.
func File(path, dir, name string) (library.Saved, error) {
	m, err := theme.Load(path, dir)
	if err != nil {
		return library.Saved{}, err
	}
	return Values(theme.Resolve(theme.Flatten(m)), name)
}

// Values captures a theme from its resolved values.
func Values(v map[string]any, name string) (library.Saved, error) {
	str := func(k string) string { s, _ := v[k].(string); return s }
	num := func(k string) float64 { f, _ := v[k].(float64); return f }

	s := library.Saved{Name: name, Paper: str("base.background color"), Ink: str("base.text color")}
	if theme.ValidateHex(s.Paper) != nil || theme.ValidateHex(s.Ink) != nil {
		return s, fmt.Errorf("theme has no usable background and text colors")
	}
	s.Paper, s.Ink = strings.ToUpper(s.Paper), strings.ToUpper(s.Ink)

	// Fonts and typography.
	s.Font = str("editor.text font")
	if p, ok := fonts.PickByPostScript(s.Font); ok {
		s.FontSource = p.Source
	}
	if hf := str("editor.headers.font"); hf != "" && hf != headingDefault(s.Font) {
		s.HeadingFont = hf
		if p, ok := fonts.PickByPostScript(hf); ok {
			s.HeadingFontSource = p.Source
		}
	}
	s.Size = int(num("editor.text size"))
	if s.Size == 0 {
		s.Size = 17
	}
	s.Type = theme.Typography{
		LineHeight:       num("editor.line height multiplier"),
		LineWidth:        num("editor.line width multiplier"),
		ParagraphSpacing: num("editor.paragraph spacing multiplier"),
		ParagraphIndent:  num("editor.paragraph indent multiplier"),
	}
	if s.Type.LineHeight == theme.DefaultLineHeight {
		s.Type.LineHeight = 0
	}
	if scale := num("editor.headers.modular scale"); scale != 0 && scale != theme.DefaultHeadingScale {
		s.HeadingScale = scale
	}
	if cf := str("editor.code.font"); cf != "" && cf != theme.DefaultCodeFont {
		s.Code.Font = cf
	}

	// Colours: set each one, then drop it again wherever bearty would derive
	// the same value anyway, so the saved theme stays short.
	base := s.Options()
	keep := func(field *string, key, derived string) {
		if c := strings.ToUpper(str(key)); theme.ValidateHex(c) == nil && c != derived {
			*field = c
		}
	}
	keep(&s.Headings, "editor.headers.text color", base.HeadingColor())
	if s.Headings == theme.Rule {
		s.Headings = theme.HeadingsBlack
	}
	o := s.Options()
	keep(&s.Lines, "base.stroke color", o.LineColor())
	o = s.Options()

	acc := o.AccentsResolved()
	keep(&s.Accents.Links, "editor.link color", acc.Links)
	keep(&s.Accents.Markers, "editor.list marker color", acc.Markers)
	keep(&s.Accents.Ticks, "editor.task.check color", acc.Ticks)

	keep(&s.Sidebar.Background, "sidebar.background color", o.SidebarResolved().Background)
	o = s.Options()
	side := o.SidebarResolved()
	keep(&s.Sidebar.Text, "sidebar.text color", side.Text)
	keep(&s.Sidebar.Icons, "sidebar.icon color", side.Icons)
	keep(&s.Sidebar.Selected, "sidebar.background secondary color", side.Selected)

	o = s.Options()
	list := o.NoteListResolved()
	keep(&s.NoteList.Title, "notes.title color", list.Title)
	keep(&s.NoteList.Preview, "notes.subtitle color", list.Preview)
	keep(&s.NoteList.Date, "notes.date color", list.Date)
	keep(&s.NoteList.Selected, "notes.selection background color", list.Selected)

	o = s.Options()
	hl := o.HighlightsResolved()
	keep(&s.Highlights.Default, "editor.highlighter.default.background color", hl.Default)
	keep(&s.Highlights.Red, "editor.highlighter.red.background color", hl.Red)
	keep(&s.Highlights.Blue, "editor.highlighter.blue.background color", hl.Blue)
	keep(&s.Highlights.Green, "editor.highlighter.green.background color", hl.Green)
	keep(&s.Highlights.Yellow, "editor.highlighter.yellow.background color", hl.Yellow)
	keep(&s.Highlights.Purple, "editor.highlighter.purple.background color", hl.Purple)

	code := o.CodeResolved()
	keep(&s.Code.Background, "editor.code.background color", code.Background)
	keep(&s.Code.Text, "editor.code.text color", code.Text)
	syn := "editor.code.syntax highlight."
	keep(&s.Code.Comment, syn+"comment", code.Comment)
	keep(&s.Code.Constant, syn+"constant", code.Constant)
	keep(&s.Code.Number, syn+"number", code.Number)
	keep(&s.Code.String, syn+"string", code.String)
	keep(&s.Code.Entity, syn+"entity", code.Entity)
	keep(&s.Code.Keyword, syn+"keyword", code.Keyword)
	keep(&s.Code.Function, syn+"function", code.Function)
	keep(&s.Code.Variable, syn+"variable", code.Variable)

	tag := o.TagResolved()
	keep(&s.Tag.Background, "editor.tag.background color", tag.Background)
	keep(&s.Tag.Text, "editor.tag.text color", tag.Text)

	// Everything still different goes into Extra. An extra key can change
	// what other keys' references resolve to, so repeat until stable.
	for range 5 {
		built, err := theme.Build(s.Options()).Values()
		if err != nil {
			return s, err
		}
		changed := false
		for k, want := range v {
			if !same(built[k], want) {
				if s.Extra == nil {
					s.Extra = map[string]any{}
				}
				s.Extra[k] = want
				changed = true
			}
		}
		if !changed {
			return s, nil
		}
	}
	return s, fmt.Errorf("theme could not be reproduced exactly")
}

// headingDefault is the heading font bearty pairs with a text font.
func headingDefault(text string) string {
	if p, ok := fonts.PickByPostScript(text); ok {
		return p.HeadingFont()
	}
	return text
}

func same(a, b any) bool {
	as, aok := a.(string)
	bs, bok := b.(string)
	if aok && bok {
		return strings.EqualFold(as, bs)
	}
	return a == b
}
