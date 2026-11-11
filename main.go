// Command bearty builds custom themes for the Bear notes app and can
// install them into one of Bear's built-in theme slots.
//
// Run it with no arguments for the interactive editor, or with a command
// (see `bearty help`) to script it.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/manojsachwani/bearty/internal/fonts"
	"github.com/manojsachwani/bearty/internal/install"
	"github.com/manojsachwani/bearty/internal/library"
	"github.com/manojsachwani/bearty/internal/theme"
	"github.com/manojsachwani/bearty/internal/ui"
)

var version = "dev"

// ok and warn mark output lines, in colour only when printing to a terminal.
var ok, warn = "✓", "!"

func init() {
	if ui.Interactive() {
		ok = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Inks[0].Hex)).Render(ok)
		warn = lipgloss.NewStyle().Foreground(lipgloss.Color("#E5B92E")).Render(warn)
	}
}

func main() {
	var err error
	if len(os.Args) > 1 && !strings.HasPrefix(os.Args[1], "-") {
		err = runCommand(os.Args[1], os.Args[2:])
	} else {
		showVersion := flag.Bool("version", false, "print the version and exit")
		noMotion := flag.Bool("no-motion", os.Getenv("BEARTY_NO_MOTION") != "", "skip the splash screen and animations (or set BEARTY_NO_MOTION)")
		flag.Usage = func() { fmt.Fprint(os.Stderr, usage) }
		flag.Parse()
		if *showVersion {
			fmt.Println("bearty", version)
			return
		}
		err = runTUI(*noMotion)
	}
	if err != nil {
		if errors.Is(err, ui.ErrAborted) {
			return
		}
		fmt.Fprintln(os.Stderr, "bearty:", err)
		os.Exit(1)
	}
}

func runTUI(noMotion bool) error {
	slots, _ := install.Slots() // no slots just means Bear isn't installed
	saved, libErr := library.List()
	if libErr != nil {
		fmt.Fprintln(os.Stderr, warn, libErr)
	}
	env := ui.Env{
		Slots:            slots,
		Backups:          install.Backups(slots),
		AddedFonts:       install.AddedFonts(),
		TypographyBackup: typographyBackupExists(),
		Saved:            saved,
		BearRunning:      install.BearRunning(),
		ThemesDir:        install.ThemesDir(),
		Version:          version,
		NoMotion:         noMotion,
	}
	if in, ok := install.LoadInstalled(installedPath()); ok && in.Undone() {
		env.Reapply = in.Theme
	}
	var faces []fonts.Face
	env.LoadFaces = func() []fonts.Face {
		faces = fonts.Installed(fonts.Dirs(), install.FontsDir())
		return faces
	}

	c, err := ui.Run(env)
	if err != nil {
		return err
	}
	switch {
	case c.Restoring():
		return restore(c.RestoreSlot, c.RemoveFonts, c.RestoreTypography)
	case c.Reapplying():
		return reapply(c.Restart)
	case c.Using():
		for _, s := range saved {
			if library.Slug(s.Name) == c.UseSel {
				return useTheme(s, fonts.Installed(fonts.Dirs(), install.FontsDir()), c.Restart)
			}
		}
		return fmt.Errorf("no saved theme to switch to")
	}
	if faces == nil {
		faces = fonts.Installed(fonts.Dirs(), install.FontsDir())
	}

	entry := c.Saved()
	entry.Slot, entry.DarkSlot = c.Slot, c.DarkSlot
	r := request{
		Entry:         entry,
		Install:       c.Installing() && c.Confirm,
		SetTypography: c.SetTypography,
		Restart:       c.Restart,
	}
	if c.Exporting() {
		r.Export = c.SavePath
	}
	if c.Installing() && !c.Confirm {
		fmt.Println(warn, "Install cancelled; Bear was not changed.")
	}
	return apply(r, faces)
}

func typographyBackupExists() bool {
	path, err := library.TypographyBackupPath()
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}

// restore puts Bear's original theme back in slot and, if asked, removes
// the fonts bearty added and puts the user's own Bear settings back.
func restore(slot string, removeFonts, restoreSettings bool) error {
	fmt.Println("bearty needs your password (sudo) to copy files into Bear.app.")
	if err := install.Restore(slot); err != nil {
		return err
	}
	fmt.Printf("%s Restored Bear's original %q\n", ok, slot)
	if added := install.AddedFonts(); removeFonts && len(added) > 0 {
		if err := install.RemoveFonts(added); err != nil {
			return fmt.Errorf("remove fonts: %w", err)
		}
		fmt.Println(ok, "Removed", strings.Join(added, ", "), "from Bear")
	}
	if restoreSettings {
		if path, err := library.TypographyBackupPath(); err == nil {
			if _, err := os.Stat(path); err == nil {
				if err := install.RestoreTypography(path); err != nil {
					return fmt.Errorf("restore Bear's settings: %w", err)
				}
				fmt.Println(ok, "Restored your own Bear settings (typography and theme choice)")
			}
		}
	}
	// bearty no longer has anything installed there to re-apply.
	if in, ok := install.LoadInstalled(installedPath()); ok &&
		(in.Light.Slot == slot || (in.Dark != nil && in.Dark.Slot == slot)) {
		os.Remove(installedPath())
	}
	fmt.Println()
	if install.BearRunning() {
		fmt.Println("Quit Bear (⌘Q) and reopen it to see the change.")
	}
	return nil
}
