package appos

import (
	"errors"
	"os/exec"
	"runtime"
	"time"
)

func OpenBrowser(url string, delay time.Duration) error {
	time.Sleep(delay)

	switch runtime.GOOS {
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		return exec.Command("open", url).Start()
	case "linux":
		fallthrough
	default:
		return errors.New("your OS is not supported")
	}
}
