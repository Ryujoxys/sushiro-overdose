//go:build desktop && darwin

package app

import "os/exec"

func showDesktopError(message string) {
	_ = exec.Command("osascript", "-e", `on run argv
display alert "无法打开寿司郎" message (item 1 of argv)
end run`, "--", message).Run()
}
