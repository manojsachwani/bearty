package theme

// Swatch is a named colour preset.
type Swatch struct {
	Name string
	Hex  string
}

// Papers are the page colours, sampled from the reference screenshots.
var Papers = []Swatch{
	{"White", "#F9F9F9"},
	{"Cream", "#EFDEC4"},
	{"Blush", "#E4C3C3"},
}

// Inks are the text and accent colours, sampled from the reference screenshots.
var Inks = []Swatch{
	{"Pumpkin", "#CF5624"},
	{"Navy", "#3D4F74"},
	{"Plum", "#69395F"},
	{"Clay", "#AB744C"},
	{"Teal", "#4D7374"},
	{"Charcoal", "#2F2F2F"},
}

// Heading colours: the note's ink, or the near-black of the reference
// settings screen.
const (
	HeadingsInk   = "ink"
	HeadingsBlack = "black"
)

// Font is the typeface written into a theme, by PostScript name.
type Font struct {
	Name       string
	PostScript string
	Heading    string // PostScript name used for headers
}

// Sizes maps the size names to Bear's "text size".
var Sizes = []struct {
	Name string
	Pt   int
}{
	{"Small", 15},
	{"Medium", 17},
	{"Large", 19},
	{"Extra large", 22},
	{"Huge", 24},
}
