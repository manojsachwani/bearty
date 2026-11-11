// Package ui is the interactive theme builder: a Huh form beside a live preview.
package ui

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"

	"github.com/manojsachwani/bearty/internal/capture"
	"github.com/manojsachwani/bearty/internal/fonts"
	"github.com/manojsachwani/bearty/internal/library"
	"github.com/manojsachwani/bearty/internal/theme"
)

const (
	custom = "custom"

	ModeCreate  = "create"
	ModeEdit    = "edit"
	ModeRestore = "restore"
	ModeReapply = "reapply"
	ModeCapture = "capture"
	ModeImport  = "import"
	ModeUse     = "use"

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

	TypographyBackup bool   // bearty saved the user's Bear settings and can put them back
	BearRunning      bool   // Bear is open right now
	ThemesDir        string // Bear's themes folder, for capturing
	Reapply          string // theme a Bear update removed, offered first when set
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

	FontSrc     string // FontPicks, FontInstalled or FontGoogle
	FontPick    string // PostScript name of a pick
	FontQuery   string // narrows the installed family list
	FontFamily  string // installed family
	FontFace    string // installed face, by PostScript name
	GoogleName  string // Google Fonts family
	GooglePS    string // unused; kept so older code paths compile
	GoogleStyle string // weight of a Google font, e.g. "700" or "300i"
	PickStyle   string // weight of a pick: a Google style, or a Mac face's PostScript name

	HeadingStyle string // weight of the heading pick, like PickStyle

	knownPS map[string]string // PostScript names already known, by download source

	HeadingFont   string // PostScript name for headings; empty = same as the text
	headingSource string // where a heading font from a saved theme downloads from

	Type theme.Typography

	Regions  string // RegionsAuto or RegionsCustom
	Sidebar  theme.SidebarColors
	NoteList theme.NoteListColors

	AccentMode string // RegionsAuto: links, markers and ticks follow the ink
	Accents    theme.Accents

	DetailsMode  string // RegionsAuto or RegionsCustom: highlighters, code and tags
	Highlights   theme.Highlights
	Code         theme.CodeColors // Code.Font is chosen on the Type step either way
	Tag          theme.TagColors
	HeadingScale float64
	Extra        map[string]any // kept from a captured theme; not edited here

	Dark     theme.Dark
	DarkSlot string
	Restart  bool // quit and reopen Bear around the install

	UseSel      string // slug of the installed theme to switch Bear to
	CaptureSlot string
	ImportPath  string
	NewName     string // name for a captured or imported theme

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
		DetailsMode:       RegionsAuto,
		Restart:           true,
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

	c.knownPS = map[string]string{}
	if s.FontSource != "" && s.Font != "" {
		c.knownPS[s.FontSource] = s.Font
	}
	if s.HeadingFontSource != "" && s.HeadingFont != "" {
		c.knownPS[s.HeadingFontSource] = s.HeadingFont
	}
	if p, style, ok := pickFor(s.Font, s.FontSource, faces); ok {
		c.FontSrc, c.FontPick, c.PickStyle = FontPicks, p.PostScript, style
	} else if fam, ok := fonts.GoogleFamily(s.FontSource); ok {
		c.FontSrc, c.GoogleName, c.GoogleStyle = FontGoogle, fam, fonts.GoogleStyleOf(s.FontSource)
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
	c.Highlights, c.Code, c.Tag, c.HeadingScale = s.Highlights, s.Code, s.Tag, s.HeadingScale
	c.Extra, c.Dark, c.DarkSlot = s.Extra, s.Dark, s.DarkSlot
	if !s.Highlights.IsZero() || !s.Tag.IsZero() || (s.Code != theme.CodeColors{Font: s.Code.Font}) {
		c.DetailsMode = RegionsCustom
	}
	c.HeadingFont, c.headingSource, c.Type = s.HeadingFont, s.HeadingFontSource, s.Type
	if p, style, ok := pickFor(s.HeadingFont, s.HeadingFontSource, faces); ok {
		c.HeadingFont, c.HeadingStyle = p.PostScript, style
	}
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
// A Google weight that hasn't been downloaded yet has no PostScript name;
// it's filled in when bearty fetches the file.
func (c *Choices) Font() theme.Font {
	f := c.textFont()
	if c.HeadingFont != "" {
		f.Heading = c.headingPS()
	}
	return f
}

// isStyle reports whether s is a Google style such as "700" or "300i".
func isStyle(s string) bool {
	s = strings.TrimSuffix(s, "i")
	_, err := strconv.Atoi(s)
	return err == nil
}

// pickFont resolves a pick plus its chosen weight to a PostScript name, a
// display name and a download source.
func (c *Choices) pickFont(p fonts.Pick, weight string) (ps, name, source string) {
	if fam, isGoogle := fonts.GoogleFamily(p.Source); isGoogle {
		style := fonts.RegularStyle
		if isStyle(weight) {
			style = weight
		}
		source = fonts.GoogleSourceStyle(fam, style)
		if style == fonts.RegularStyle {
			return p.PostScript, p.Name, source
		}
		return c.knownPS[source], p.Name + " " + fonts.StyleLabel(style), source
	}
	// A Mac or Bear font: the weight is one of the family's faces.
	if base, ok := fonts.ByPostScript(c.faces, p.PostScript); ok {
		if f, ok := fonts.ByPostScript(fonts.Styles(c.faces, base.Family), weight); ok && f.PostScript != p.PostScript {
			return f.PostScript, p.Name + " " + f.Style, p.Source
		}
	}
	return p.PostScript, p.Name, p.Source
}

// headingPS is the PostScript name of the chosen heading font and weight.
func (c *Choices) headingPS() string {
	if p, ok := fonts.PickByPostScript(c.HeadingFont); ok {
		ps, _, _ := c.pickFont(p, c.HeadingStyle)
		return ps
	}
	return c.HeadingFont
}

// HeadingFontSource says where the heading font downloads from, if anywhere.
func (c *Choices) HeadingFontSource() string {
	if c.HeadingFont == "" {
		return fonts.SourceInstalled
	}
	if p, ok := fonts.PickByPostScript(c.HeadingFont); ok {
		_, _, source := c.pickFont(p, c.HeadingStyle)
		return source
	}
	return c.headingSource
}

// HeadingFontName is the heading font's display name.
func (c *Choices) HeadingFontName() string {
	if p, ok := fonts.PickByPostScript(c.HeadingFont); ok {
		_, name, _ := c.pickFont(p, c.HeadingStyle)
		return name
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
		name := strings.TrimSpace(c.GoogleName)
		source := c.FontSource()
		ps := c.knownPS[source]
		if fonts.GoogleStyleOf(source) == fonts.RegularStyle {
			for _, p := range fonts.Picks {
				if fam, ok := fonts.GoogleFamily(p.Source); ok && strings.EqualFold(fam, name) {
					ps = p.PostScript
				}
			}
		}
		if s := fonts.GoogleStyleOf(source); s != fonts.RegularStyle {
			name += " " + fonts.StyleLabel(s)
		}
		return theme.Font{Name: name, PostScript: ps, Heading: ps}
	}
	if p, ok := fonts.PickByPostScript(c.FontPick); ok {
		ps, name, _ := c.pickFont(p, c.PickStyle)
		heading := p.HeadingFont()
		if ps != p.PostScript {
			heading = ps // a chosen weight applies to headings too
		}
		return theme.Font{Name: name, PostScript: ps, Heading: heading}
	}
	return theme.Font{Name: c.FontPick, PostScript: c.FontPick, Heading: c.FontPick}
}

// FontSource says where the font must be downloaded from, if anywhere.
func (c *Choices) FontSource() string {
	switch c.FontSrc {
	case FontGoogle:
		style := fonts.RegularStyle
		if isStyle(c.GoogleStyle) {
			style = c.GoogleStyle
		}
		return fonts.GoogleSourceStyle(strings.TrimSpace(c.GoogleName), style)
	case FontPicks:
		if p, ok := fonts.PickByPostScript(c.FontPick); ok {
			_, _, source := c.pickFont(p, c.PickStyle)
			return source
		}
	}
	return fonts.SourceInstalled
}

// Restoring reports whether the user chose to restore instead of create.
func (c *Choices) Restoring() bool { return c.Mode == ModeRestore }

// Reapplying reports whether the user chose to put back the last install.
func (c *Choices) Reapplying() bool { return c.Mode == ModeReapply }

// Using reports whether the user chose to switch Bear to an installed theme.
func (c *Choices) Using() bool { return c.Mode == ModeUse }

// details returns highlighter, code and tag overrides; the code font always
// counts, the colours only when chosen.
func (c *Choices) details() (theme.Highlights, theme.CodeColors, theme.TagColors) {
	if c.DetailsMode != RegionsCustom {
		return theme.Highlights{}, theme.CodeColors{Font: c.Code.Font}, theme.TagColors{}
	}
	return c.Highlights, c.Code, c.Tag
}

// codeSource says where the code font downloads from, if anywhere.
func (c *Choices) codeSource() string {
	if p, ok := fonts.CodePickByPostScript(c.Code.Font); ok {
		return p.Source
	}
	return fonts.SourceInstalled
}

// dark returns the dark settings, dropping colours a mode doesn't use.
func (c *Choices) dark() theme.Dark {
	switch c.Dark.Mode {
	case theme.DarkAuto:
		return theme.Dark{Mode: theme.DarkAuto}
	case theme.DarkCustom:
		return c.Dark
	}
	return theme.Dark{}
}

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
	hl, code, tag := c.details()
	return theme.Options{
		Name: c.Name, Paper: c.Paper(), Ink: c.Ink(), Font: c.Font(), Size: c.Size,
		Headings: c.headings(), Lines: c.Lines, Sidebar: side, NoteList: list,
		Accents: c.accents(), Type: c.Type,
		Highlights: hl, Code: code, Tag: tag, HeadingScale: c.HeadingScale, Extra: c.Extra,
	}
}

// Saved converts the choices for the theme library.
func (c *Choices) Saved() library.Saved {
	side, list := c.regions()
	hl, code, tag := c.details()
	return library.Saved{
		Name: strings.TrimSpace(c.Name), Paper: c.Paper(), Ink: c.Ink(),
		Font: c.Font().PostScript, FontSource: c.FontSource(), Size: c.Size,
		Headings: c.headings(), Lines: c.Lines, Slot: c.Slot,
		Sidebar: side, NoteList: list, Accents: c.accents(),
		HeadingFont: c.savedHeadingFont(), HeadingFontSource: c.HeadingFontSource(), Type: c.Type,
		Highlights: hl, Code: code, CodeSource: c.codeSource(), Tag: tag, HeadingScale: c.HeadingScale,
		Extra: c.Extra, Dark: c.dark(), DarkSlot: c.DarkSlot,
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

// pickFor finds the handwriting pick a saved font belongs to and the
// weight chosen for it.
func pickFor(ps, source string, faces []fonts.Face) (fonts.Pick, string, bool) {
	if p, ok := fonts.PickByPostScript(ps); ok {
		return p, "", true
	}
	if fam, ok := fonts.GoogleFamily(source); ok {
		for _, p := range fonts.Picks {
			if pf, ok := fonts.GoogleFamily(p.Source); ok && strings.EqualFold(pf, fam) {
				return p, fonts.GoogleStyleOf(source), true
			}
		}
		return fonts.Pick{}, "", false
	}
	if f, ok := fonts.ByPostScript(faces, ps); ok {
		for _, p := range fonts.Picks {
			if base, ok := fonts.ByPostScript(faces, p.PostScript); ok && base.Family == f.Family {
				return p, ps, true
			}
		}
	}
	return fonts.Pick{}, "", false
}

// weightOptions lists the weights of a pick: a Google family's styles (looked
// up online, in the background) or a Mac family's faces.
func weightOptions(c *Choices, pickPS string) []huh.Option[string] {
	p, ok := fonts.PickByPostScript(pickPS)
	if !ok {
		return []huh.Option[string]{huh.NewOption("Regular", "")}
	}
	if fam, isGoogle := fonts.GoogleFamily(p.Source); isGoogle {
		return googleWeightOptions(fam)
	}
	base, ok := fonts.ByPostScript(c.faces, p.PostScript)
	if !ok {
		return []huh.Option[string]{huh.NewOption("Regular", "")}
	}
	var opts []huh.Option[string]
	for _, f := range fonts.Styles(c.faces, base.Family) {
		o := huh.NewOption(f.Style, f.PostScript)
		if f.PostScript == p.PostScript {
			o = o.Selected(true)
		}
		opts = append(opts, o)
	}
	return opts
}

// googleWeightOptions lists a Google family's styles. Huh runs this in the
// background, so the network call doesn't block typing.
func googleWeightOptions(family string) []huh.Option[string] {
	regular := []huh.Option[string]{huh.NewOption("Regular", fonts.RegularStyle)}
	if fonts.GoogleSlug(family) == "" {
		return regular
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	styles, err := fonts.GoogleStyles(ctx, family)
	if err != nil {
		return regular
	}
	opts := make([]huh.Option[string], len(styles))
	for i, st := range styles {
		opts[i] = huh.NewOption(fonts.StyleLabel(st), st)
		if st == fonts.RegularStyle {
			opts[i] = opts[i].Selected(true)
		}
	}
	return opts
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
	var modes []huh.Option[string]
	if env.Reapply != "" {
		modes = append(modes, huh.NewOption(fmt.Sprintf("Re-apply %s after Bear's update", env.Reapply), ModeReapply))
	}
	var installed []huh.Option[string]
	for _, s := range env.Saved {
		if s.Slot != "" {
			label := swatch(s.Paper) + swatch(s.Ink) + " " + s.Name +
				lipgloss.NewStyle().Foreground(dim).Render("  in "+s.Slot)
			installed = append(installed, huh.NewOption(label, library.Slug(s.Name)))
		}
	}
	if len(installed) > 0 {
		modes = append(modes, huh.NewOption("Switch Bear to a saved theme", ModeUse))
		if c.UseSel == "" {
			c.UseSel = installed[0].Value
		}
	}
	modes = append(modes,
		huh.NewOption("Create a new theme", ModeCreate),
		huh.NewOption("Edit a saved theme", ModeEdit),
	)
	if len(env.Slots) > 0 {
		modes = append(modes, huh.NewOption("Capture one of Bear's themes", ModeCapture))
	}
	modes = append(modes, huh.NewOption("Import a .theme file", ModeImport))
	if len(env.Backups) > 0 {
		modes = append(modes, huh.NewOption("Restore an original Bear theme", ModeRestore))
	}

	taken := map[string]bool{}
	for _, s := range env.Saved {
		taken[library.Slug(s.Name)] = true
	}
	newName := func(fallback func() string) func(string) error {
		return func(v string) error {
			if v == "" {
				v = fallback()
			}
			if taken[library.Slug(v)] {
				return fmt.Errorf("you already have a theme called %q; type another name", v)
			}
			return nil
		}
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
			huh.NewSelect[string]().Key("use").Title("Which theme?").
				Description("bearty selects it in Bear and sets Bear's fonts and sizes to match. Bear can't do the fonts itself: it draws every theme with one set of Typography settings.").
				Options(installed...).Height(min(len(installed), 8)+5).Value(&c.UseSel),
			huh.NewConfirm().Key("use-restart").
				Title(restartTitle(env.BearRunning)).
				Description("Bear only loads theme changes when it starts.").
				Affirmative("Yes").Negative("No, I'll do it").Value(&c.Restart),
		).WithHideFunc(func() bool { return c.Mode != ModeUse }),

		huh.NewGroup(
			huh.NewConfirm().Key("reapply-restart").
				Title(restartTitle(env.BearRunning)).
				Description("bearty copies back its own copy of the theme and fonts. You'll be asked for your password.").
				Affirmative("Yes").Negative("No, I'll do it").Value(&c.Restart),
		).WithHideFunc(func() bool { return c.Mode != ModeReapply }),

		huh.NewGroup(
			huh.NewSelect[string]().Key("capture").Title("Which Bear theme?").
				Description("It becomes a bearty theme you can edit. Everything bearty can't edit is kept as it is.").
				Options(huh.NewOptions(env.Slots...)...).Height(14).Value(&c.CaptureSlot),
			huh.NewInput().Key("capture-name").Title("Name it").
				PlaceholderFunc(func() string { return c.CaptureSlot }, &c.CaptureSlot).
				Value(&c.NewName).Validate(newName(func() string { return c.CaptureSlot })),
		).WithHideFunc(func() bool { return c.Mode != ModeCapture }),

		huh.NewGroup(
			huh.NewInput().Key("import").Title("Theme file").
				Description("A Bear .theme file, e.g. one someone shared or bearty exported.").
				Placeholder("~/Downloads/my.theme").Value(&c.ImportPath).
				Validate(func(v string) error {
					_, err := capture.File(expandHome(v), env.ThemesDir, "check")
					return err
				}),
			huh.NewInput().Key("import-name").Title("Name it").
				PlaceholderFunc(func() string { return importName(c.ImportPath) }, &c.ImportPath).
				Value(&c.NewName).Validate(newName(func() string { return importName(c.ImportPath) })),
		).WithHideFunc(func() bool { return c.Mode != ModeImport }),

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
			huh.NewConfirm().Key("restore-typography").Title("Also restore your own Bear settings?").
				Description("Puts back the font, size, spacing and theme choice you had before bearty changed them.").
				Affirmative("Restore").Negative("Keep").Value(&c.RestoreTypography),
		).WithHideFunc(func() bool { return !c.Restoring() || !c.RestoreConfirm || !env.TypographyBackup }),
	))
}

// newEditForm edits every setting of the theme in c.
func newEditForm(c *Choices, env Env) *huh.Form {
	c.faces = env.Faces
	// Start the slot lists on Bear's plain light and dark themes, unless the
	// theme was installed somewhere before.
	if c.Slot == "" && slices.Contains(env.Slots, "Notes") {
		c.Slot = "Notes"
	}
	if c.DarkSlot == "" && slices.Contains(env.Slots, "Dark Notes") {
		c.DarkSlot = "Dark Notes"
	}
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
			huh.NewSelect[string]().Key("pickweight").Title("Weight").
				OptionsFunc(func() []huh.Option[string] { return weightOptions(c, c.FontPick) }, &c.FontPick).
				Height(8).Value(&c.PickStyle),
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
			huh.NewSelect[string]().Key("googleweight").Title("Weight").
				Description("Looked up on Google Fonts as you type.").
				OptionsFunc(func() []huh.Option[string] { return googleWeightOptions(c.GoogleName) }, &c.GoogleName).
				Height(8).Value(&c.GoogleStyle),
		).WithHideFunc(func() bool { return c.FontSrc != FontGoogle }),
		huh.NewGroup(
			huh.NewSelect[string]().Key("headingfont").Title("Heading typeface").
				Options(headingFonts...).Height(min(len(headingFonts), 12)+2).Value(&c.HeadingFont),
			huh.NewSelect[string]().Key("headingweight").Title("Heading weight").
				OptionsFunc(func() []huh.Option[string] {
					if c.HeadingFont == "" {
						return []huh.Option[string]{huh.NewOption("Same as the text", "")}
					}
					return weightOptions(c, c.HeadingFont)
				}, &c.HeadingFont).
				Height(8).Value(&c.HeadingStyle),
		),
		huh.NewGroup(
			huh.NewSelect[string]().Key("codefont").Title("Code font").
				Description("For code blocks and inline code.").
				Options(codeFontOptions(c, env.Faces)...).Height(12).Value(&c.Code.Font),
		),

		huh.NewGroup(
			huh.NewSelect[int]().Key("size").Title("Size").Options(sizes...).Inline(true).Value(&c.Size),
			huh.NewSelect[float64]().Key("headingscale").Title("Heading sizes").
				Description("How much bigger each heading level is.").
				Options(floatOptions([]float64{1.05, 1.1, 1.2, 1.25, 1.3, 1.4}, c.HeadingScale, "1.15 (default)")...).
				Inline(true).Value(&c.HeadingScale),
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
			huh.NewSelect[string]().Key("details").Title("Highlighters, code and tags").
				Description("Automatic colors are worked out from your paper and ink.").
				Options(
					huh.NewOption("Automatic", RegionsAuto),
					huh.NewOption("Pick each color", RegionsCustom),
				).Height(5).Value(&c.DetailsMode),
		),
		huh.NewGroup(
			colourInput(c, "hl-default", "Default highlighter", &c.Highlights.Default,
				func(o theme.Options) string { return o.HighlightsResolved().Default }),
			colourInput(c, "hl-red", "Red", &c.Highlights.Red,
				func(o theme.Options) string { return o.HighlightsResolved().Red }),
			colourInput(c, "hl-blue", "Blue", &c.Highlights.Blue,
				func(o theme.Options) string { return o.HighlightsResolved().Blue }),
			colourInput(c, "hl-green", "Green", &c.Highlights.Green,
				func(o theme.Options) string { return o.HighlightsResolved().Green }),
			colourInput(c, "hl-yellow", "Yellow", &c.Highlights.Yellow,
				func(o theme.Options) string { return o.HighlightsResolved().Yellow }),
			colourInput(c, "hl-purple", "Purple", &c.Highlights.Purple,
				func(o theme.Options) string { return o.HighlightsResolved().Purple }),
		).Title("Highlighters").WithHideFunc(func() bool { return c.DetailsMode != RegionsCustom }),
		huh.NewGroup(
			colourInput(c, "code-bg", "Code background", &c.Code.Background,
				func(o theme.Options) string { return o.CodeResolved().Background }),
			colourInput(c, "code-text", "Code text", &c.Code.Text,
				func(o theme.Options) string { return o.CodeResolved().Text }),
			colourInput(c, "code-keyword", "Keywords", &c.Code.Keyword,
				func(o theme.Options) string { return o.CodeResolved().Keyword }),
			colourInput(c, "code-string", "Strings", &c.Code.String,
				func(o theme.Options) string { return o.CodeResolved().String }),
			colourInput(c, "code-number", "Numbers", &c.Code.Number,
				func(o theme.Options) string { return o.CodeResolved().Number }),
			colourInput(c, "code-comment", "Comments", &c.Code.Comment,
				func(o theme.Options) string { return o.CodeResolved().Comment }),
			colourInput(c, "code-function", "Functions", &c.Code.Function,
				func(o theme.Options) string { return o.CodeResolved().Function }),
			colourInput(c, "code-variable", "Variables", &c.Code.Variable,
				func(o theme.Options) string { return o.CodeResolved().Variable }),
			colourInput(c, "code-constant", "Constants", &c.Code.Constant,
				func(o theme.Options) string { return o.CodeResolved().Constant }),
			colourInput(c, "code-entity", "Types and entities", &c.Code.Entity,
				func(o theme.Options) string { return o.CodeResolved().Entity }),
		).Title("Code blocks").WithHideFunc(func() bool { return c.DetailsMode != RegionsCustom }),
		huh.NewGroup(
			colourInput(c, "tag-bg", "Tag background", &c.Tag.Background,
				func(o theme.Options) string { return o.TagResolved().Background }),
			colourInput(c, "tag-text", "Tag text", &c.Tag.Text,
				func(o theme.Options) string { return o.TagResolved().Text }),
		).Title("Tags").WithHideFunc(func() bool { return c.DetailsMode != RegionsCustom }),

		huh.NewGroup(
			huh.NewSelect[string]().Key("dark").Title("Dark mode").
				Description("Bear uses a separate theme when your Mac is in dark mode.").
				Options(
					huh.NewOption("Use this theme in dark mode too", theme.DarkOff),
					huh.NewOption("Add an automatic dark version", theme.DarkAuto),
					huh.NewOption("Choose the dark paper and ink", theme.DarkCustom),
				).Height(6).Value(&c.Dark.Mode),
		),
		huh.NewGroup(
			newPicker("darkpaper", "Dark paper", &c.Dark.Paper, func() string { return theme.DarkPaperFor(c.Paper()) }),
			newPicker("darkink", "Dark ink", &c.Dark.Ink, func() string { return theme.DarkInkFor(c.Ink()) }),
		).WithHideFunc(func() bool { return c.Dark.Mode != theme.DarkCustom }),

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
				Description("Bear can't add new themes, so bearty replaces one and selects it for you.").
				Options(huh.NewOptions(env.Slots...)...).Height(14).Value(&c.Slot),
		).WithHideFunc(func() bool { return !c.Installing() }),
		huh.NewGroup(
			huh.NewSelect[string]().Key("darkslot").Title("Which Bear theme should the dark version replace?").
				Description("A second slot, used when your Mac is in dark mode.").
				Options(huh.NewOptions(env.Slots...)...).Height(14).Value(&c.DarkSlot).
				Validate(func(v string) error {
					if v == c.Slot {
						return errors.New("pick a different theme from the light one")
					}
					return nil
				}),
		).WithHideFunc(func() bool { return !c.Installing() || !c.dark().Enabled() }),
		huh.NewGroup(
			huh.NewConfirm().Key("confirm").
				TitleFunc(func() string {
					if c.dark().Enabled() {
						return fmt.Sprintf("Replace %q and %q?", c.Slot, c.DarkSlot)
					}
					return fmt.Sprintf("Replace %q?", c.Slot)
				}, c).
				Description("The originals are backed up to ~/Downloads. You'll be asked for your password.").
				Affirmative("Replace").Negative("Don't install").Value(&c.Confirm),
		).WithHideFunc(func() bool { return !c.Installing() }),
		huh.NewGroup(
			huh.NewConfirm().Key("typography").Title("Use this theme's fonts in Bear?").
				Description("Bear ignores fonts inside themes and uses one set of Typography settings for every theme, so bearty sets them to this theme's fonts and sizes. Your own settings are saved first, and Restore puts them back. To switch themes later with their fonts, use Switch Bear to a saved theme.").
				Affirmative("Yes, set it").Negative("No").Value(&c.SetTypography),
		).WithHideFunc(func() bool { return !c.Installing() || !c.Confirm }),
		huh.NewGroup(
			huh.NewConfirm().Key("restart").
				Title(restartTitle(env.BearRunning)).
				Description("Bear only loads themes when it starts.").
				Affirmative("Yes").Negative("No, I'll do it").Value(&c.Restart),
		).WithHideFunc(func() bool { return !c.Installing() || !c.Confirm }),
	))
}

// savedHeadingFont is the heading font's PostScript name for saving: empty
// when headings use the text font.
func (c *Choices) savedHeadingFont() string {
	if c.HeadingFont == "" {
		return ""
	}
	return c.headingPS()
}

func restartTitle(running bool) string {
	if running {
		return "Quit and reopen Bear to apply it now?"
	}
	return "Open Bear when it's done?"
}

// codeFontOptions lists the code font picks, marking ones that download and
// leaving out Mac fonts this Mac doesn't have.
func codeFontOptions(c *Choices, faces []fonts.Face) []huh.Option[string] {
	tag := lipgloss.NewStyle().Foreground(dim)
	var opts []huh.Option[string]
	known := false
	for _, p := range fonts.CodePicks {
		label := p.Name
		_, have := fonts.ByPostScript(faces, p.PostScript)
		switch {
		case p.Source != fonts.SourceInstalled && !have:
			label += tag.Render("  ↓ download")
		case !have && p.PostScript != theme.DefaultCodeFont:
			continue
		}
		value := p.PostScript
		if value == theme.DefaultCodeFont {
			value = "" // Bear's default
		}
		opts = append(opts, huh.NewOption(label, value))
		known = known || value == c.Code.Font
	}
	if !known && c.Code.Font != "" {
		opts = append(opts, huh.NewOption(c.Code.Font+tag.Render("  from file"), c.Code.Font))
	}
	return opts
}

// expandHome turns a leading ~ into the home folder.
func expandHome(p string) string {
	p = strings.TrimSpace(p)
	if rest, ok := strings.CutPrefix(p, "~/"); ok {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, rest)
		}
	}
	return p
}

// importName suggests a name from a theme file's name.
func importName(path string) string {
	return strings.TrimSuffix(filepath.Base(strings.TrimSpace(path)), ".theme")
}

// Captured turns the capture or import choices into a saved theme.
func (c *Choices) Captured(themesDir string) (library.Saved, error) {
	if c.Mode == ModeCapture {
		name := c.NewName
		if name == "" {
			name = c.CaptureSlot
		}
		return capture.File(filepath.Join(themesDir, c.CaptureSlot+".theme"), themesDir, name)
	}
	name := c.NewName
	if name == "" {
		name = importName(c.ImportPath)
	}
	return capture.File(expandHome(c.ImportPath), themesDir, name)
}
