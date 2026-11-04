// Package install puts generated themes and fonts into Bear.app.
//
// Bear has no theme import, so a theme replaces one of the built-in slots.
// Everything inside /Applications/Bear.app needs sudo, and a Bear update puts
// the original files back.
package install

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// BearApp is where Bear is installed.
var BearApp = "/Applications/Bear.app"

// ThemesDir holds Bear's built-in .theme files.
func ThemesDir() string {
	return filepath.Join(BearApp, "Contents/Frameworks/BearCore.framework/Versions/A/Resources")
}

// FontsDir is loaded by Bear at launch (ATSApplicationFontsPath = ".").
func FontsDir() string { return filepath.Join(BearApp, "Contents/Resources") }

// Slots lists the built-in theme names that can be replaced.
func Slots() ([]string, error) {
	paths, err := filepath.Glob(filepath.Join(ThemesDir(), "*.theme"))
	if err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("no themes found in %s; is Bear installed?", ThemesDir())
	}
	names := make([]string, len(paths))
	for i, p := range paths {
		names[i] = strings.TrimSuffix(filepath.Base(p), ".theme")
	}
	sort.Strings(names)
	return names, nil
}

// BackupPath is where the original slot theme is kept.
func BackupPath(slot string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Downloads", slot+".original.theme"), nil
}

// Backup copies the original slot theme to ~/Downloads once. A backup that
// already exists is never overwritten, so re-installing keeps the real original.
func Backup(slot string) (path string, created bool, err error) {
	path, err = BackupPath(slot)
	if err != nil {
		return "", false, err
	}
	if _, err := os.Stat(path); err == nil {
		return path, false, nil
	}
	b, err := os.ReadFile(filepath.Join(ThemesDir(), slot+".theme"))
	if err != nil {
		return "", false, err
	}
	return path, true, os.WriteFile(path, b, 0o644)
}

// Backups lists the slots that have an original saved by Backup.
func Backups(slots []string) []string {
	var have []string
	for _, s := range slots {
		if p, err := BackupPath(s); err == nil {
			if _, err := os.Stat(p); err == nil {
				have = append(have, s)
			}
		}
	}
	return have
}

// Restore copies the slot's backup back into Bear. The backup is kept.
func Restore(slot string) error {
	p, err := BackupPath(slot)
	if err != nil {
		return err
	}
	if _, err := os.Stat(p); err != nil {
		return fmt.Errorf("no backup for %s at %s", slot, p)
	}
	return sudoCopy(p, filepath.Join(ThemesDir(), slot+".theme"))
}

// AddedFonts lists fonts bearty downloaded that are now inside Bear.app.
// Everything bearty downloads is cached, so the cache is the record.
func AddedFonts() []string {
	dir, err := CacheDir()
	if err != nil {
		return nil
	}
	entries, _ := os.ReadDir(dir)
	var out []string
	for _, e := range entries {
		switch strings.ToLower(filepath.Ext(e.Name())) {
		case ".otf", ".ttf", ".ttc":
			if _, err := os.Stat(filepath.Join(FontsDir(), e.Name())); err == nil {
				out = append(out, e.Name())
			}
		}
	}
	return out
}

// RemoveFonts deletes fonts bearty copied into Bear.
func RemoveFonts(files []string) error {
	if len(files) == 0 {
		return nil
	}
	args := []string{"rm", "-f"}
	for _, f := range files {
		args = append(args, filepath.Join(FontsDir(), filepath.Base(f)))
	}
	cmd := exec.Command("sudo", args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	return cmd.Run()
}

// Theme replaces the slot's theme file with data.
func Theme(slot string, data []byte) error {
	tmp, err := os.CreateTemp("", "bearty-*.theme")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return sudoCopy(tmp.Name(), filepath.Join(ThemesDir(), slot+".theme"))
}

// Font copies an .otf into Bear's fonts folder unless an identical copy is there.
func Font(src string) (copied bool, err error) {
	dst := filepath.Join(FontsDir(), filepath.Base(src))
	want, err := os.ReadFile(src)
	if err != nil {
		return false, err
	}
	if have, err := os.ReadFile(dst); err == nil && bytes.Equal(have, want) {
		return false, nil
	}
	return true, sudoCopy(src, dst)
}

// BearRunning reports whether Bear is open.
func BearRunning() bool {
	return exec.Command("pgrep", "-x", "Bear").Run() == nil
}

func sudoCopy(src, dst string) error {
	cmd := exec.Command("sudo", "cp", src, dst)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("copy to %s: %w", dst, err)
	}
	return nil
}
