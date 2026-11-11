package theme

import (
	"fmt"
	"math"
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

// parse reads #RRGGBB; anything else reads as black rather than failing,
// so a hand-edited theme with a bad colour can't crash bearty.
func parse(hex string) rgb {
	if ValidateHex(hex) != nil {
		return rgb{}
	}
	v, _ := strconv.ParseUint(hex[1:], 16, 32)
	return rgb{float64(v >> 16 & 0xff), float64(v >> 8 & 0xff), float64(v & 0xff)}
}

func (c rgb) hex() string {
	ch := func(v float64) uint8 { return uint8(math.Max(0, math.Min(255, math.Round(v)))) }
	return fmt.Sprintf("#%02X%02X%02X", ch(c.r), ch(c.g), ch(c.b))
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

// Luminance is the WCAG relative luminance of #RRGGBB (0 black … 1 white).
func Luminance(hex string) float64 {
	c := parse(hex)
	lin := func(v float64) float64 {
		v /= 255
		if v <= 0.03928 {
			return v / 12.92
		}
		return math.Pow((v+0.055)/1.055, 2.4)
	}
	return 0.2126*lin(c.r) + 0.7152*lin(c.g) + 0.0722*lin(c.b)
}

// Contrast is the WCAG contrast ratio between two colours (1 … 21).
func Contrast(a, b string) float64 {
	la, lb := Luminance(a), Luminance(b)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

// IsDark reports whether text on this background should be light.
func IsDark(hex string) bool { return Luminance(hex) < 0.18 }

// ToHSL converts #RRGGBB to hue (0–360), saturation and lightness (0–1).
func ToHSL(hex string) (h, s, l float64) {
	c := parse(hex)
	r, g, b := c.r/255, c.g/255, c.b/255
	mx, mn := math.Max(r, math.Max(g, b)), math.Min(r, math.Min(g, b))
	l = (mx + mn) / 2
	if mx == mn {
		return 0, 0, l
	}
	d := mx - mn
	if l > 0.5 {
		s = d / (2 - mx - mn)
	} else {
		s = d / (mx + mn)
	}
	switch mx {
	case r:
		h = (g - b) / d
		if g < b {
			h += 6
		}
	case g:
		h = (b-r)/d + 2
	default:
		h = (r-g)/d + 4
	}
	return h * 60, s, l
}

// FromHSL converts hue, saturation and lightness back to #RRGGBB.
func FromHSL(h, s, l float64) string {
	if s == 0 {
		v := l * 255
		return rgb{v, v, v}.hex()
	}
	q := l * (1 + s)
	if l >= 0.5 {
		q = l + s - l*s
	}
	p := 2*l - q
	hk := h / 360
	ch := func(t float64) float64 {
		t = math.Mod(t+1, 1)
		switch {
		case t < 1.0/6:
			return p + (q-p)*6*t
		case t < 0.5:
			return q
		case t < 2.0/3:
			return p + (q-p)*(2.0/3-t)*6
		}
		return p
	}
	return rgb{ch(hk+1.0/3) * 255, ch(hk) * 255, ch(hk-1.0/3) * 255}.hex()
}

// AtLeastLightness raises a colour's lightness to l, keeping its hue, so it
// reads on a dark background.
func AtLeastLightness(hex string, l float64) string {
	h, s, cur := ToHSL(hex)
	if cur >= l {
		return hex
	}
	return FromHSL(h, math.Min(s, 0.85), l)
}
