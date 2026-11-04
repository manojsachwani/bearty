package fonts

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// googleRaw serves the github.com/google/fonts repository. Each family has a
// METADATA.pb listing its files and PostScript names, so no API key is needed.
var googleRaw = "https://raw.githubusercontent.com/google/fonts/main"

// Google Fonts are OFL, Apache or UFL licensed; the repo is split by licence.
var licences = []string{"ofl", "apache", "ufl"}

// GoogleFont is one downloadable file of a Google Fonts family.
type GoogleFont struct {
	Family     string
	PostScript string
	URL        string
	File       string
}

// GoogleSlug is the repository folder name for a family.
func GoogleSlug(family string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(family) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// LookupGoogle finds the regular (weight 400, upright) file of a family.
func LookupGoogle(ctx context.Context, family string) (GoogleFont, error) {
	slug := GoogleSlug(family)
	if slug == "" {
		return GoogleFont{}, fmt.Errorf("empty font name")
	}
	for _, lic := range licences {
		base := googleRaw + "/" + lic + "/" + slug
		body, err := get(ctx, base+"/METADATA.pb", 1<<20)
		if err != nil {
			continue
		}
		name, faces := parseMetadata(string(body))
		f, ok := regular(faces)
		if !ok {
			return GoogleFont{}, fmt.Errorf("%s has no upright style", family)
		}
		return GoogleFont{Family: name, PostScript: f.postScript, URL: base + "/" + f.file, File: f.file}, nil
	}
	return GoogleFont{}, fmt.Errorf("%q isn't on Google Fonts (check the spelling at fonts.google.com)", family)
}

// FetchGoogle downloads the font into dir, reusing a cached copy.
func FetchGoogle(ctx context.Context, g GoogleFont, dir string) (string, error) {
	path := filepath.Join(dir, g.File)
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}
	body, err := get(ctx, g.URL, 20<<20)
	if err != nil {
		return "", fmt.Errorf("download %s: %w", g.Family, err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return path, os.WriteFile(path, body, 0o644)
}

func get(ctx context.Context, url string, limit int64) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s", resp.Status)
	}
	return io.ReadAll(io.LimitReader(resp.Body, limit))
}

type metaFace struct {
	style, file, postScript string
	weight                  int
}

// parseMetadata reads the text-format protobuf: a top-level name and one
// "fonts { … }" block per file.
func parseMetadata(s string) (family string, faces []metaFace) {
	var cur *metaFace
	sc := bufio.NewScanner(strings.NewReader(s))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		key, val, ok := strings.Cut(line, ":")
		val = strings.Trim(strings.TrimSpace(val), `"`)
		switch {
		case line == "fonts {":
			faces = append(faces, metaFace{})
			cur = &faces[len(faces)-1]
		case line == "}":
			cur = nil
		case !ok:
		case cur == nil && key == "name":
			family = val
		case cur != nil && key == "style":
			cur.style = val
		case cur != nil && key == "weight":
			cur.weight, _ = strconv.Atoi(val)
		case cur != nil && key == "filename":
			cur.file = val
		case cur != nil && key == "post_script_name":
			cur.postScript = val
		}
	}
	return family, faces
}

// regular picks the upright face closest to weight 400.
func regular(faces []metaFace) (metaFace, bool) {
	best, found := metaFace{}, false
	for _, f := range faces {
		if f.style != "normal" || f.file == "" {
			continue
		}
		if !found || abs(f.weight-400) < abs(best.weight-400) {
			best, found = f, true
		}
	}
	return best, found
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
