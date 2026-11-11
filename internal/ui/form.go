// Package ui is the interactive theme builder: a Huh form beside a live preview.
package ui

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"

	"github.com/manojsachwani/bearty/internal/fonts"
	"github.com/manojsachwani/bearty/internal/library"
	"github.com/manojsachwani/bearty/internal/theme"
)

const (
	custom = "custom"

	ModeCreate  = "create"
	ModeEdit    = "edit"
	ModeRestore = "restore"

	OutputInstall = "install"
	OutputExport  = "export"
	OutputBoth    = "both"
	OutputNone    = "none"

	RegionsAuto   = "auto" // also used for accents
	RegionsCustom = "custom"

	FontPicks     = "picks"
	FontInstalled = "installed"
	FontGoogle    = "google"
)

// Env describes what the form is offered against.
type Env struct {
	Slots      []string            // Bear themes that can be replaced
	Backups    []string            // slots with a saved original
	AddedFonts []string            // fonts bearty copied into Bear.app
	Saved      []library.Saved     // built-in and user themes
	Faces      []fonts.Face        // fonts installed on this Mac and in Bear
	LoadFaces  func() []fonts.Face // scans for Faces in the background when set
	Version    string
	NoMotion   bool // skip the splash screen and animations

	TypographyBackup bool // bearty saved the user's Bear typography and can put it back
}

// Choices holds everything the form collects.
type Choices struct {
	Mode     string
	SavedSel string // slug of the theme being edited

	Name     string
	PaperSel string // preset hex or "custom"
	PaperHex string
	InkSel   string
	InkHex   string
	Size     int
	Lines    string

	Headings    string // theme.HeadingsInk, theme.HeadingsBlack or "custom"
	HeadingsHex string

	FontSrc    string // FontPicks, FontInstalled or FontGoogle
	FontPick   string // PostScript name of a pick
	FontQuery  string // narrows the installed family list
	FontFamily string // installed family
	FontFace   string // installed face, by PostScript name
	GoogleName string // Google Fonts family
	GooglePS   string // its PostScript name, once looked up

	HeadingFont   string // PostScript name for headings; empty = same as the text
	headingSource string // where a heading font from a saved theme downloads from

	Type theme.Typography

	Regions  string // RegionsAuto or RegionsCustom
	Sidebar  theme.SidebarColors
	NoteList theme.NoteListColors

	AccentMode string // RegionsAuto: links, markers and ticks follow the ink
	Accents    theme.Accents

	Output   string
	SavePath string
	Slot     string
	Confirm  bool

	SetTypography bool // also write Bear's Settings → Typography to match

	RestoreSlot       string
	RestoreConfirm    bool
	RemoveFonts       bool
	RestoreTypography bool

	faces []fonts.Face // installed fonts, for resolving a family to a face
}

func defaults() *Choices {
	return &Choices{
		Mode:              ModeCreate,
		Name:              "My Theme",
		PaperSel:          theme.Papers[0].Hex,
		InkSel:            theme.Inks[0].Hex,
		Size:              theme.Sizes[1].Pt,
		Headings:          theme.HeadingsInk,
		FontSrc:           FontPicks,
		FontPick:          fonts.Picks[0].PostScript,
		Regions:           RegionsAuto,
		AccentMode:        RegionsAuto,
		Output:            OutputInstall,
		SetTypography:     true,
		RestoreTypography: true,
	}
}

// presetOrCustom selects the preset matching hex, or "custom" with hex filled in.
func presetOrCustom(presets []theme.Swatch, hex string) (sel, customHex string) {
	for _, p := range presets {
		if strings.EqualFold(p.Hex, hex) {
			return p.Hex, ""
		}
	}
	return custom, strings.ToUpper(hex)
}

// fromSaved fills the editor with a saved theme's settings.
func fromSaved(s library.Saved, faces []fonts.Face) *Choices {
	c := defaults()
	c.faces = faces
	c.Mode, c.SavedSel = ModeEdit, library.Slug(s.Name)
	c.Name = s.Name
	c.PaperSel, c.PaperHex = presetOrCustom(theme.Papers, s.Paper)
	c.InkSel, c.InkHex = presetOrCustom(theme.Inks, s.Ink)
	c.Size, c.Lines, c.Slot = s.Size, s.Lines, s.Slot

	switch {
	case theme.ValidateHex(s.Headings) == nil:
		c.Headings, c.HeadingsHex = custom, strings.ToUpper(s.Headings)
	case s.Headings != "":
		c.Headings = s.Headings
	}

	if _, ok := fonts.PickByPostScript(s.Font); ok {
		c.FontSrc, c.FontPick = FontPicks, s.Font
	} else if fam, ok := fonts.GoogleFamily(s.FontSource); ok {
		c.FontSrc, c.GoogleName, c.GooglePS = FontGoogle, fam, s.Font
	} else if f, ok := fonts.ByPostScript(faces, s.Font); ok {
		c.FontSrc, c.FontFamily, c.FontFace = FontInstalled, f.Family, f.PostScript
	} else if s.Font != "" {
		c.FontSrc, c.FontPick = FontPicks, s.Font // listed as "from file"
	}

	c.Sidebar, c.NoteList = s.Sidebar, s.NoteList
	if !s.Sidebar.IsZero() || !s.NoteList.IsZero() {
		c.Regions = RegionsCustom
	}
	c.Accents = s.Accents
	c.HeadingFont, c.headingSource, c.Type = s.HeadingFont, s.HeadingFontSource, s.Type
	if !s.Accents.IsZero() {
		c.AccentMode = RegionsCustom
	}
	return c
}

func pick(sel, hex, fallback string) string {
	if sel != custom {
		return sel
	}
	if theme.ValidateHex(hex) == nil {
		return hex
	}
	return fallback
}

// Paper is the chosen page colour.
func (c *Choices) Paper() string { return pick(c.PaperSel, c.PaperHex, theme.Papers[0].Hex) }

// Ink is the chosen text colour.
func (c *Choices) Ink() string { return pick(c.InkSel, c.InkHex, theme.Inks[0].Hex) }

// Font is the chosen typeface, with the heading font if one was chosen.
// A Google font that isn't one of the picks has no PostScript name until it
// has been looked up.
func (c *Choices) Font() theme.Font {
	f := c.textFont()
	if c.HeadingFont != "" {
		f.Heading = c.HeadingFont
	}
	return f
}

// HeadingFontSource says where the heading font downloads from, if anywhere.
func (c *Choices) HeadingFontSource() string {
	if c.HeadingFont == "" {
		return fonts.SourceInstalled
	}
	if p, ok := fonts.PickByPostScript(c.HeadingFont); ok {
		return p.Source
	}
	return c.headingSource
}

// HeadingFontName is the heading font's display name.
func (c *Choices) HeadingFontName() string {
	if p, ok := fonts.PickByPostScript(c.HeadingFont); ok {
		return p.Name
	}
	return c.HeadingFont
}

func (c *Choices) textFont() theme.Font {
	switch c.FontSrc {
	case FontInstalled:
		ps := c.FontFace
		styles := fonts.Styles(c.faces, c.FontFamily)
		if f, ok := fonts.ByPostScript(styles, ps); !ok && len(styles) > 0 {
			ps = styles[0].PostScript // the family changed; take its first style
		} else if ok {
			ps = f.PostScript
		}
		return theme.Font{Name: c.FontFamily, PostScript: ps, Heading: ps}
	case FontGoogle:
		ps := c.GooglePS
		for _, p := range fonts.Picks {
			if fam, ok := fonts.GoogleFamily(p.Source); ok && strings.EqualFold(fam, strings.TrimSpace(c.GoogleName)) {
				ps = p.PostScript
			}
		}
		return theme.Font{Name: c.GoogleName, PostScript: ps, Heading: ps}
	}
	if p, ok := fonts.PickByPostScript(c.FontPick); ok {
		return theme.Font{Name: p.Name, PostScript: p.PostScript, Heading: p.HeadingFont()}
	}
	return theme.Font{Name: c.FontPick, PostScript: c.FontPick, Heading: c.FontPick}
}

// FontSource says where the font must be downloaded from, if anywhere.
func (c *Choices) FontSource() string {
	switch c.FontSrc {
	case FontGoogle:
		return fonts.GoogleSource(strings.TrimSpace(c.GoogleName))
	case FontPicks:
		if p, ok := fonts.PickByPostScript(c.FontPick); ok {
			return p.Source
		}
	}
	return fonts.SourceInstalled
}

// Restoring reports whether the user chose to restore instead of create.
func (c *Choices) Restoring() bool { return c.Mode == ModeRestore }

// Installing and Exporting report the chosen outputs.
func (c *Choices) Installing() bool { return c.Output == OutputInstall || c.Output == OutputBoth }
func (c *Choices) Exporting() bool  { return c.Output == OutputExport || c.Output == OutputBoth }

// headings resolves the headings choice to "ink", "black" or a hex colour.
func (c *Choices) headings() string {
	if c.Headings == custom {
		return pick(custom, c.HeadingsHex, theme.HeadingsInk)
	}
	return c.Headings
}

// accents returns the link, marker and tick overrides, or none when they follow the ink.
func (c *Choices) accents() theme.Accents {
	if c.AccentMode != RegionsCustom {
		return theme.Accents{}
	}
	return c.Accents
}

// regions returns the sidebar and note list overrides, or none when automatic.
func (c *Choices) regions() (theme.SidebarColors, theme.NoteListColors) {
	if c.Regions != RegionsCustom {
		return theme.SidebarColors{}, theme.NoteListColors{}
	}
	return c.Sidebar, c.NoteList
}

// Options converts the choices for the theme builder.
func (c *Choices) Options() theme.Options {
	side, list := c.regions()
	return theme.Options{
		Name: c.Name, Paper: c.Paper(), Ink: c.Ink(), Font: c.Font(), Size: c.Size,
		Headings: c.headings(), Lines: c.Lines, Sidebar: side, NoteList: list,
		Accents: c.accents(), Type: c.Type,
	}
}

// Saved converts the choices for the theme library.
func (c *Choices) Saved() library.Saved {
	side, list := c.regions()
	return library.Saved{
		Name: strings.TrimSpace(c.Name), Paper: c.Paper(), Ink: c.Ink(),
		Font: c.Font().PostScript, FontSource: c.FontSource(), Size: c.Size,
		Headings: c.headings(), Lines: c.Lines, Slot: c.Slot,
		Sidebar: side, NoteList: list, Accents: c.accents(),
		HeadingFont: c.HeadingFont, HeadingFontSource: c.HeadingFontSource(), Type: c.Type,
	}
}

// DefaultSavePath is ./<name>.theme.
func (c *Choices) DefaultSavePath() string { return "./" + library.Slug(c.Name) + ".theme" }

func swatch(hex string) string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(hex)).Render("██")
}

func swatchOptions(presets []theme.Swatch) []huh.Option[string] {
	opts := make([]huh.Option[string], 0, len(presets)+1)
	for _, s := range presets {
		opts = append(opts, huh.NewOption(swatch(s.Hex)+" "+s.Name, s.Hex))
	}
	return append(opts, huh.NewOption("   Custom color…", custom))
}

// colourInput edits one optional colour. While empty it shows, and uses,
// the automatic colour for the current paper and ink.
func colourInput(c *Choices, key, title string, v *string, effective func(theme.Options) string) huh.Field {
	return newOptionalPicker(key, title, v, func() string {
		saved := *v
		*v = "" // what it would be if left empty
		auto := effective(c.Options())
		*v = saved
		return auto
	})
}

// pickOptions lists the handwriting picks, marking ones that download.
func pickOptions(c *Choices, faces []fonts.Face) []huh.Option[string] {
	tag := lipgloss.NewStyle().Foreground(dim)
	var opts []huh.Option[string]
	known := false
	for _, p := range fonts.Picks {
		label := p.Name
		_, have := fonts.ByPostScript(faces, p.PostScript)
		switch {
		case p.Source != fonts.SourceInstalled && !have:
			label += tag.Render("  ↓ download")
		case !have:
			continue // a Mac font that isn't on this Mac
		}
		opts = append(opts, huh.NewOption(label, p.PostScript))
		known = known || p.PostScript == c.FontPick
	}
	if !known && c.FontPick != "" {
		opts = append(opts, huh.NewOption(c.FontPick+tag.Render("  from file"), c.FontPick))
	}
	return opts
}

// familyOptions lists installed families containing query. With no match
// it offers a single empty choice that fails validation.
func familyOptions(c *Choices, families []string) []huh.Option[string] {
	q := strings.ToLower(strings.TrimSpace(c.FontQuery))
	var opts []huh.Option[string]
	current := false
	for _, f := range families {
		if q == "" || strings.Contains(strings.ToLower(f), q) {
			opts = append(opts, huh.NewOption(f, f))
			current = current || f == c.FontFamily
		}
	}
	if len(opts) == 0 {
		return []huh.Option[string]{huh.NewOption(lipgloss.NewStyle().Foreground(dim).Render("No fonts match"), "")}
	}
	if !current {
		opts[0] = opts[0].Selected(true) // move the cursor to the first match
	}
	return opts
}

// headingFontOptions offers the text font or any handwriting pick for headings.
func headingFontOptions(c *Choices, faces []fonts.Face) []huh.Option[string] {
	tag := lipgloss.NewStyle().Foreground(dim)
	opts := []huh.Option[string]{huh.NewOption("Same as the text", "")}
	known := c.HeadingFont == ""
	for _, p := range fonts.Picks {
		label := p.Name
		_, have := fonts.ByPostScript(faces, p.PostScript)
		switch {
		case p.Source != fonts.SourceInstalled && !have:
			label += tag.Render("  ↓ download")
		case !have:
			continue
		}
		opts = append(opts, huh.NewOption(label, p.PostScript))
		known = known || p.PostScript == c.HeadingFont
	}
	if !known {
		opts = append(opts, huh.NewOption(c.HeadingFont+tag.Render("  from file"), c.HeadingFont))
	}
	return opts
}

// floatOptions lists values for a typography setting; zero means the default.
func floatOptions(values []float64, current float64, zero string) []huh.Option[float64] {
	if current != 0 && !slices.Contains(values, current) {
		values = append(values, current)
		slices.Sort(values)
	}
	opts := []huh.Option[float64]{huh.NewOption(zero, 0.0)}
	for _, v := range values {
		opts = append(opts, huh.NewOption(strconv.FormatFloat(v, 'f', -1, 64), v))
	}
	return opts
}

// faceOptions lists a family's styles.
func faceOptions(faces []fonts.Face, family string) []huh.Option[string] {
	var opts []huh.Option[string]
	for _, f := range fonts.Styles(faces, family) {
		label := f.Style
		if f.BearOnly {
			label += lipgloss.NewStyle().Foreground(dim).Render("  inside Bear only")
		}
		opts = append(opts, huh.NewOption(label, f.PostScript))
	}
	return opts
}

func styled(f *huh.Form) *huh.Form {
	// Huh's "/" list filter keeps a stale scroll position (in v2.0.3), so a
	// search box narrows the long font list instead.
	km := huh.NewDefaultKeyMap()
	km.Select.Filter.SetEnabled(false)
	km.MultiSelect.Filter.SetEnabled(false)
	return f.WithTheme(huh.ThemeFunc(formTheme)).WithKeyMap(km).WithWidth(formWidth).WithShowHelp(true)
}

// newStartForm asks what to do: create, edit a saved theme, or restore.
func newStartForm(c *Choices, env Env) *huh.Form {
	modes := []huh.Option[string]{
		huh.NewOption("Create a new theme", ModeCreate),
		huh.NewOption("Edit a saved theme", ModeEdit),
	}
	if len(env.Backups) > 0 {
		modes = append(modes, huh.NewOption("Restore an original Bear theme", ModeRestore))
	}

	saved := make([]huh.Option[string], len(env.Saved))
	for i, s := range env.Saved {
		label := swatch(s.Paper) + swatch(s.Ink) + " " + s.Name
		if s.Builtin {
			label += lipgloss.NewStyle().Foreground(dim).Render("  built-in")
		}
		saved[i] = huh.NewOption(label, library.Slug(s.Name))
	}

	return styled(huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().Key("mode").Title("What would you like to do?").
				Options(modes...).Height(len(modes)+2).Value(&c.Mode),
		),
		huh.NewGroup(
			huh.NewSelect[string]().Key("saved").Title("Which saved theme?").
				Description("From ~/.config/bearty/themes. You can change any setting next.").
				Options(saved...).Height(min(len(saved), 8)+4).Value(&c.SavedSel),
		).WithHideFunc(func() bool { return c.Mode != ModeEdit }),

		huh.NewGroup(
			huh.NewSelect[string]().Key("restore").Title("Which theme should be restored?").
				Description("Only themes bearty backed up are listed.").
				Options(huh.NewOptions(env.Backups...)...).Height(min(len(env.Backups), 8)+3).Value(&c.RestoreSlot),
			huh.NewConfirm().Key("restore-confirm").
				TitleFunc(func() string { return fmt.Sprintf("Restore Bear's original %q?", c.RestoreSlot) }, &c.RestoreSlot).
				DescriptionFunc(func() string {
					return fmt.Sprintf("Copies ~/Downloads/%s.original.theme back into Bear. You'll be asked for your password.", c.RestoreSlot)
				}, &c.RestoreSlot).
				Affirmative("Restore").Negative("Cancel").Value(&c.RestoreConfirm),
		).WithHideFunc(func() bool { return !c.Restoring() }),
		huh.NewGroup(
			huh.NewConfirm().Key("remove-font").Title("Also remove the fonts bearty added to Bear?").
				Description(strings.Join(env.AddedFonts, ", ")+". Keep them if another theme still uses them.").
				Affirmative("Remove").Negative("Keep").Value(&c.RemoveFonts),
		).WithHideFunc(func() bool { return !c.Restoring() || !c.RestoreConfirm || len(env.AddedFonts) == 0 }),
		huh.NewGroup(
			huh.NewConfirm().Key("restore-typography").Title("Also restore your own Bear typography?").
				Description("Puts back the font, size and spacing you had before bearty changed them.").
				Affirmative("Restore").Negative("Keep").Value(&c.RestoreTypography),
		).WithHideFunc(func() bool { return !c.Restoring() || !c.RestoreConfirm || !env.TypographyBackup }),
	))
}

// newEditForm edits every setting of the theme in c.
func newEditForm(c *Choices, env Env) *huh.Form {
	c.faces = env.Faces
	picks := pickOptions(c, env.Faces)
	families := fonts.Families(env.Faces)
	sizes := make([]huh.Option[int], 0, len(theme.Sizes)+1)
	known := false
	for _, s := range theme.Sizes {
		sizes = append(sizes, huh.NewOption(fmt.Sprintf("%s · %dpt", s.Name, s.Pt), s.Pt))
		known = known || s.Pt == c.Size
	}
	if !known {
		sizes = append(sizes, huh.NewOption(fmt.Sprintf("%dpt", c.Size), c.Size))
	}
	headingFonts := headingFontOptions(c, env.Faces)

	outputs := []huh.Option[string]{}
	if len(env.Slots) > 0 {
		outputs = append(outputs,
			huh.NewOption("Install into Bear", OutputInstall),
			huh.NewOption("Install and export a .theme file", OutputBoth),
		)
	} else {
		c.Output = OutputExport
	}
	outputs = append(outputs,
		huh.NewOption("Export a .theme file only", OutputExport),
		huh.NewOption("Just save it to my themes", OutputNone),
	)
	outputDesc := "Every theme is also saved to your themes, so you can edit it later."
	if len(env.Slots) == 0 {
		outputDesc = "Bear.app wasn't found, so installing isn't available."
	}

	// Creating can't silently overwrite a saved theme; editing may keep its name.
	taken := map[string]bool{}
	for _, s := range env.Saved {
		taken[library.Slug(s.Name)] = true
	}
	editing := c.SavedSel

	return styled(huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Key("name").Title("Theme name").Value(&c.Name).Validate(func(s string) error {
				slug := library.Slug(s)
				switch {
				case strings.TrimSpace(s) == "":
					return errors.New("give it a name")
				case taken[slug] && slug != editing:
					return errors.New("you already have a theme with this name")
				}
				return nil
			}),
			huh.NewSelect[string]().Key("paper").Title("Paper").
				Options(swatchOptions(theme.Papers)...).Height(len(theme.Papers)+3).Value(&c.PaperSel),
		),
		huh.NewGroup(newPicker("paperhex", "Paper color", &c.PaperHex, func() string { return theme.Papers[0].Hex })).
			WithHideFunc(func() bool { return c.PaperSel != custom }),

		huh.NewGroup(
			huh.NewSelect[string]().Key("ink").Title("Ink").
				Description("Main text color. Links, list markers and ticks use it too, unless you pick them next.").
				Options(swatchOptions(theme.Inks)...).Height(len(theme.Inks)+4).Value(&c.InkSel),
		),
		huh.NewGroup(newPicker("inkhex", "Ink color", &c.InkHex, func() string { return theme.Inks[0].Hex })).
			WithHideFunc(func() bool { return c.InkSel != custom }),

		huh.NewGroup(
			huh.NewSelect[string]().Key("accentmode").Title("Links, list markers and to-do ticks").
				Options(
					huh.NewOption("Same as the text", RegionsAuto),
					huh.NewOption("Pick each color", RegionsCustom),
				).Height(4).Value(&c.AccentMode),
		),
		huh.NewGroup(
			colourInput(c, "acc-links", "Links", &c.Accents.Links,
				func(o theme.Options) string { return o.AccentsResolved().Links }),
			colourInput(c, "acc-markers", "List markers", &c.Accents.Markers,
				func(o theme.Options) string { return o.AccentsResolved().Markers }),
			colourInput(c, "acc-ticks", "To-do ticks", &c.Accents.Ticks,
				func(o theme.Options) string { return o.AccentsResolved().Ticks }),
		).WithHideFunc(func() bool { return c.AccentMode != RegionsCustom }),

		huh.NewGroup(
			huh.NewSelect[string]().Key("headings").Title("Headings").
				Options(
					huh.NewOption("   Same as the text", theme.HeadingsInk),
					huh.NewOption(swatch(theme.Rule)+" Black", theme.HeadingsBlack),
					huh.NewOption("   Custom color…", custom),
				).Height(5).Value(&c.Headings),
		),
		huh.NewGroup(newPicker("headinghex", "Heading color", &c.HeadingsHex, func() string { return theme.Rule })).
			WithHideFunc(func() bool { return c.Headings != custom }),

		huh.NewGroup(
			huh.NewSelect[string]().Key("fontsrc").Title("Typeface").
				Description("Used for note text. Bear always uses the system font in the sidebar and note list.").
				Options(
					huh.NewOption("Handwriting fonts", FontPicks),
					huh.NewOption(fmt.Sprintf("Any font on this Mac (%d families)", len(families)), FontInstalled),
					huh.NewOption("Any Google Font (downloaded on install)", FontGoogle),
				).Height(7).Value(&c.FontSrc),
		),
		huh.NewGroup(
			huh.NewSelect[string]().Key("pick").Title("Handwriting").
				Description("Fonts marked ↓ are downloaded when you install.").
				Options(picks...).Height(min(len(picks), 12)+3).Value(&c.FontPick),
		).WithHideFunc(func() bool { return c.FontSrc != FontPicks }),
		huh.NewGroup(
			huh.NewInput().Key("fontquery").Title("Search fonts").
				Placeholder("Type to filter, e.g. Avenir").Value(&c.FontQuery),
			huh.NewSelect[string]().Key("family").Title("Font family").
				DescriptionFunc(func() string {
					n := len(familyOptions(c, families))
					if c.FontQuery == "" {
						return fmt.Sprintf("%d families", n)
					}
					return fmt.Sprintf("%d matching %q", n, c.FontQuery)
				}, &c.FontQuery).
				OptionsFunc(func() []huh.Option[string] { return familyOptions(c, families) }, &c.FontQuery).
				Height(12).Value(&c.FontFamily).
				Validate(func(v string) error {
					if v == "" {
						return errors.New("no fonts match; change the search")
					}
					return nil
				}),
			huh.NewSelect[string]().Key("face").Title("Style").
				OptionsFunc(func() []huh.Option[string] { return faceOptions(env.Faces, c.FontFamily) }, &c.FontFamily).
				Height(7).Value(&c.FontFace),
		).WithHideFunc(func() bool { return c.FontSrc != FontInstalled }),
		huh.NewGroup(
			huh.NewInput().Key("google").Title("Google Fonts family").
				Description("Any family from fonts.google.com. Press ctrl+e to accept a suggestion.").
				Placeholder("Caveat").Suggestions(fonts.GoogleSuggestions).Value(&c.GoogleName).
				Validate(func(s string) error {
					if fonts.GoogleSlug(s) == "" {
						return errors.New("type a font family name")
					}
					return nil
				}),
		).WithHideFunc(func() bool { return c.FontSrc != FontGoogle }),
		huh.NewGroup(
			huh.NewSelect[string]().Key("headingfont").Title("Heading typeface").
				Options(headingFonts...).Height(min(len(headingFonts), 12)+2).Value(&c.HeadingFont),
		),

		huh.NewGroup(
			huh.NewSelect[int]().Key("size").Title("Size").Options(sizes...).Inline(true).Value(&c.Size),
			huh.NewSelect[float64]().Key("lineheight").Title("Line height").
				Options(floatOptions([]float64{1.2, 1.3, 1.4, 1.5, 1.8, 2}, c.Type.LineHeight, "1.6 (default)")...).
				Inline(true).Value(&c.Type.LineHeight),
			huh.NewSelect[float64]().Key("pspacing").Title("Paragraph spacing").
				Options(floatOptions([]float64{0.2, 0.4, 0.6, 0.8, 1}, c.Type.ParagraphSpacing, "Bear default")...).
				Inline(true).Value(&c.Type.ParagraphSpacing),
			huh.NewSelect[float64]().Key("pindent").Title("Paragraph indent").
				Options(floatOptions([]float64{0.2, 0.4, 0.6, 1}, c.Type.ParagraphIndent, "Bear default")...).
				Inline(true).Value(&c.Type.ParagraphIndent),
			huh.NewSelect[float64]().Key("linewidth").Title("Line width").
				Options(floatOptions([]float64{48, 56, 64, 72, 80}, c.Type.LineWidth, "Bear default")...).
				Inline(true).Value(&c.Type.LineWidth),
			colourInput(c, "lines", "Rule lines", &c.Lines,
				func(o theme.Options) string { return o.LineColor() }).(*picker).
				Description(func() string { return "Used for note-list separators and table borders." }),
		),

		huh.NewGroup(
			huh.NewSelect[string]().Key("regions").Title("Sidebar and note list colors").
				Description("Automatic colors are worked out from your paper and ink.").
				Options(
					huh.NewOption("Automatic", RegionsAuto),
					huh.NewOption("Pick each color", RegionsCustom),
				).Height(5).Value(&c.Regions),
		),
		huh.NewGroup(
			colourInput(c, "side-bg", "Sidebar background", &c.Sidebar.Background,
				func(o theme.Options) string { return o.SidebarResolved().Background }),
			colourInput(c, "side-text", "Sidebar text", &c.Sidebar.Text,
				func(o theme.Options) string { return o.SidebarResolved().Text }),
			colourInput(c, "side-icons", "Sidebar icons", &c.Sidebar.Icons,
				func(o theme.Options) string { return o.SidebarResolved().Icons }),
			colourInput(c, "side-sel", "Selected sidebar row", &c.Sidebar.Selected,
				func(o theme.Options) string { return o.SidebarResolved().Selected }),
		).Title("Sidebar").WithHideFunc(func() bool { return c.Regions != RegionsCustom }),
		huh.NewGroup(
			colourInput(c, "list-title", "Note titles", &c.NoteList.Title,
				func(o theme.Options) string { return o.NoteListResolved().Title }),
			colourInput(c, "list-preview", "Preview text", &c.NoteList.Preview,
				func(o theme.Options) string { return o.NoteListResolved().Preview }),
			colourInput(c, "list-date", "Dates", &c.NoteList.Date,
				func(o theme.Options) string { return o.NoteListResolved().Date }),
			colourInput(c, "list-sel", "Selected note", &c.NoteList.Selected,
				func(o theme.Options) string { return o.NoteListResolved().Selected }),
		).Title("Note list").WithHideFunc(func() bool { return c.Regions != RegionsCustom }),

		huh.NewGroup(
			huh.NewSelect[string]().Key("output").Title("What should bearty do with it?").Description(outputDesc).
				Options(outputs...).Height(len(outputs)+4).Value(&c.Output),
		),
		huh.NewGroup(
			huh.NewInput().Key("path").Title("Export file").Value(&c.SavePath).
				PlaceholderFunc(c.DefaultSavePath, &c.Name),
		).WithHideFunc(func() bool { return !c.Exporting() }),

		huh.NewGroup(
			huh.NewSelect[string]().Key("slot").Title("Which Bear theme should it replace?").
				Description("Bear can't add new themes, so bearty replaces one. Afterwards, choose this name in Bear → Settings → Themes.").
				Options(huh.NewOptions(env.Slots...)...).Height(14).Value(&c.Slot),
			huh.NewConfirm().Key("confirm").
				TitleFunc(func() string { return fmt.Sprintf("Replace %q?", c.Slot) }, &c.Slot).
				DescriptionFunc(func() string {
					return fmt.Sprintf("The original is backed up to ~/Downloads/%s.original.theme. You'll be asked for your password.", c.Slot)
				}, &c.Slot).
				Affirmative("Replace").Negative("Don't install").Value(&c.Confirm),
		).WithHideFunc(func() bool { return !c.Installing() }),
		huh.NewGroup(
			huh.NewConfirm().Key("typography").Title("Also set Bear's typography to match?").
				Description("Bear's own Typography settings override a theme's font and size. Your current settings are saved first, and Restore puts them back.").
				Affirmative("Yes, set it").Negative("No").Value(&c.SetTypography),
		).WithHideFunc(func() bool { return !c.Installing() || !c.Confirm }),
	))
}
