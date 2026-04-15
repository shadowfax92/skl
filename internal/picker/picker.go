package picker

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

var ErrCancelled = errors.New("picker cancelled")

type Item struct {
	ID      string
	Display string
}

type Opts struct {
	Prompt string
	Multi  bool
	Header string
}

func Pick(items []Item, opts Opts) ([]string, error) {
	if len(items) == 0 {
		return nil, errors.New("nothing to pick")
	}
	if _, err := exec.LookPath("fzf"); err != nil {
		return nil, fmt.Errorf("fzf not installed; install with `brew install fzf` or pass arguments explicitly")
	}

	var lines []string
	for _, it := range items {
		display := it.Display
		if display == "" {
			display = it.ID
		}
		lines = append(lines, it.ID+"\t"+display)
	}

	args := []string{
		"--height", "100%", "--reverse",
		"--delimiter", "\t", "--with-nth", "2",
	}
	if opts.Prompt != "" {
		args = append(args, "--prompt", opts.Prompt)
	}
	if opts.Header != "" {
		args = append(args, "--header", opts.Header)
	}
	if opts.Multi {
		args = append(args, "--multi")
	}

	cmd := exec.Command("fzf", args...)
	cmd.Stdin = strings.NewReader(strings.Join(lines, "\n"))
	cmd.Stderr = os.Stderr

	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 130 {
			return nil, ErrCancelled
		}
		return nil, fmt.Errorf("fzf failed: %w", err)
	}

	var picked []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if idx := strings.Index(line, "\t"); idx >= 0 {
			picked = append(picked, line[:idx])
		} else {
			picked = append(picked, line)
		}
	}
	return picked, nil
}
