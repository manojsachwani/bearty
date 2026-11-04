package ui

import (
	"fmt"
	"math"
	"strconv"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/harmonica"
)

const fps = 60

// Springs: a lively one for things that move, a calm one for colours,
// which would look wrong if they overshot.
var (
	bouncy = harmonica.NewSpring(harmonica.FPS(fps), 7.0, 0.55)
	glide  = harmonica.NewSpring(harmonica.FPS(fps), 9.0, 0.85)
	calm   = harmonica.NewSpring(harmonica.FPS(fps), 10.0, 1.0)
)

type frameMsg time.Time

func frame() tea.Cmd {
	return tea.Tick(time.Second/fps, func(t time.Time) tea.Msg { return frameMsg(t) })
}

// value is one animated number.
type value struct {
	pos, vel, target float64
}

func (v *value) step(s harmonica.Spring) {
	v.pos, v.vel = s.Update(v.pos, v.vel, v.target)
}

func (v *value) settled() bool {
	return math.Abs(v.pos-v.target) < 0.01 && math.Abs(v.vel) < 0.01
}

// snap jumps straight to the target.
func (v *value) snap() { v.pos, v.vel = v.target, 0 }

// colour animates a #RRGGBB colour channel by channel.
type colour struct{ r, g, b value }

func newColour(hex string) colour {
	var c colour
	c.set(hex)
	c.snap()
	return c
}

func (c *colour) set(hex string) {
	r, g, b := rgb(hex)
	c.r.target, c.g.target, c.b.target = r, g, b
}

func (c *colour) step() {
	c.r.step(calm)
	c.g.step(calm)
	c.b.step(calm)
}

func (c *colour) snap() {
	c.r.snap()
	c.g.snap()
	c.b.snap()
}

func (c colour) settled() bool { return c.r.settled() && c.g.settled() && c.b.settled() }

func (c colour) hex() string {
	clamp := func(v float64) uint8 { return uint8(math.Max(0, math.Min(255, math.Round(v)))) }
	return fmt.Sprintf("#%02X%02X%02X", clamp(c.r.pos), clamp(c.g.pos), clamp(c.b.pos))
}

func rgb(hex string) (r, g, b float64) {
	v, err := strconv.ParseUint(hex[1:], 16, 32)
	if err != nil || len(hex) != 7 {
		return 0, 0, 0
	}
	return float64(v >> 16 & 0xff), float64(v >> 8 & 0xff), float64(v & 0xff)
}
