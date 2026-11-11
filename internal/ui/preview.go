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
	// A touch darker than Bear's own rules so they survive a terminal's palette.
	faint := theme.Mix(o.LineColor(), theme.Rule, 0.15)
	muted := theme.Mix(theme.Rule, paper, 0.45)
	inner := previewWidth - 2*gutter

	bg := lipgloss.NewStyle().Background(lipgloss.Color(paper))
	seg := func(fg string, s string) string { return bg.Foreground(lipgloss.Color(fg)).Render(s) }
	margin := seg(ink, strings.Repeat(" ", gutter))

	// blank fills the rest of a line with paper. Bear draws no page patterns,
	// so neither does the preview.
	pattern := func(from int, _ bool) string { return seg(faint, strings.Repeat(" ", max(inner-from, 0))) }

	var rows []string
	row := func(body string) { rows = append(rows, margin+body+margin) }
	// rich writes one line from styled parts, then fills the rest with the pattern.
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
		// One blank column keeps text off the grid lines.
		row(b.String() + seg(faint, " ") + pattern(w+1, false))
	}
	part := func(style lipgloss.Style, s string) [2]string { return [2]string{style.Render(s), s} }
	text := func(style lipgloss.Style, s string) { rich(part(style, s)) }
	ruled := func() { row(pattern(0, true)) }

	inkStyle := bg.Foreground(lipgloss.Color(ink))
	row(pattern(0, false))
	text(bg.Foreground(lipgloss.Color(o.HeadingColor())).Bold(true), sampleTitle)
	row(seg(theme.Rule, strings.Repeat("━", inner)))
	ruled()
	for _, l := range wrap(sampleBody, inner) {
		text(inkStyle, l)
		ruled()
	}
	rich(
		part(bg.Foreground(lipgloss.Color(acc.Markers)), "• "),
		part(inkStyle, "Lists with "),
		part(bg.Foreground(lipgloss.Color(acc.Links)).Underline(true), "a link"),
	)
	ruled()
	rich(
		part(bg.Foreground(lipgloss.Color(acc.Ticks)), "✓ "),
		part(bg.Foreground(lipgloss.Color(theme.Mix(ink, paper, 0.5))), "Done to-dos fade"),
	)
	ruled()

	caption := fmt.Sprintf("%s · %dpt", c.Font().Name, c.Size)
	if c.HeadingFont != "" {
		caption = fmt.Sprintf("%s + %s · %dpt", c.Font().Name, c.HeadingFontName(), c.Size)
	}
	if len([]rune(caption)) > inner {
		caption = fit(caption, inner)
	}
	text(bg.Foreground(lipgloss.Color(muted)), caption)
	row(pattern(0, false))

	page := lipgloss.JoinVertical(lipgloss.Left, rows...)
	title := lipgloss.NewStyle().Foreground(ember).Bold(true).Render(strings.ToUpper(c.Name))
	note := lipgloss.NewStyle().Foreground(faintFg).Render("Note")
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
