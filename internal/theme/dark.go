package theme

// Dark mode settings. Bear picks a separate theme when macOS is in dark
// mode, so a theme can come with a dark version.
const (
	DarkOff    = ""       // use the light theme in dark mode too
	DarkAuto   = "auto"   // derive a dark version from the paper and ink
	DarkCustom = "custom" // use the dark paper and ink below
)

// Dark describes a theme's dark version.
type Dark struct {
	Mode  string `json:"mode,omitempty"`
	Paper string `json:"paper,omitempty"`
	Ink   string `json:"ink,omitempty"`
}

// Enabled reports whether the theme has a dark version.
func (d Dark) Enabled() bool { return d.Mode == DarkAuto || d.Mode == DarkCustom }

// DarkPaperFor is a near-black page tinted very slightly by the light paper.
func DarkPaperFor(paper string) string { return Mix("#1D1E22", paper, 0.05) }

// DarkInkFor lifts an ink so it reads on dark paper, keeping its hue.
func DarkInkFor(ink string) string { return AtLeastLightness(ink, 0.66) }

// DarkVariant is the dark version of a light theme. Colours tuned for the
// light page (sidebar, note list, details and extra keys) are re-derived for
// the dark one; chosen accent and heading colours keep their hue but are
// lifted to stay readable.
func (o Options) DarkVariant(d Dark) Options {
	v := o
	v.Paper, v.Ink = DarkPaperFor(o.Paper), DarkInkFor(o.Ink)
	if d.Mode == DarkCustom {
		v.Paper = or(d.Paper, v.Paper)
		v.Ink = or(d.Ink, v.Ink)
	}
	lift := func(c string) string {
		if ValidateHex(c) != nil {
			return c
		}
		return AtLeastLightness(c, 0.62)
	}
	if ValidateHex(o.Headings) == nil {
		v.Headings = lift(o.Headings)
	}
	v.Accents = Accents{Links: lift(o.Accents.Links), Markers: lift(o.Accents.Markers), Ticks: lift(o.Accents.Ticks)}
	v.Lines, v.Sidebar, v.NoteList = "", SidebarColors{}, NoteListColors{}
	v.Highlights, v.Tag = Highlights{}, TagColors{}
	v.Code = CodeColors{Font: o.Code.Font}
	v.Extra = nil
	if o.Name != "" {
		v.Name = o.Name + " Dark"
	}
	return v
}
