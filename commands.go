package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/manojsachwani/bearty/internal/capture"
	"github.com/manojsachwani/bearty/internal/fonts"
	"github.com/manojsachwani/bearty/internal/install"
	"github.com/manojsachwani/bearty/internal/library"
)

const usage = `bearty: custom themes for the Bear notes app

Usage:
  bearty                          open the interactive editor
  bearty list                     list your themes and what's installed
  bearty slots                    list Bear's themes you can replace
  bearty install <theme> [flags]  install a saved theme into Bear
      --slot <name>               Bear theme to replace (default: the last one used)
      --dark-slot <name>          Bear theme for the dark version, if it has one
      --no-typography             leave Bear's own typography settings alone
      --no-restart                don't quit and reopen Bear
      --export <file>             also write the .theme file
  bearty use <theme> [--no-restart]
                                  switch Bear to an installed theme, fonts included
  bearty export <theme> [-o file] write a saved theme as a .theme file
  bearty capture <bear theme> [--name <name>]
                                  turn one of Bear's themes into a bearty theme
  bearty import <file.theme> [--name <name>]
                                  turn a .theme file into a bearty theme
  bearty reapply [--no-restart]   put back the last install after a Bear update
  bearty restore <bear theme> [--remove-fonts] [--keep-settings]
                                  put Bear's original theme back
  bearty help | version

Interactive flags:
  --no-motion                     skip the splash screen and animations
  --version                       print the version
`

// parse reads flags that may come before or after the positional arguments.
func parse(fs *flag.FlagSet, args []string) ([]string, error) {
	var pos []string
	for {
		if err := fs.Parse(args); err != nil {
			return nil, err
		}
		if fs.NArg() == 0 {
			return pos, nil
		}
		pos = append(pos, fs.Arg(0))
		args = fs.Args()[1:]
	}
}

func runCommand(name string, args []string) error {
	fs := flag.NewFlagSet("bearty "+name, flag.ContinueOnError)
	fs.Usage = func() { fmt.Fprint(os.Stderr, usage) }

	switch name {
	case "help", "-h", "--help":
		fmt.Print(usage)
		return nil
	case "version":
		fmt.Println("bearty", version)
		return nil

	case "list":
		return list()

	case "slots":
		slots, err := install.Slots()
		if err != nil {
			return err
		}
		in, haveRecord := install.LoadInstalled(installedPath())
		for _, s := range slots {
			mark := ""
			switch {
			case haveRecord && in.Light.Slot == s:
				mark = "  ← " + in.Theme
			case haveRecord && in.Dark != nil && in.Dark.Slot == s:
				mark = "  ← " + in.Theme + " (dark)"
			}
			fmt.Println(s + mark)
		}
		return nil

	case "install":
		slot := fs.String("slot", "", "Bear theme to replace")
		darkSlot := fs.String("dark-slot", "", "Bear theme for the dark version")
		noTypography := fs.Bool("no-typography", false, "leave Bear's typography alone")
		noRestart := fs.Bool("no-restart", false, "don't quit and reopen Bear")
		exportPath := fs.String("export", "", "also write the .theme file")
		pos, err := parse(fs, args)
		if err != nil {
			return err
		}
		entry, err := theme1(pos)
		if err != nil {
			return err
		}
		if entry.Slot, err = pickSlot(*slot, entry.Slot, "--slot"); err != nil {
			return err
		}
		if entry.Dark.Enabled() {
			if entry.DarkSlot, err = pickSlot(*darkSlot, entry.DarkSlot, "--dark-slot"); err != nil {
				return err
			}
			if entry.DarkSlot == entry.Slot {
				return fmt.Errorf("the dark version needs its own slot; choose a different --dark-slot")
			}
		}
		faces := fonts.Installed(fonts.Dirs(), install.FontsDir())
		return apply(request{Entry: entry, Install: true, Export: *exportPath,
			SetTypography: !*noTypography, Restart: !*noRestart}, faces)

	case "use":
		noRestart := fs.Bool("no-restart", false, "don't quit and reopen Bear")
		pos, err := parse(fs, args)
		if err != nil {
			return err
		}
		entry, err := theme1(pos)
		if err != nil {
			return err
		}
		return useTheme(entry, fonts.Installed(fonts.Dirs(), install.FontsDir()), !*noRestart)

	case "export":
		out := fs.String("o", "", "file to write (default ./<theme>.theme)")
		pos, err := parse(fs, args)
		if err != nil {
			return err
		}
		entry, err := theme1(pos)
		if err != nil {
			return err
		}
		if *out == "" {
			*out = "./" + library.Slug(entry.Name) + ".theme"
		}
		return apply(request{Entry: entry, Export: *out}, fonts.Installed(fonts.Dirs(), install.FontsDir()))

	case "capture", "import":
		nameFlag := fs.String("name", "", "name for the new theme")
		pos, err := parse(fs, args)
		if err != nil {
			return err
		}
		if len(pos) != 1 {
			fs.Usage()
			return fmt.Errorf("%s needs one argument", name)
		}
		path := pos[0]
		if name == "capture" {
			slot, err := pickSlot(pos[0], "", "")
			if err != nil {
				return err
			}
			path = filepath.Join(install.ThemesDir(), slot+".theme")
			pos[0] = slot
		}
		themeName := *nameFlag
		if themeName == "" {
			themeName = strings.TrimSuffix(filepath.Base(pos[0]), ".theme")
		}
		if _, exists := findSaved(themeName); exists {
			return fmt.Errorf("you already have a theme called %q; pick another with --name", themeName)
		}
		s, err := capture.File(path, install.ThemesDir(), themeName)
		if err != nil {
			return err
		}
		saved, err := library.Save(s)
		if err != nil {
			return err
		}
		fmt.Println(ok, "Saved", themeName, "to", saved)
		if len(s.Extra) > 0 {
			fmt.Printf("  %d settings bearty doesn't edit were kept as they are, so it matches the original exactly.\n", len(s.Extra))
		}
		fmt.Printf("Edit it in bearty, or install it with: bearty install %q --slot <bear theme>\n", themeName)
		return nil

	case "reapply":
		noRestart := fs.Bool("no-restart", false, "don't quit and reopen Bear")
		if _, err := parse(fs, args); err != nil {
			return err
		}
		return reapply(!*noRestart)

	case "restore":
		removeFonts := fs.Bool("remove-fonts", false, "also remove fonts bearty added to Bear")
		keepSettings := fs.Bool("keep-settings", false, "keep Bear's typography and theme choice as they are")
		pos, err := parse(fs, args)
		if err != nil {
			return err
		}
		if len(pos) != 1 {
			return fmt.Errorf("restore needs the Bear theme to restore, e.g. bearty restore Notes")
		}
		slot, err := pickSlot(pos[0], "", "")
		if err != nil {
			return err
		}
		if !contains(install.Backups([]string{slot}), slot) {
			return fmt.Errorf("bearty has no backup of %q, so there's nothing to restore", slot)
		}
		return restore(slot, *removeFonts, !*keepSettings)
	}
	fmt.Fprint(os.Stderr, usage)
	return fmt.Errorf("unknown command %q", name)
}

// theme1 finds the saved theme named by the one positional argument.
func theme1(pos []string) (library.Saved, error) {
	if len(pos) != 1 {
		return library.Saved{}, fmt.Errorf("name one theme (quote names with spaces); see `bearty list`")
	}
	s, ok := findSaved(pos[0])
	if !ok {
		return s, fmt.Errorf("no theme called %q; see `bearty list`", pos[0])
	}
	return s, nil
}

// pickSlot matches a Bear theme name, case-insensitively. An empty choice
// falls back to the theme's last slot.
func pickSlot(choice, fallback, flagName string) (string, error) {
	slots, err := install.Slots()
	if err != nil {
		return "", err
	}
	if choice == "" {
		choice = fallback
	}
	if choice == "" {
		return "", fmt.Errorf("choose which Bear theme to replace with %s; see `bearty slots`", flagName)
	}
	for _, s := range slots {
		if strings.EqualFold(s, choice) {
			return s, nil
		}
	}
	return "", fmt.Errorf("Bear has no theme called %q; see `bearty slots`", choice)
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

func list() error {
	saved, err := library.List()
	if err != nil {
		fmt.Fprintln(os.Stderr, warn, err)
	}
	in, haveRecord := install.LoadInstalled(installedPath())
	for _, s := range saved {
		var tags []string
		if s.Builtin {
			tags = append(tags, "built-in")
		}
		if s.Dark.Enabled() {
			tags = append(tags, "has dark version")
		}
		if haveRecord && library.Slug(in.Theme) == library.Slug(s.Name) {
			where := in.Light.Slot
			if in.Dark != nil {
				where += ", dark in " + in.Dark.Slot
			}
			tags = append(tags, "installed in "+where)
		}
		line := fmt.Sprintf("%-22s paper %s  ink %s  %s %dpt", s.Name, s.Paper, s.Ink, s.Font, s.Size)
		if len(tags) > 0 {
			line += "  (" + strings.Join(tags, "; ") + ")"
		}
		fmt.Println(line)
	}
	if haveRecord && in.Undone() {
		fmt.Println()
		fmt.Println(warn, "A Bear update removed", in.Theme+". Run `bearty reapply` to put it back.")
	}
	return nil
}
