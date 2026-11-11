package install

import (
	"fmt"
	"os/exec"
	"time"
)

// BearBundleID identifies Bear to macOS.
const BearBundleID = "net.shinyfrog.bear"

// QuitBear asks Bear to quit the way ⌘Q does, so it saves its state, and
// waits for it to exit. macOS may ask once whether the terminal may control
// Bear.
func QuitBear() error {
	if !BearRunning() {
		return nil
	}
	out, err := exec.Command("osascript", "-e", `tell application id "`+BearBundleID+`" to quit`).CombinedOutput()
	if err != nil {
		return fmt.Errorf("couldn't quit Bear (%s); quit it with ⌘Q and try again", trim(out, err))
	}
	for range 50 { // up to 10 seconds
		if !BearRunning() {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("Bear didn't quit; quit it with ⌘Q and try again")
}

// OpenBear launches Bear.
func OpenBear() error {
	if out, err := exec.Command("open", "-b", BearBundleID).CombinedOutput(); err != nil {
		return fmt.Errorf("couldn't open Bear: %s", trim(out, err))
	}
	return nil
}

func trim(out []byte, err error) string {
	if s := string(out); len(s) > 0 {
		if len(s) > 200 {
			s = s[:200]
		}
		return s
	}
	return err.Error()
}
