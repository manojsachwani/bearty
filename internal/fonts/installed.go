// Package fonts finds typefaces Bear can use: fonts installed on the Mac,
// Bear's own bundled fonts, and Google Fonts that can be downloaded.
package fonts

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"golang.org/x/image/font/sfnt"
)

// Face is one installed font style.
type Face struct {
	Family     string
	Style      string
	PostScript string
	Path       string
	BearOnly   bool // bundled inside Bear.app, so only Bear can use it
}

// Dirs are searched for installed fonts. BearDir is added by Installed.
func Dirs() []string {
	home, _ := os.UserHomeDir()
	return []string{
		"/System/Library/Fonts",
		"/System/Library/Fonts/Supplemental",
		"/Library/Fonts",
		filepath.Join(home, "Library/Fonts"),
	}
}

// Installed lists every face in dirs plus Bear's bundled fonts in bearDir,
// sorted by family then style. Hidden system faces (names starting with ".")
// are skipped.
func Installed(dirs []string, bearDir string) []Face {
	type job struct {
		path     string
		bearOnly bool
	}
	var jobs []job
	for _, d := range dirs {
		for _, p := range fontFiles(d) {
			jobs = append(jobs, job{p, false})
		}
	}
	for _, p := range fontFiles(bearDir) {
		jobs = append(jobs, job{p, true})
	}

	var (
		mu    sync.Mutex
		wg    sync.WaitGroup
		seen  = map[string]bool{}
		faces []Face
		sem   = make(chan struct{}, 8)
	)
	for _, j := range jobs {
		wg.Add(1)
		sem <- struct{}{}
		go func(j job) {
			defer wg.Done()
			defer func() { <-sem }()
			for _, f := range readFaces(j.path) {
				f.BearOnly = j.bearOnly
				mu.Lock()
				if !seen[f.PostScript] {
					seen[f.PostScript] = true
					faces = append(faces, f)
				}
				mu.Unlock()
			}
		}(j)
	}
	wg.Wait()

	sort.Slice(faces, func(i, j int) bool {
		a, b := strings.ToLower(faces[i].Family), strings.ToLower(faces[j].Family)
		if a != b {
			return a < b
		}
		return styleRank(faces[i].Style) < styleRank(faces[j].Style) ||
			(styleRank(faces[i].Style) == styleRank(faces[j].Style) && faces[i].Style < faces[j].Style)
	})
	return faces
}

// styleRank puts Regular first so it's the default pick.
func styleRank(s string) int {
	switch strings.ToLower(s) {
	case "regular", "book", "roman", "normal":
		return 0
	}
	return 1
}

func fontFiles(dir string) []string {
	if dir == "" {
		return nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		switch strings.ToLower(filepath.Ext(e.Name())) {
		case ".ttf", ".otf", ".ttc", ".otc":
			if !e.IsDir() {
				out = append(out, filepath.Join(dir, e.Name()))
			}
		}
	}
	return out
}

// readFaces reads only each face's name table, not the whole file.
func readFaces(path string) []Face {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var fonts []*sfnt.Font
	if c, err := sfnt.ParseCollectionReaderAt(f); err == nil {
		for i := 0; i < c.NumFonts(); i++ {
			if font, err := c.Font(i); err == nil {
				fonts = append(fonts, font)
			}
		}
	} else if font, err := sfnt.ParseReaderAt(f); err == nil {
		fonts = append(fonts, font)
	}

	var faces []Face
	var buf sfnt.Buffer
	for _, font := range fonts {
		ps, err := font.Name(&buf, sfnt.NameIDPostScript)
		if err != nil || ps == "" || strings.HasPrefix(ps, ".") {
			continue
		}
		family := name(font, &buf, sfnt.NameIDTypographicFamily, sfnt.NameIDFamily)
		style := name(font, &buf, sfnt.NameIDTypographicSubfamily, sfnt.NameIDSubfamily)
		if family == "" || strings.HasPrefix(family, ".") {
			continue
		}
		faces = append(faces, Face{Family: family, Style: style, PostScript: ps, Path: path})
	}
	return faces
}

// name returns the first non-empty name among ids.
func name(f *sfnt.Font, buf *sfnt.Buffer, ids ...sfnt.NameID) string {
	for _, id := range ids {
		if s, err := f.Name(buf, id); err == nil && s != "" {
			return s
		}
	}
	return ""
}

// Families groups faces by family, keeping sort order.
func Families(faces []Face) []string {
	var out []string
	for i, f := range faces {
		if i == 0 || f.Family != faces[i-1].Family {
			out = append(out, f.Family)
		}
	}
	return out
}

// Styles returns the faces of one family.
func Styles(faces []Face, family string) []Face {
	var out []Face
	for _, f := range faces {
		if f.Family == family {
			out = append(out, f)
		}
	}
	return out
}

// ByPostScript finds a face.
func ByPostScript(faces []Face, ps string) (Face, bool) {
	for _, f := range faces {
		if f.PostScript == ps {
			return f, true
		}
	}
	return Face{}, false
}
