package gui

import (
	"fmt"
	"os/exec"
	"runtime"
)

// OpenBrowser opens the given URL in the user's default browser.
func OpenBrowser(url string) error {
	name, args := browserCommand(url)
	path, err := exec.LookPath(name)
	if err != nil {
		return fmt.Errorf("browser command %q not found: %w", name, err)
	}
	return exec.Command(path, args...).Start()
}

// browserCommand returns the command name and arguments for opening a URL.
func browserCommand(url string) (string, []string) {
	switch runtime.GOOS {
	case "darwin":
		return "open", []string{url}
	case "windows":
		return "cmd", []string{"/c", "start", url}
	default:
		return "xdg-open", []string{url}
	}
}
