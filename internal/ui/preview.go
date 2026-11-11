package ui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/manojsachwani/bearty/internal/theme"
)

const (
	formWidth    = 46
	previewWidth = 42
	gutter       = 2 // paper margin on each side of the preview
)

const sampleTitle = "Make Bear yours"

const sampleBody = "Pick a paper, an ink and a typeface, then fine-tune any color you like."

func wrap(s string, width int) []string {
	var lines []string
	line := ""
	for _, word := range strings.Fields(s) {
		switch {
		case line == "":
			line = word
		case len(line)+1+len(word) <= width:
			line += " " + word
		default:
			lines = append(lines, line)
			line = word
		}
	}
	return append(lines, line)
}

// renderPreview draws a page in the colours of o, which may be mid-animation.
// Every segment carries the paper background, because a nested reset would
// otherwise clear it.
func renderPreview(c *Choices, o theme.Options) string {
	paper, ink := o.Paper, o.Ink
	acc := o.AccentsResolved()
	strong := theme.Rule // separators and secondary text, flipped for dark paper
	if theme.IsDark(paper) {
		strong = "#ECECEC"
	}
	muted := theme.Mix(strong, paper, 0.45)
	inner := previewWidth - 2*gutter

	bg := lipgloss.NewStyle().Background(lipgloss.Color(paper))
	seg := func(fg string, s string) string { return bg.Foreground(lipgloss.Color(fg)).Render(s) }
	margin := seg(ink, strings.Repeat(" ", gutter))

	// fill pads a line with paper up to the page width.
	fill := func(from int) string { return seg(ink, strings.Repeat(" ", max(inner-from, 0))) }

	var rows []string
	row := func(body string) { rows = append(rows, margin+body+margin) }
	// rich writes one line from styled parts, then pads it with paper.
	rich := func(parts ...[2]string) {
		var b strings.Builder
		w := 0
		for _, p := range parts {
			b.WriteString(p[0])
			w += lipgloss.Width(p[1])
		}
		if w >= inner {
			row(b.String())
			return
		}
		row(b.String() + fill(w))
	}
	part := func(style lipgloss.Style, s string) [2]string { return [2]string{style.Render(s), s} }
	text := func(style lipgloss.Style, s string) { rich(part(style, s)) }
	gap := func() { row(fill(0)) }

	inkStyle := bg.Foreground(lipgloss.Color(ink))
	gap()
	text(bg.Foreground(lipgloss.Color(o.HeadingColor())).Bold(true), sampleTitle)
	row(seg(strong, strings.Repeat("━", inner)))
	gap()
	for _, l := range wrap(sampleBody, inner) {
		text(inkStyle, l)
		gap()
	}
	rich(
		part(bg.Foreground(lipgloss.Color(acc.Markers)), "• "),
		part(inkStyle, "Lists with "),
		part(bg.Foreground(lipgloss.Color(acc.Links)).Underline(true), "a link"),
	)
	gap()
	rich(
		part(bg.Foreground(lipgloss.Color(acc.Ticks)), "✓ "),
		part(bg.Foreground(lipgloss.Color(theme.Mix(ink, paper, 0.5))), "Done to-dos fade"),
	)
	gap()

	// Details: a highlight and a tag, then a line of code.
	hl, tag, code := o.HighlightsResolved(), o.TagResolved(), o.CodeResolved()
	on := func(fg, bgc string) lipgloss.Style {
		return lipgloss.NewStyle().Foreground(lipgloss.Color(fg)).Background(lipgloss.Color(bgc))
	}
	rich(
		part(inkStyle, "A "),
		part(on(contrastOn(hl.Default), hl.Default), "highlight"),
		part(inkStyle, " and a "),
		part(on(tag.Text, tag.Background), " #tag "),
	)
	gap()
	rich(
		part(on(code.Keyword, code.Background), " let "),
		part(on(code.Text, code.Background), "total = "),
		part(on(code.Number, code.Background), "42"),
		part(on(code.Comment, code.Background), " // code "),
	)
	gap()

	caption := fmt.Sprintf("%s · %dpt", c.Font().Name, c.Size)
	if c.HeadingFont != "" {
		caption = fmt.Sprintf("%s + %s · %dpt", c.Font().Name, c.HeadingFontName(), c.Size)
	}
	if len([]rune(caption)) > inner {
		caption = fit(caption, inner)
	}
	text(bg.Foreground(lipgloss.Color(muted)), caption)
	gap()

	page := lipgloss.JoinVertical(lipgloss.Left, rows...)
	title := lipgloss.NewStyle().Foreground(ember).Bold(true).Render(strings.ToUpper(c.Name))
	mode := "Note"
	if theme.IsDark(o.Paper) {
		mode = "Note · dark mode"
	}
	note := lipgloss.NewStyle().Foreground(faintFg).Render(mode)
	if w := contrastWarnings(o); w != "" {
		note += "\n" + w
	}
	return lipgloss.JoinVertical(lipgloss.Left, title, "", page, note, "", renderWindow(o))
}

const sidebarWidth = 15

type navItem struct{ icon, label string }

var sampleNav = []navItem{{"▤", "Notes"}, {"☐", "Todo"}, {"◷", "Today"}, {"✉", "inbox"}, {"◎", "work"}, {"◌", "ideas"}, {"", ""}}

type noteItem struct{ title, preview, date string }

var sampleNotes = []noteItem{
	{"Weekly plan", "Errands, calls and the…", "8 hours ago"},
	{"Reading list", "Three books to finish…", "29 Oct"},
}

func fit(s string, w int) string {
	r := []rune(s)
	if len(r) > w {
		return string(r[:w-1]) + "…"
	}
	return s + strings.Repeat(" ", w-len(r))
}

// renderWindow draws Bear's sidebar and note list in the colours of o.
func renderWindow(o theme.Options) string {
	side, list := o.SidebarResolved(), o.NoteListResolved()
	paper := o.Paper

	cell := func(fg, bg, s string, bold bool) string {
		return lipgloss.NewStyle().Foreground(lipgloss.Color(fg)).Background(lipgloss.Color(bg)).Bold(bold).Render(s)
	}

	nav := make([]string, len(sampleNav))
	for i, n := range sampleNav {
		bg, iconFg, textFg := side.Background, side.Icons, side.Text
		if i == 0 { // selected row: Bear draws it in the paper colour
			bg, iconFg, textFg = side.Selected, paper, paper
		}
		if n.label == "" {
			nav[i] = cell(textFg, bg, strings.Repeat(" ", sidebarWidth), false)
			continue
		}
		nav[i] = cell(iconFg, bg, " "+n.icon+" ", false) + cell(textFg, bg, fit(n.label, sidebarWidth-3), false)
	}

	listW := previewWidth - sidebarWidth
	var notes []string
	for i, n := range sampleNotes {
		bg := paper
		if i == 1 {
			bg = list.Selected
		}
		notes = append(notes,
			cell(list.Title, bg, " "+fit(n.title, listW-1), true),
			cell(list.Preview, bg, " "+fit(n.preview, listW-1), false),
			cell(o.Ink, bg, " ⚲ ", false)+cell(list.Date, bg, fit(n.date, listW-3), false),
		)
		if i == 0 {
			notes = append(notes, cell(o.LineColor(), paper, " "+strings.Repeat("─", listW-2)+" ", false))
		}
	}

	rows := make([]string, len(nav))
	for i := range nav {
		rows[i] = nav[i] + notes[i]
	}
	label := lipgloss.NewStyle().Foreground(faintFg).Render("Sidebar and note list")
	return lipgloss.JoinVertical(lipgloss.Left, label, lipgloss.JoinVertical(lipgloss.Left, rows...))
}

// contrastOn picks readable text for a highlight, like Bear does.
func contrastOn(bg string) string {
	if theme.Contrast(bg, theme.Rule) >= theme.Contrast(bg, "#F4F4F4") {
		return theme.Rule
	}
	return "#F4F4F4"
}

// contrastWarnings flags colour pairs that are hard to read (WCAG: 4.5:1 for
// body text, 3:1 for larger or secondary text).
func contrastWarnings(o theme.Options) string {
	v, err := theme.Build(o).Values()
	if err != nil {
		return ""
	}
	str := func(k string) string { s, _ := v[k].(string); return s }
	checks := []struct {
		what, fg, bg string
		min          float64
	}{
		{"Text", str("base.text color"), str("base.background color"), 4.5},
		{"Headings", str("editor.headers.text color"), str("base.background color"), 3},
		{"Links", str("editor.link color"), str("base.background color"), 3},
		{"Sidebar text", str("sidebar.text color"), str("sidebar.background color"), 4.5},
		{"Note titles", str("notes.title color"), str("base.background color"), 4.5},
		{"Code", str("editor.code.text color"), str("editor.code.background color"), 4.5},
	}
	var lines []string
	for _, c := range checks {
		if theme.ValidateHex(c.fg) != nil || theme.ValidateHex(c.bg) != nil {
			continue
		}
		if r := theme.Contrast(c.fg, c.bg); r < c.min {
			lines = append(lines, fmt.Sprintf("⚠ %s is hard to read (%.1f:1)", c.what, r))
		}
	}
	if len(lines) > 2 {
		lines = append(lines[:2], fmt.Sprintf("⚠ and %d more", len(lines)-2))
	}
	return lipgloss.NewStyle().Foreground(ember).Render(strings.Join(lines, "\n"))
}
