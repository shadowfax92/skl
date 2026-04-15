package style

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	ClrCyan    = lipgloss.Color("6")
	ClrYellow  = lipgloss.Color("11")
	ClrGreen   = lipgloss.Color("2")
	ClrHiGreen = lipgloss.Color("10")
	ClrRed     = lipgloss.Color("9")
)

func Header(s string) string {
	return lipgloss.NewStyle().Bold(true).Foreground(ClrCyan).Render(s)
}

func Faint(s string) string {
	return lipgloss.NewStyle().Faint(true).Render(s)
}

func Cmd(s string) string {
	return lipgloss.NewStyle().Foreground(ClrHiGreen).Render(s)
}

func Hint(s string) string {
	return lipgloss.NewStyle().Faint(true).Render(s)
}

func OK(s string) string {
	return lipgloss.NewStyle().Foreground(ClrGreen).Bold(true).Render(s)
}

func Warn(s string) string {
	return lipgloss.NewStyle().Foreground(ClrYellow).Bold(true).Render(s)
}

func Err(s string) string {
	return lipgloss.NewStyle().Foreground(ClrRed).Bold(true).Render(s)
}

func Aliases(aliases []string) string {
	return lipgloss.NewStyle().Foreground(ClrYellow).
		Render(fmt.Sprintf("(aliases: %s)", strings.Join(aliases, ", ")))
}
