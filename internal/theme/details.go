package theme

// Highlights override the six highlighter backgrounds. Text on each is
// picked automatically for contrast.
type Highlights struct {
	Default string `json:"default,omitempty"`
	Red     string `json:"red,omitempty"`
	Blue    string `json:"blue,omitempty"`
	Green   string `json:"green,omitempty"`
	Yellow  string `json:"yellow,omitempty"`
	Purple  string `json:"purple,omitempty"`
}

// IsZero reports whether nothing is overridden.
func (h Highlights) IsZero() bool { return h == Highlights{} }

// CodeColors override code blocks: their font, background, text and the
// syntax highlighting palette.
type CodeColors struct {
	Font       string `json:"font,omitempty"` // PostScript name
	Background string `json:"background,omitempty"`
	Text       string `json:"text,omitempty"`
	Comment    string `json:"comment,omitempty"`
	Constant   string `json:"constant,omitempty"`
	Number     string `json:"number,omitempty"`
	String     string `json:"string,omitempty"`
	Entity     string `json:"entity,omitempty"`
	Keyword    string `json:"keyword,omitempty"`
	Function   string `json:"function,omitempty"`
	Variable   string `json:"variable,omitempty"`
}

// IsZero reports whether nothing is overridden.
func (c CodeColors) IsZero() bool { return c == CodeColors{} }

// TagColors override #tag pills.
type TagColors struct {
	Background string `json:"background,omitempty"`
	Text       string `json:"text,omitempty"`
}

// IsZero reports whether nothing is overridden.
func (t TagColors) IsZero() bool { return t == TagColors{} }

// DefaultCodeFont is the monospace font Bear bundles.
const DefaultCodeFont = "RobotoMono-Regular"

// DefaultHeadingScale is how much bigger each heading level is than the next.
const DefaultHeadingScale = 1.15

// contrastText is near-black on light paper and near-white on dark paper.
func (o Options) contrastText() string {
	if IsDark(o.Paper) {
		return "#ECECEC"
	}
	return Rule
}

// depth moves a colour one step "into" the page: darker on light paper,
// lighter on dark paper. Used for secondary backgrounds and selections.
func (o Options) depth(c string, t float64) string {
	if IsDark(o.Paper) {
		return Mix(c, "#FFFFFF", t)
	}
	return Shade(c, t)
}

// readableOn picks black or white text for a background.
func readableOn(bg string) string {
	if Contrast(bg, Rule) >= Contrast(bg, "#F4F4F4") {
		return Rule
	}
	return "#F4F4F4"
}

// highlightBase are the hues Bear's six highlighters are named after.
var highlightBase = [6]string{"", "#D9534F", "#3B7DD8", "#4E9A52", "#E5B92E", "#8E5FB5"}

// HighlightsResolved fills in every highlighter background.
func (o Options) HighlightsResolved() Highlights {
	tint := func(c string) string { return Mix(c, o.Paper, 0.7) }
	return Highlights{
		Default: or(o.Highlights.Default, Mix(o.Ink, o.Paper, 0.75)),
		Red:     or(o.Highlights.Red, tint(highlightBase[1])),
		Blue:    or(o.Highlights.Blue, tint(highlightBase[2])),
		Green:   or(o.Highlights.Green, tint(highlightBase[3])),
		Yellow:  or(o.Highlights.Yellow, tint(highlightBase[4])),
		Purple:  or(o.Highlights.Purple, tint(highlightBase[5])),
	}
}

// syntax palettes tuned for light and dark paper.
var (
	lightSyntax = CodeColors{Comment: "#8A8A8A", Constant: "#2F6F9F", Number: "#B5562A", String: "#5E7D3A",
		Entity: "#7A4E8C", Keyword: "#3D4E7A", Function: "#4D7573", Variable: "#8C5A2E"}
	darkSyntax = CodeColors{Comment: "#8C8C93", Constant: "#79B8FF", Number: "#F0A070", String: "#A5D6A7",
		Entity: "#C9A0DC", Keyword: "#8FA8E0", Function: "#7FC8C4", Variable: "#E0B080"}
)

// CodeResolved fills in every code block setting.
func (o Options) CodeResolved() CodeColors {
	def := lightSyntax
	if IsDark(o.Paper) {
		def = darkSyntax
	}
	c := o.Code
	fontName := c.Font
	if fontName == "" {
		fontName = DefaultCodeFont
	}
	return CodeColors{
		Font:       fontName,
		Background: or(c.Background, o.depth(o.Paper, 0.04)),
		Text:       or(c.Text, o.contrastText()),
		Comment:    or(c.Comment, def.Comment),
		Constant:   or(c.Constant, def.Constant),
		Number:     or(c.Number, def.Number),
		String:     or(c.String, def.String),
		Entity:     or(c.Entity, def.Entity),
		Keyword:    or(c.Keyword, def.Keyword),
		Function:   or(c.Function, def.Function),
		Variable:   or(c.Variable, def.Variable),
	}
}

// TagResolved fills in the tag pill colours.
func (o Options) TagResolved() TagColors {
	return TagColors{
		Background: or(o.Tag.Background, Mix(o.Ink, o.Paper, 0.88)),
		Text:       or(o.Tag.Text, o.Ink),
	}
}

// headingScale is the modular scale for headings.
func (o Options) headingScale() float64 {
	if o.HeadingScale > 0 {
		return o.HeadingScale
	}
	return DefaultHeadingScale
}
