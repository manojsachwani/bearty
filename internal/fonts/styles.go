package fonts

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// RegularStyle is upright weight 400.
const RegularStyle = "400"

// A style is a weight plus "i" for italic, e.g. "700" or "300i".

// ParseStyle splits a style into weight and italic.
func ParseStyle(style string) (weight int, italic bool) {
	italic = strings.HasSuffix(style, "i")
	weight, err := strconv.Atoi(strings.TrimSuffix(style, "i"))
	if err != nil || weight == 0 {
		weight = 400
	}
	return weight, italic
}

var weightNames = map[int]string{
	100: "Thin", 200: "ExtraLight", 300: "Light", 400: "Regular", 500: "Medium",
	600: "SemiBold", 700: "Bold", 800: "ExtraBold", 900: "Black",
}

// StyleLabel names a style, e.g. "Bold Italic".
func StyleLabel(style string) string {
	w, italic := ParseStyle(style)
	name, ok := weightNames[w]
	if !ok {
		name = strconv.Itoa(w)
	}
	if italic {
		if w == 400 {
			return "Italic"
		}
		return name + " Italic"
	}
	return name
}

var axisRe = regexp.MustCompile(`(?s)axes\s*\{[^}]*tag:\s*"wght"[^}]*min_value:\s*([0-9.]+)[^}]*max_value:\s*([0-9.]+)`)

// GoogleStyles lists the weights and italics a Google family offers, from
// its METADATA.pb. Variable fonts offer every hundred within their range.
func GoogleStyles(ctx context.Context, family string) ([]string, error) {
	meta, err := metadata(ctx, family)
	if err != nil {
		return nil, err
	}
	_, faces := parseMetadata(meta)
	seen := map[string]bool{}
	var italics bool
	for _, f := range faces {
		style := strconv.Itoa(f.weight)
		if f.style == "italic" {
			style += "i"
			italics = true
		}
		seen[style] = true
	}
	if m := axisRe.FindStringSubmatch(meta); m != nil {
		lo, _ := strconv.ParseFloat(m[1], 64)
		hi, _ := strconv.ParseFloat(m[2], 64)
		for w := 100; w <= 900; w += 100 {
			if float64(w) >= lo && float64(w) <= hi {
				seen[strconv.Itoa(w)] = true
				if italics {
					seen[strconv.Itoa(w)+"i"] = true
				}
			}
		}
	}
	styles := make([]string, 0, len(seen))
	for s := range seen {
		styles = append(styles, s)
	}
	sort.Slice(styles, func(i, j int) bool {
		wi, ii := ParseStyle(styles[i])
		wj, ij := ParseStyle(styles[j])
		if wi != wj {
			return wi < wj
		}
		return !ii && ij
	})
	if len(styles) == 0 {
		return []string{RegularStyle}, nil
	}
	return styles, nil
}

// metadata fetches a family's METADATA.pb from whichever licence folder has it.
func metadata(ctx context.Context, family string) (string, error) {
	slug := GoogleSlug(family)
	if slug == "" {
		return "", fmt.Errorf("empty font name")
	}
	for _, lic := range licences {
		if body, err := get(ctx, googleRaw+"/"+lic+"/"+slug+"/METADATA.pb", 1<<20); err == nil {
			return string(body), nil
		}
	}
	return "", fmt.Errorf("%q isn't on Google Fonts (check the spelling at fonts.google.com)", family)
}

// googleCSS serves @font-face rules; asked without a browser user agent it
// links plain TrueType files cut to one weight, even for variable fonts.
var googleCSS = "https://fonts.googleapis.com/css2"

var cssURL = regexp.MustCompile(`url\((https://[^)]+)\)`)

// FetchGoogleStyle downloads one style of a Google family into dir, reusing
// a cached copy, and returns the file and its PostScript name.
func FetchGoogleStyle(ctx context.Context, family, style, dir string) (path, postScript string, err error) {
	weight, italic := ParseStyle(style)
	file := strings.ReplaceAll(family, " ", "") + "-" + strconv.Itoa(weight)
	ital := "0"
	if italic {
		file += "i"
		ital = "1"
	}
	path = filepath.Join(dir, file+".ttf")
	if _, statErr := os.Stat(path); statErr != nil {
		q := url.QueryEscape(family) + ":ital,wght@" + ital + "," + strconv.Itoa(weight)
		css, err := get(ctx, googleCSS+"?family="+strings.ReplaceAll(q, "%2C", ","), 1<<20)
		if err != nil {
			return "", "", fmt.Errorf("%s %s isn't available: %w", family, StyleLabel(style), err)
		}
		m := cssURL.FindSubmatch(css)
		if m == nil {
			return "", "", fmt.Errorf("%s %s isn't available", family, StyleLabel(style))
		}
		body, err := get(ctx, string(m[1]), 20<<20)
		if err != nil {
			return "", "", fmt.Errorf("download %s: %w", family, err)
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", "", err
		}
		if err := os.WriteFile(path, body, 0o644); err != nil {
			return "", "", err
		}
	}
	faces := readFaces(path)
	if len(faces) == 0 {
		os.Remove(path)
		return "", "", fmt.Errorf("the %s download isn't a font", family)
	}
	return path, faces[0].PostScript, nil
}
