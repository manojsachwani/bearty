// Command bearty builds custom themes for the Bear notes app and can
// install them into one of Bear's built-in theme slots.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/manojsachwani/bearty/internal/fonts"
	"github.com/manojsachwani/bearty/internal/install"
	"github.com/manojsachwani/bearty/internal/library"
	"github.com/manojsachwani/bearty/internal/theme"
	"github.com/manojsachwani/bearty/internal/ui"
)

var version = "dev"

var (
	ok   = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Inks[0].Hex)).Render("✓")
	warn = lipgloss.NewStyle().Foreground(lipgloss.Color("#E5B92E")).Render("!")
)

func main() {
	showVersion := flag.Bool("version", false, "print the version and exit")
	noMotion := flag.Bool("no-motion", os.Getenv("BEARTY_NO_MOTION") != "", "skip the splash screen and animations (or set BEARTY_NO_MOTION)")
	flag.Parse()
	if *showVersion {
		fmt.Println("bearty", version)
		return
	}
	if err := run(*noMotion); err != nil {
		if errors.Is(err, ui.ErrAborted) {
			return
		}
		fmt.Fprintln(os.Stderr, "bearty:", err)
		os.Exit(1)
	}
}

func run(noMotion bool) error {
	slots, _ := install.Slots() // no slots just means Bear isn't installed
	saved, libErr := library.List()
	if libErr != nil {
		fmt.Fprintln(os.Stderr, warn, libErr)
	}
	var faces []fonts.Face
	c, err := ui.Run(ui.Env{
		Slots:            slots,
		Backups:          install.Backups(slots),
		AddedFonts:       install.AddedFonts(),
		TypographyBackup: typographyBackupExists(),
		Saved:            saved,
		LoadFaces: func() []fonts.Face {
			faces = fonts.Installed(fonts.Dirs(), install.FontsDir())
			return faces
		},
		Version:  version,
		NoMotion: noMotion,
	})
	if err != nil {
		return err
	}
	if c.Restoring() {
		return restoreTheme(c)
	}

	installing := c.Installing() && c.Confirm

	// Saved first, so nothing chosen is lost if a later step fails.
	save := func() error {
		entry := c.Saved()
		if !installing {
			entry.Slot = "" // only remember slots that were really installed
			if prev, ok := findSaved(saved, entry.Name); ok {
				entry.Slot = prev.Slot
			}
		}
		path, err := library.Save(entry)
		if err != nil {
			return fmt.Errorf("save theme: %w", err)
		}
		fmt.Println(ok, "Saved", entry.Name, "to", path)
		return nil
	}
	if err := save(); err != nil {
		return err
	}

	fontPaths, resolved, err := resolveFonts(c, faces, installing)
	if err != nil {
		return err
	}
	if resolved { // a Google font's PostScript name is now known
		if err := save(); err != nil {
			return err
		}
	}

	opts := c.Options()
	data, err := theme.Build(opts).JSON()
	if err != nil {
		return err
	}

	if c.Exporting() {
		if err := os.WriteFile(c.SavePath, data, 0o644); err != nil {
			return err
		}
		abs, _ := filepath.Abs(c.SavePath)
		fmt.Println(ok, "Exported", abs)
		if !installing && c.FontSource() != fonts.SourceInstalled && !haveFont(faces, opts.Font.PostScript) {
			fmt.Println(warn, opts.Font.Name, "isn't on this Mac yet. Install the theme with bearty to add it to Bear.")
		}
	}

	if c.Installing() && !c.Confirm {
		fmt.Println(warn, "Install cancelled; Bear was not changed.")
		return nil
	}
	if !installing {
		return nil
	}
	var typography []install.Pref
	if c.SetTypography {
		typography = install.TypographyPrefs(opts)
	}
	return installTheme(c.Slot, data, fontPaths, typography)
}

func typographyBackupExists() bool {
	path, err := library.TypographyBackupPath()
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}

func haveFont(faces []fonts.Face, ps string) bool {
	_, ok := fonts.ByPostScript(faces, ps)
	return ok
}

// resolveFonts downloads the theme's text and heading fonts when Bear will
// need them. For a Google text font it also looks up the PostScript name,
// which an export needs too. It returns the downloaded files and whether c
// gained a PostScript name.
func resolveFonts(c *ui.Choices, faces []fonts.Face, installing bool) (paths []string, resolved bool, err error) {
	font := c.Font()
	path, resolved, err := fetchFont(c.FontSource(), font.PostScript, faces, installing, func(ps string) {
		resolved = c.GooglePS != ps
		c.GooglePS = ps
	})
	if err != nil {
		return nil, resolved, fmt.Errorf("%w (fix it with Edit a saved theme)", err)
	}
	if path != "" {
		paths = append(paths, path)
	}
	if c.HeadingFont != "" && c.HeadingFont != font.PostScript {
		path, _, err := fetchFont(c.HeadingFontSource(), c.HeadingFont, faces, installing, nil)
		if err != nil {
			return nil, resolved, fmt.Errorf("heading font: %w", err)
		}
		if path != "" {
			paths = append(paths, path)
		}
	}
	return paths, resolved, nil
}

// fetchFont downloads one font if Bear needs it and it isn't on this Mac. A
// Google font with no known PostScript name is looked up even for an export;
// learned receives the name.
func fetchFont(source, ps string, faces []fonts.Face, installing bool, learned func(string)) (path string, resolved bool, err error) {
	needFile := installing && !haveFont(faces, ps)
	cache, err := install.CacheDir()
	if err != nil {
		return "", false, err
	}
	switch family, isGoogle := fonts.GoogleFamily(source); {
	case isGoogle && (ps == "" || needFile):
		err = ui.Spin("Fetching "+family+" from Google Fonts…", func() error {
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()
			g, err := fonts.LookupGoogle(ctx, family)
			if err != nil {
				return err
			}
			if learned != nil {
				learned(g.PostScript)
				resolved = true
			}
			if installing && !haveFont(faces, g.PostScript) {
				path, err = fonts.FetchGoogle(ctx, g, cache)
			}
			return err
		})
	case source == fonts.SourceInkina && needFile:
		err = ui.Spin("Fetching Inkina…", func() error {
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()
			path, err = install.FetchInkina(ctx)
			return err
		})
	}
	return path, resolved, err
}

func findSaved(list []library.Saved, name string) (library.Saved, bool) {
	for _, s := range list {
		if library.Slug(s.Name) == library.Slug(name) {
			return s, true
		}
	}
	return library.Saved{}, false
}

func restoreTheme(c *ui.Choices) error {
	if !c.RestoreConfirm {
		fmt.Println(warn, "Restore cancelled; Bear was not changed.")
		return nil
	}
	fmt.Println("bearty needs your password (sudo) to copy files into Bear.app.")
	if err := install.Restore(c.RestoreSlot); err != nil {
		return err
	}
	fmt.Printf("%s Restored Bear's original %q\n", ok, c.RestoreSlot)
	if added := install.AddedFonts(); c.RemoveFonts && len(added) > 0 {
		if err := install.RemoveFonts(added); err != nil {
			return fmt.Errorf("remove fonts: %w", err)
		}
		fmt.Println(ok, "Removed", strings.Join(added, ", "), "from Bear")
	}
	if c.RestoreTypography {
		if path, err := library.TypographyBackupPath(); err == nil {
			if _, err := os.Stat(path); err == nil {
				if err := install.RestoreTypography(path); err != nil {
					return fmt.Errorf("restore typography: %w", err)
				}
				fmt.Println(ok, "Restored your own Bear typography")
			}
		}
	}
	fmt.Println()
	printNextSteps(c.RestoreSlot)
	return nil
}

func printNextSteps(slot string) {
	if install.BearRunning() {
		fmt.Println("Quit Bear (⌘Q) and reopen it to see", slot, "in Settings → Themes.")
	} else {
		fmt.Println("Open Bear to see", slot, "in Settings → Themes.")
	}
}

func installTheme(slot string, data []byte, fontPaths []string, typography []install.Pref) error {
	backup, created, err := install.Backup(slot)
	if err != nil {
		return fmt.Errorf("back up %s: %w", slot, err)
	}
	if created {
		fmt.Println(ok, "Backed up the original to", backup)
	} else {
		fmt.Println(ok, "Keeping the existing backup at", backup)
	}

	fmt.Println("bearty needs your password (sudo) to copy files into Bear.app.")
	for _, fontPath := range fontPaths {
		copied, err := install.Font(fontPath)
		if err != nil {
			return err
		}
		if copied {
			fmt.Println(ok, "Installed", filepath.Base(fontPath), "into Bear")
		}
	}
	if err := install.Theme(slot, data); err != nil {
		return err
	}
	fmt.Printf("%s Replaced %q\n", ok, slot)

	if len(typography) > 0 {
		path, err := library.TypographyBackupPath()
		if err != nil {
			return err
		}
		if created, err := install.BackupTypography(path); err != nil {
			return fmt.Errorf("back up Bear's typography: %w", err)
		} else if created {
			fmt.Println(ok, "Saved your Bear typography to", path)
		}
		if err := install.ApplyTypography(typography); err != nil {
			return err
		}
		fmt.Println(ok, "Set Bear's typography to match the theme")
	}
	fmt.Println()

	printNextSteps(slot)
	fmt.Println("A Bear update restores the original; run bearty again afterwards.")
	return nil
}
