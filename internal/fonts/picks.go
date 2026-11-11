package fonts

import "strings"

// Font sources. A theme remembers where its font came from so it can be
// fetched again on another Mac or after a Bear update.
const (
	SourceInstalled = ""       // already on the Mac or inside Bear
	SourceInkina    = "inkina" // shapedfonts.com download
	googlePrefix    = "google:"
)

// GoogleSource is the source string for a Google Fonts family.
func GoogleSource(family string) string { return googlePrefix + family }

// GoogleFamily returns the family of a Google source, without its style.
func GoogleFamily(source string) (string, bool) {
	rest, ok := strings.CutPrefix(source, googlePrefix)
	family, _, _ := strings.Cut(rest, "@")
	return family, ok
}

// GoogleStyleOf returns the style of a Google source, such as "700" or
// "400i"; sources saved before styles existed mean Regular.
func GoogleStyleOf(source string) string {
	if _, style, ok := strings.Cut(source, "@"); ok && style != "" {
		return style
	}
	return RegularStyle
}

// GoogleSourceStyle is the source for one style of a Google family.
func GoogleSourceStyle(family, style string) string {
	if style == "" || style == RegularStyle {
		return GoogleSource(family)
	}
	return GoogleSource(family) + "@" + style
}

// InkinaPostScript is the PostScript name inside Inkina-Regular.otf.
const InkinaPostScript = "Inkina-Regular"

// Pick is a recommended handwriting font.
type Pick struct {
	Name       string
	PostScript string
	Heading    string // PostScript name for headers; same as PostScript when empty
	Source     string
}

// HeadingFont is the face used for headers.
func (p Pick) HeadingFont() string {
	if p.Heading != "" {
		return p.Heading
	}
	return p.PostScript
}

// Picks are the handwriting fonts offered first. PostScript names for the
// Google ones come from each family's METADATA.pb.
var Picks = []Pick{
	{Name: "Inkina", PostScript: InkinaPostScript, Source: SourceInkina},
	{Name: "Caveat", PostScript: "Caveat-Regular", Source: GoogleSource("Caveat")},
	{Name: "Patrick Hand", PostScript: "PatrickHand-Regular", Source: GoogleSource("Patrick Hand")},
	{Name: "Kalam", PostScript: "Kalam-Regular", Source: GoogleSource("Kalam")},
	{Name: "Indie Flower", PostScript: "IndieFlower-Regular", Source: GoogleSource("Indie Flower")},
	{Name: "Gochi Hand", PostScript: "GochiHand-Regular", Source: GoogleSource("Gochi Hand")},
	{Name: "Architects Daughter", PostScript: "ArchitectsDaughter-Regular", Source: GoogleSource("Architects Daughter")},
	{Name: "Shadows Into Light", PostScript: "ShadowsIntoLight", Source: GoogleSource("Shadows Into Light")},
	{Name: "Homemade Apple", PostScript: "HomemadeApple-Regular", Source: GoogleSource("Homemade Apple")},
	{Name: "Handlee", PostScript: "Handlee-Regular", Source: GoogleSource("Handlee")},
	{Name: "Noteworthy", PostScript: "Noteworthy-Light", Heading: "Noteworthy-Bold"},
	{Name: "Bradley Hand", PostScript: "BradleyHandITCTT-Bold"},
	{Name: "Marker Felt", PostScript: "MarkerFelt-Thin", Heading: "MarkerFelt-Wide"},
	{Name: "Bear Sans", PostScript: "BearSansUI-Regular", Heading: "BearSansUIHeading-Regular"},
}

// GoogleSuggestions are offered while typing a Google Fonts name.
var GoogleSuggestions = []string{
	"Caveat", "Patrick Hand", "Kalam", "Indie Flower", "Gochi Hand", "Architects Daughter",
	"Shadows Into Light", "Homemade Apple", "Handlee", "Reenie Beanie", "Nanum Pen Script",
	"Gaegu", "Coming Soon", "Schoolbell", "Delius", "Just Another Hand", "Sue Ellen Francisco",
	"Covered By Your Grace", "Nothing You Could Do", "Rock Salt", "Permanent Marker",
	"Amatic SC", "Dancing Script", "Satisfy", "Sacramento", "Great Vibes", "Merriweather",
	"Lora", "EB Garamond", "Crimson Pro", "Literata", "Inter", "IBM Plex Sans", "Nunito",
}

// PickByPostScript finds a recommended font.
func PickByPostScript(ps string) (Pick, bool) {
	for _, p := range Picks {
		if p.PostScript == ps {
			return p, true
		}
	}
	return Pick{}, false
}

// CodePicks are the monospace fonts offered for code blocks. Google fonts
// whose METADATA names only a light default instance are left out, since
// Bear would draw code too thin.
var CodePicks = []Pick{
	{Name: "Roboto Mono (Bear's default)", PostScript: "RobotoMono-Regular"},
	{Name: "Menlo", PostScript: "Menlo-Regular"},
	{Name: "Monaco", PostScript: "Monaco"},
	{Name: "Courier New", PostScript: "CourierNewPSMT"},
	{Name: "Andale Mono", PostScript: "AndaleMono"},
	{Name: "PT Mono", PostScript: "PTMono-Regular"},
	{Name: "JetBrains Mono", PostScript: "JetBrainsMono-Regular", Source: GoogleSource("JetBrains Mono")},
	{Name: "IBM Plex Mono", PostScript: "IBMPlexMono-Regular", Source: GoogleSource("IBM Plex Mono")},
	{Name: "Space Mono", PostScript: "SpaceMono-Regular", Source: GoogleSource("Space Mono")},
	{Name: "Inconsolata", PostScript: "Inconsolata-Regular", Source: GoogleSource("Inconsolata")},
}

// CodePickByPostScript finds a code font pick.
func CodePickByPostScript(ps string) (Pick, bool) {
	for _, p := range CodePicks {
		if p.PostScript == ps {
			return p, true
		}
	}
	return Pick{}, false
}
