package theme

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Bear theme keys contain spaces but never dots, so a dotted path such as
// "editor.headers.text color" names one value.

// SetPath sets a value in nested maps, creating sections as needed.
func SetPath(m map[string]any, path string, v any) {
	parts := strings.Split(path, ".")
	for _, p := range parts[:len(parts)-1] {
		next, ok := m[p].(map[string]any)
		if !ok {
			next = map[string]any{}
			m[p] = next
		}
		m = next
	}
	m[parts[len(parts)-1]] = v
}

// Flatten turns a theme into dotted path → value.
func Flatten(m map[string]any) map[string]any {
	out := map[string]any{}
	var walk func(prefix string, v any)
	walk = func(prefix string, v any) {
		if sub, ok := v.(map[string]any); ok {
			for k, x := range sub {
				p := k
				if prefix != "" {
					p = prefix + "." + k
				}
				walk(p, x)
			}
			return
		}
		out[prefix] = v
	}
	walk("", m)
	return out
}

// Resolve follows "$section.key" references until each value is concrete.
// References that lead nowhere, or in a loop, are left as they are.
func Resolve(flat map[string]any) map[string]any {
	out := make(map[string]any, len(flat))
	for k := range flat {
		v := flat[k]
		for i := 0; i < 10; i++ {
			s, ok := v.(string)
			if !ok || !strings.HasPrefix(s, "$") {
				break
			}
			next, found := flat[s[1:]]
			if !found {
				break
			}
			v = next
		}
		out[k] = v
	}
	return out
}

// Load reads a theme file and merges in the theme it builds on, if its
// "meta" names a "base theme" found in dir (Bear's themes folder). The
// result has no "meta" section.
func Load(path, dir string) (map[string]any, error) {
	return load(path, dir, 0)
}

func load(path, dir string, depth int) (map[string]any, error) {
	if depth > 5 {
		return nil, fmt.Errorf("%s: base themes nest too deeply", path)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("%s is not a Bear theme: %w", filepath.Base(path), err)
	}
	meta, _ := m["meta"].(map[string]any)
	delete(m, "meta")
	base, _ := meta["base theme"].(string)
	if base == "" {
		return m, nil
	}
	parent, err := load(filepath.Join(dir, base+".theme"), dir, depth+1)
	if err != nil {
		return nil, fmt.Errorf("base theme %q: %w", base, err)
	}
	merge(parent, m)
	return parent, nil
}

// merge copies src over dst, section by section.
func merge(dst, src map[string]any) {
	for k, v := range src {
		if sub, ok := v.(map[string]any); ok {
			if d, ok := dst[k].(map[string]any); ok {
				merge(d, sub)
				continue
			}
		}
		dst[k] = v
	}
}

// Values is a theme's concrete values by dotted path, references resolved.
func (t Theme) Values() (map[string]any, error) {
	b, err := t.JSON()
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return Resolve(Flatten(m)), nil
}
