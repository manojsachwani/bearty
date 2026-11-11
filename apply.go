package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/manojsachwani/bearty/internal/fonts"
	"github.com/manojsachwani/bearty/internal/install"
	"github.com/manojsachwani/bearty/internal/library"
	"github.com/manojsachwani/bearty/internal/theme"
	"github.com/manojsachwani/bearty/internal/ui"
)

// request is everything one run of bearty does with a theme. The TUI and
// the commands both end up here.
type request struct {
	Entry         library.Saved // Slot and DarkSlot say where to install
	Install       bool
	Export        string // .theme path to write, or ""
	SetTypography bool   // also set Bear's own typography to match
	Restart       bool   // quit Bear first and open it again afterwards
}

// apply saves the theme to the library, then exports and installs it.
func apply(r request, faces []fonts.Face) error {
	entry := r.Entry
	if !r.Install { // only remember slots that were really installed
		if prev, ok := findSaved(entry.Name); ok {
			entry.Slot, entry.DarkSlot = prev.Slot, prev.DarkSlot
		} else {
			entry.Slot, entry.DarkSlot = "", ""
		}
	}
	// Saved first, so nothing chosen is lost if a later step fails.
	if err := save(entry); err != nil {
		return err
	}

	fontPaths, learned, err := resolveFonts(&entry, faces, r.Install)
	if err != nil {
		return err
	}
	if learned { // a Google font's PostScript name is now known
		if err := save(entry); err != nil {
			return err
		}
	}

	light := entry.Options()
	data, err := theme.Build(light).JSON()
	if err != nil {
		return err
	}
	var darkData []byte
	if entry.Dark.Enabled() {
		if darkData, err = theme.Build(light.DarkVariant(entry.Dark)).JSON(); err != nil {
			return err
		}
	}

	if r.Export != "" {
		if err := export(r.Export, data, darkData); err != nil {
			return err
		}
		if !r.Install && entry.FontSource != fonts.SourceInstalled && !haveFont(faces, light.Font.PostScript) {
			fmt.Println(warn, light.Font.Name, "isn't on this Mac yet. Install the theme with bearty to add it to Bear.")
		}
	}
	if !r.Install {
		return nil
	}
	return installBear(entry, data, darkData, usedBeartyFonts(entry, faces, fontPaths), r)
}

func save(entry library.Saved) error {
	path, err := library.Save(entry)
	if err != nil {
		return fmt.Errorf("save theme: %w", err)
	}
	fmt.Println(ok, "Saved", entry.Name, "to", path)
	return nil
}

// export writes the theme, and its dark version next to it as "<name>-dark.theme".
func export(path string, light, dark []byte) error {
	if err := os.WriteFile(path, light, 0o644); err != nil {
		return err
	}
	abs, _ := filepath.Abs(path)
	fmt.Println(ok, "Exported", abs)
	if dark != nil {
		dpath := strings.TrimSuffix(path, ".theme") + "-dark.theme"
		if err := os.WriteFile(dpath, dark, 0o644); err != nil {
			return err
		}
		abs, _ := filepath.Abs(dpath)
		fmt.Println(ok, "Exported", abs)
	}
	return nil
}

func findSaved(name string) (library.Saved, bool) {
	list, _ := library.List()
	for _, s := range list {
		if library.Slug(s.Name) == library.Slug(name) {
			return s, true
		}
	}
	return library.Saved{}, false
}

func haveFont(faces []fonts.Face, ps string) bool {
	_, ok := fonts.ByPostScript(faces, ps)
	return ok
}

// resolveFonts downloads the text, heading and code fonts when Bear will
// need them. A Google text font with no PostScript name yet is looked up
// even for an export. learned reports that entry.Font was filled in.
func resolveFonts(entry *library.Saved, faces []fonts.Face, installing bool) (paths []string, learned bool, err error) {
	path, err := fetchFont(entry.FontSource, entry.Font, faces, installing, func(ps string) {
		learned = entry.Font != ps
		entry.Font = ps
	})
	if err != nil {
		return nil, learned, fmt.Errorf("%w (fix it with Edit a saved theme)", err)
	}
	add := func(p string) {
		if p != "" && !slices.Contains(paths, p) {
			paths = append(paths, p)
		}
	}
	add(path)
	if entry.HeadingFontSource != "" || (entry.HeadingFont != "" && entry.HeadingFont != entry.Font) {
		path, err := fetchFont(entry.HeadingFontSource, entry.HeadingFont, faces, installing, func(ps string) {
			learned = learned || entry.HeadingFont != ps
			entry.HeadingFont = ps
		})
		if err != nil {
			return nil, learned, fmt.Errorf("heading font: %w", err)
		}
		add(path)
	}
	if entry.Code.Font != "" {
		path, err := fetchFont(entry.CodeSource, entry.Code.Font, faces, installing, nil)
		if err != nil {
			return nil, learned, fmt.Errorf("code font: %w", err)
		}
		add(path)
	}
	return paths, learned, nil
}

// fetchFont downloads one font if Bear needs it and it isn't on this Mac.
// learned, if set, receives the PostScript name of a Google font.
func fetchFont(source, ps string, faces []fonts.Face, installing bool, learned func(string)) (path string, err error) {
	needFile := installing && (ps == "" || !haveFont(faces, ps))
	cache, err := install.CacheDir()
	if err != nil {
		return "", err
	}
	switch family, isGoogle := fonts.GoogleFamily(source); {
	case isGoogle && (ps == "" || needFile):
		style := fonts.GoogleStyleOf(source)
		err = ui.Spin("Fetching "+family+" "+fonts.StyleLabel(style)+" from Google Fonts…", func() error {
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()
			file, got, err := fonts.FetchGoogleStyle(ctx, family, style, cache)
			if err != nil {
				return err
			}
			if learned != nil {
				learned(got)
			}
			if installing && !haveFont(faces, got) {
				path = file
			}
			return nil
		})
	case source == fonts.SourceInkina && needFile:
		err = ui.Spin("Fetching Inkina…", func() error {
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()
			path, err = install.FetchInkina(ctx)
			return err
		})
	}
	return path, err
}

// usedBeartyFonts are the font files this theme needs from bearty: the ones
// just downloaded plus any it uses that bearty added to Bear earlier. They're
// recorded so a Bear update that removes them can be noticed.
func usedBeartyFonts(entry library.Saved, faces []fonts.Face, fresh []string) []string {
	files := slices.Clone(fresh)
	added := install.AddedFonts()
	for _, ps := range []string{entry.Font, entry.HeadingFont, entry.Code.Font} {
		if f, ok := fonts.ByPostScript(faces, ps); ok && f.BearOnly && slices.Contains(added, filepath.Base(f.Path)) {
			if !slices.ContainsFunc(files, func(p string) bool { return filepath.Base(p) == filepath.Base(f.Path) }) {
				files = append(files, filepath.Join(install.FontsDir(), filepath.Base(f.Path)))
			}
		}
	}
	return files
}

// installBear puts the theme (and its dark version) into Bear, selects it,
// and records what was installed.
func installBear(entry library.Saved, light, dark []byte, fontFiles []string, r request) error {
	slots := []string{entry.Slot}
	if dark != nil {
		slots = append(slots, entry.DarkSlot)
	}

	wasRunning := install.BearRunning()
	if r.Restart && wasRunning {
		fmt.Println("Quitting Bear…")
		if err := install.QuitBear(); err != nil {
			return err
		}
	}

	for _, slot := range slots {
		backup, created, err := install.Backup(slot)
		if err != nil {
			return fmt.Errorf("back up %s: %w", slot, err)
		}
		if created {
			fmt.Println(ok, "Backed up the original", slot, "to", backup)
		}
	}

	fmt.Println("bearty needs your password (sudo) to copy files into Bear.app.")
	for _, f := range fontFiles {
		if filepath.Dir(f) == install.FontsDir() {
			continue // already inside Bear
		}
		copied, err := install.Font(f)
		if err != nil {
			return err
		}
		if copied {
			fmt.Println(ok, "Installed", filepath.Base(f), "into Bear")
		}
	}
	if err := install.Theme(entry.Slot, light); err != nil {
		return err
	}
	fmt.Printf("%s Replaced %q with %s\n", ok, entry.Slot, entry.Name)
	if dark != nil {
		if err := install.Theme(entry.DarkSlot, dark); err != nil {
			return err
		}
		fmt.Printf("%s Replaced %q with %s (dark)\n", ok, entry.DarkSlot, entry.Name)
	}

	if err := setBearPrefs(entry, r.SetTypography); err != nil {
		return err
	}
	if err := record(entry, light, dark, fontFiles, r.SetTypography); err != nil {
		fmt.Println(warn, "Couldn't record the install, so bearty can't re-apply it after a Bear update:", err)
	}
	finish(r.Restart, wasRunning)
	return nil
}

// setBearPrefs selects the theme for light and dark mode and, if asked,
// sets Bear's typography to match. The user's own settings are saved first.
func setBearPrefs(entry library.Saved, typography bool) error {
	path, err := library.TypographyBackupPath()
	if err != nil {
		return err
	}
	if created, err := install.BackupTypography(path); err != nil {
		return fmt.Errorf("back up Bear's settings: %w", err)
	} else if created {
		fmt.Println(ok, "Saved your Bear settings to", path)
	}
	darkSlot := entry.Slot // without a dark version, use the theme in dark mode too
	if entry.Dark.Enabled() {
		darkSlot = entry.DarkSlot
	}
	if err := install.SelectThemes(entry.Slot, darkSlot); err != nil {
		return err
	}
	fmt.Println(ok, "Selected it in Bear for light and dark mode")
	if typography {
		if err := install.ApplyTypography(install.TypographyPrefs(entry.Options())); err != nil {
			return err
		}
		fmt.Println(ok, "Set Bear's typography to match")
	}
	return nil
}

func record(entry library.Saved, light, dark []byte, fontFiles []string, typography bool) error {
	dir, err := library.ConfigDir()
	if err != nil {
		return err
	}
	copies := filepath.Join(dir, "installed")
	in := install.Installed{Theme: entry.Name, Typography: typography, At: time.Now()}
	if in.Light, err = install.NewSlotFile(entry.Slot, light, copies); err != nil {
		return err
	}
	if dark != nil {
		d, err := install.NewSlotFile(entry.DarkSlot, dark, copies)
		if err != nil {
			return err
		}
		in.Dark = &d
	}
	for _, f := range fontFiles {
		in.Fonts = append(in.Fonts, filepath.Base(f))
	}
	return install.SaveInstalled(filepath.Join(dir, "installed.json"), in)
}

// finish reopens Bear, or says how to see the change.
func finish(restart, wasRunning bool) {
	fmt.Println()
	switch {
	case restart:
		if err := install.OpenBear(); err != nil {
			fmt.Println(warn, err)
			return
		}
		fmt.Println(ok, "Opened Bear with the new theme")
	case wasRunning:
		fmt.Println("Quit Bear (⌘Q) and reopen it to see the theme.")
	default:
		fmt.Println("Open Bear to see the theme.")
	}
	fmt.Println("If a Bear update removes it, run bearty again and choose Re-apply, or run `bearty reapply`.")
}

// installedPath is where the install record lives.
func installedPath() string {
	dir, err := library.ConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "installed.json")
}

// reapply puts back exactly what bearty last installed, from its own copies
// and font cache, after a Bear update has removed it.
func reapply(restart bool) error {
	in, found := install.LoadInstalled(installedPath())
	if !found {
		return fmt.Errorf("nothing to re-apply: bearty hasn't installed a theme yet")
	}
	wasRunning := install.BearRunning()
	if restart && wasRunning {
		fmt.Println("Quitting Bear…")
		if err := install.QuitBear(); err != nil {
			return err
		}
	}
	cache, err := install.CacheDir()
	if err != nil {
		return err
	}
	fmt.Println("bearty needs your password (sudo) to copy files into Bear.app.")
	for _, f := range in.Fonts {
		src := filepath.Join(cache, f)
		if _, err := os.Stat(src); err != nil {
			fmt.Println(warn, f, "isn't in bearty's cache any more; install the theme again to download it")
			continue
		}
		if copied, err := install.Font(src); err != nil {
			return err
		} else if copied {
			fmt.Println(ok, "Put", f, "back into Bear")
		}
	}
	slots := []install.SlotFile{in.Light}
	if in.Dark != nil {
		slots = append(slots, *in.Dark)
	}
	for _, s := range slots {
		data, err := os.ReadFile(s.Copy)
		if err != nil {
			return fmt.Errorf("bearty's copy of %s is missing (%w); install the theme again", s.Slot, err)
		}
		if _, _, err := install.Backup(s.Slot); err != nil {
			return err
		}
		if err := install.Theme(s.Slot, data); err != nil {
			return err
		}
		fmt.Printf("%s Put %s back in %q\n", ok, in.Theme, s.Slot)
	}
	finish(restart, wasRunning)
	return nil
}

// useTheme switches Bear to a theme bearty installed: it selects the theme
// and sets Bear's typography to its fonts and sizes. Bear ignores the fonts
// inside theme files and draws every theme with Settings → Typography, so
// switching in Bear's own Themes pane can't change the font; this can.
// If the installed files are missing or out of date, it installs again.
func useTheme(entry library.Saved, faces []fonts.Face, restart bool) error {
	if entry.Slot == "" {
		return fmt.Errorf("%s isn't installed yet; install it first", entry.Name)
	}
	light := entry.Options()
	data, err := theme.Build(light).JSON()
	if err != nil {
		return err
	}
	var dark []byte
	if entry.Dark.Enabled() {
		if dark, err = theme.Build(light.DarkVariant(entry.Dark)).JSON(); err != nil {
			return err
		}
	}
	current := install.SlotHas(entry.Slot, data) && (dark == nil || install.SlotHas(entry.DarkSlot, dark))
	for _, ps := range []string{entry.Font, entry.HeadingFont, entry.Code.Font} {
		if ps != "" && !haveFont(faces, ps) {
			current = false // a font went missing, e.g. after a Bear update
		}
	}
	if !current {
		fmt.Println("Installing", entry.Name, "again, since Bear doesn't have the current version…")
		return apply(request{Entry: entry, Install: true, SetTypography: true, Restart: restart}, faces)
	}

	wasRunning := install.BearRunning()
	if restart && wasRunning {
		fmt.Println("Quitting Bear…")
		if err := install.QuitBear(); err != nil {
			return err
		}
	}
	if err := setBearPrefs(entry, true); err != nil {
		return err
	}
	finish(restart, wasRunning)
	return nil
}
