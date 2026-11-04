package theme

import (
	"fmt"
	"regexp"
	"strconv"
)

var hexRe = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

// ValidateHex reports whether s is a #RRGGBB colour.
func ValidateHex(s string) error {
	if !hexRe.MatchString(s) {
		return fmt.Errorf("use #RRGGBB, e.g. #D35A27")
	}
	return nil
}

type rgb struct{ r, g, b float64 }

func parse(hex string) rgb {
	v, _ := strconv.ParseUint(hex[1:], 16, 32)
	return rgb{float64(v >> 16 & 0xff), float64(v >> 8 & 0xff), float64(v & 0xff)}
}

func (c rgb) hex() string {
	return fmt.Sprintf("#%02X%02X%02X", uint8(c.r+0.5), uint8(c.g+0.5), uint8(c.b+0.5))
}

// Mix blends a towards b by t (0 = a, 1 = b).
func Mix(a, b string, t float64) string {
	x, y := parse(a), parse(b)
	return rgb{
		x.r + (y.r-x.r)*t,
		x.g + (y.g-x.g)*t,
		x.b + (y.b-x.b)*t,
	}.hex()
}

// Shade darkens c by t.
func Shade(c string, t float64) string { return Mix(c, "#000000", t) }
