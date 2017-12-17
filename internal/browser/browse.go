package browser

import (
	"errors"
	"os/exec"
	"runtime"
)

// Open opens the specified URL in the default browser
func Open(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
	case "linux", "freebsd", "openbsd":
		cmd = "xdg-open" // wherever the X server is used
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	default:
		return errors.New("platform not supported")
	}
	args = append(args, url)
	return exec.Command(cmd, args...).Start()
}
