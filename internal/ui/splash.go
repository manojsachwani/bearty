package ui

import (
	"math"
	"strings"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/manojsachwani/bearty/internal/theme"
)

// bearArt is the left half of the logo; each row is mirrored to make it
// symmetric. Two rows share one terminal line through half blocks.
//
//	o fur · d inner ear · p cheek · c muzzle · k eyes and nose · w eye light · . empty
var bearArt = []string{
	"..ooo.....",
	".oddoo....",
	".oddoooooo",
	"..oooooooo",
	".ooooooooo",
	"oooooooooo",
	"ooookwoooo",
	"ooookkoooo",
	"oppooooccc",
	"oppooocckk",
	".oooooccck",
	"..oooocccc",
	"...ooooooo",
	".....ooooo",
}

var bearPalette = map[byte]string{
	'o': theme.Inks[0].Hex, // pumpkin
	'd': "#9E3C18",
	'c': "#EFDEC4", // cream paper
	'k': "#2F2F2F",
	'w': "#F9F9F9",
	'p': "#E07B52",
}

func mirror(s string) string {
	b := []byte(s)
	for i := len(s) - 1; i >= 0; i-- {
		b = append(b, s[i])
	}
	return string(b)
}

// renderBear draws the logo with half blocks: the top pixel is the
// foreground of ▀ and the bottom pixel its background.
func renderBear() string {
	var lines []string
	for y := 0; y+1 < len(bearArt); y += 2 {
		top, bottom := mirror(bearArt[y]), mirror(bearArt[y+1])
		var b strings.Builder
		for x := 0; x < len(top); x++ {
			t, bt := bearPalette[top[x]], bearPalette[bottom[x]]
			switch {
			case t == "" && bt == "":
				b.WriteString(" ")
			case bt == "":
				b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(t)).Render("▀"))
			case t == "":
				b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(bt)).Render("▄"))
			default:
				b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(t)).Background(lipgloss.Color(bt)).Render("▀"))
			}
		}
		lines = append(lines, b.String())
	}
	return strings.Join(lines, "\n")
}

const (
	splashDrop = 6.0                    // lines the bear falls from
	splashHold = 900 * time.Millisecond // pause once everything has landed
	swatchLag  = 4                      // frames between swatches popping in
)

// splash animates the start screen: the bear drops in with a bounce, the
// name types itself out, then the ink swatches pop in one by one.
type splash struct {
	drop     value // lines above the resting place
	reveal   value // 0…1 share of the name shown
	swatches []value
	frames   int
	landed   time.Time
	bear     string
}

func newSplash() *splash {
	s := &splash{bear: renderBear()}
	s.drop = value{pos: splashDrop, target: 0}
	s.reveal = value{pos: 0, target: 1}
	s.swatches = make([]value, len(theme.Inks))
	return s
}

// step advances one frame and reports whether the splash is finished.
func (s *splash) step(now time.Time) bool {
	s.frames++
	s.drop.step(bouncy)
	if s.drop.pos < 2 { // start typing as the bear lands
		s.reveal.step(glide)
	}
	for i := range s.swatches {
		if s.frames > 18+i*swatchLag {
			s.swatches[i].target = 1
		}
		s.swatches[i].step(bouncy)
	}
	if s.settled() {
		if s.landed.IsZero() {
			s.landed = now
		}
		return now.Sub(s.landed) >= splashHold
	}
	return false
}

func (s *splash) settled() bool {
	if !s.drop.settled() || !s.reveal.settled() {
		return false
	}
	for _, v := range s.swatches {
		if v.target != 1 || !v.settled() {
			return false
		}
	}
	return true
}

func (s *splash) view(version string, width, height int) string {
	const name = "bearty"
	n := int(math.Round(math.Max(0, math.Min(1, s.reveal.pos)) * float64(len(name))))
	word := lipgloss.NewStyle().Foreground(ember).Bold(true).Render(name[:n]) +
		lipgloss.NewStyle().Foreground(faintFg).Render(strings.Repeat("·", len(name)-n))

	tagline := lipgloss.NewStyle().Foreground(dim).Render("custom themes for Bear")
	if s.reveal.pos < 0.6 {
		tagline = strings.Repeat(" ", lipgloss.Width(tagline))
	}

	var chips []string
	for i, ink := range theme.Inks {
		w := int(math.Round(math.Max(0, s.swatches[i].pos) * 3))
		chip := strings.Repeat("█", min(w, 4)) + strings.Repeat(" ", max(0, 3-w))
		chips = append(chips, lipgloss.NewStyle().Foreground(lipgloss.Color(ink.Hex)).Render(chip))
	}

	// The bear sits on the box's floor; the drop lifts it, and the bounce's
	// overshoot pushes it one line below the floor's padding.
	drop := int(math.Round(s.drop.pos))
	bear := s.bear + "\n" // one line of floor for the overshoot
	if drop < 0 {
		bear = s.bear
	}
	bear += strings.Repeat("\n", max(drop, 0))
	bearBox := lipgloss.NewStyle().Height(int(splashDrop) + len(bearArt)/2 + 1).AlignVertical(lipgloss.Bottom).Render(bear)

	body := lipgloss.JoinVertical(lipgloss.Center,
		bearBox, "", word, tagline, "", strings.Join(chips, " "), "",
		lipgloss.NewStyle().Foreground(faintFg).Render(version),
	)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, body)
}
