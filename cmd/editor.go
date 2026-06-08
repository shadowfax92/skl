package cmd

import (
	"os"
	"os/exec"
)

func resolveEditor() string {
	if e := os.Getenv("EDITOR"); e != "" {
		return e
	}
	for _, c := range []string{"nvim", "vim", "vi"} {
		if _, err := exec.LookPath(c); err == nil {
			return c
		}
	}
	return "vi"
}
