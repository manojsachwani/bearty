package theme

// SidebarColors override the sidebar. Empty fields are derived.
type SidebarColors struct {
	Background string `json:"background,omitempty"`
	Text       string `json:"text,omitempty"`
	Icons      string `json:"icons,omitempty"`
	Selected   string `json:"selected,omitempty"` // selected row background
}

// NoteListColors override the note list. Bear gives the list no background
// of its own; it uses the paper colour.
type NoteListColors struct {
	Title    string `json:"title,omitempty"`
	Preview  string `json:"preview,omitempty"`
	Date     string `json:"date,omitempty"`
	Selected string `json:"selected,omitempty"` // selected note background
}

// IsZero reports whether nothing is overridden.
func (s SidebarColors) IsZero() bool { return s == SidebarColors{} }

// IsZero reports whether nothing is overridden.
func (n NoteListColors) IsZero() bool { return n == NoteListColors{} }

func or(v, def string) string {
	if ValidateHex(v) == nil {
		return v
	}
	return def
}

// SidebarResolved fills in every sidebar colour.
func (o Options) SidebarResolved() SidebarColors {
	bg := or(o.Sidebar.Background, o.depth(o.Paper, 0.06))
	return SidebarColors{
		Background: bg,
		Text:       or(o.Sidebar.Text, o.sidebarText()),
		Icons:      or(o.Sidebar.Icons, o.Ink),
		// A step darker than whatever the sidebar is, so a custom background keeps a visible selection.
		Selected: or(o.Sidebar.Selected, o.depth(bg, 0.085)),
	}
}

// NoteListResolved fills in every note list colour.
func (o Options) NoteListResolved() NoteListColors {
	secondary := Mix(o.contrastText(), o.Paper, 0.45)
	return NoteListColors{
		Title:    or(o.NoteList.Title, o.HeadingColor()),
		Preview:  or(o.NoteList.Preview, secondary),
		Date:     or(o.NoteList.Date, secondary),
		Selected: or(o.NoteList.Selected, o.depth(o.Paper, 0.07)),
	}
}

// Accents override the text-coloured details in notes. Empty fields follow
// the ink.
type Accents struct {
	Links   string `json:"links,omitempty"`
	Markers string `json:"markers,omitempty"` // list bullets and numbers
	Ticks   string `json:"ticks,omitempty"`   // to-do checkmarks
}

// IsZero reports whether nothing is overridden.
func (a Accents) IsZero() bool { return a == Accents{} }

// AccentsResolved fills in every accent colour.
func (o Options) AccentsResolved() Accents {
	return Accents{
		Links:   or(o.Accents.Links, o.Ink),
		Markers: or(o.Accents.Markers, o.Ink),
		Ticks:   or(o.Accents.Ticks, o.Ink),
	}
}

// Typography tunes the editor's layout. Zero fields use bearty's defaults
// (line height 1.6) or Bear's own (the rest).
type Typography struct {
	LineHeight       float64 `json:"line_height,omitempty"`
	LineWidth        float64 `json:"line_width,omitempty"`        // Bear's line width setting, e.g. 64
	ParagraphSpacing float64 `json:"paragraph_spacing,omitempty"` // space between paragraphs
	ParagraphIndent  float64 `json:"paragraph_indent,omitempty"`  // first-line indent
}

// IsZero reports whether nothing is set.
func (t Typography) IsZero() bool { return t == Typography{} }

// DefaultLineHeight is used when Typography.LineHeight is zero.
const DefaultLineHeight = 1.6

func (o Options) sidebarText() string {
	if IsDark(o.Paper) {
		return "#D6D6D6"
	}
	return "#333333"
}
