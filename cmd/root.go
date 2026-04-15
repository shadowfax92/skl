package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"skl/internal/picker"
	"skl/internal/style"

	"github.com/spf13/cobra"
)

var Version = "dev"

var ErrCancelled = errors.New("")

func helpHeader(s string) string  { return style.Header(s) }
func helpCmdCol(s string) string  { return style.Cmd(s) }
func helpHint(s string) string    { return style.Hint(s) }
func helpAliases(a []string) string {
	if len(a) == 0 {
		return ""
	}
	return style.Aliases(a)
}

var groupOrder = []string{
	"Inspect:",
	"Load:",
	"Interactive:",
	"Library:",
	"Sync:",
	"Other:",
}

func groupedHelp(cmd *cobra.Command) string {
	groups := map[string][]*cobra.Command{}
	for _, c := range cmd.Commands() {
		if !c.IsAvailableCommand() && c.Name() != "help" {
			continue
		}
		g := c.Annotations["group"]
		if g == "" {
			g = "Other:"
		}
		groups[g] = append(groups[g], c)
	}

	var b strings.Builder
	for _, name := range groupOrder {
		cmds, ok := groups[name]
		if !ok {
			continue
		}
		b.WriteString("\n" + helpHeader(name) + "\n")
		for _, c := range cmds {
			line := "  " + helpCmdCol(fmt.Sprintf("%-10s", c.Name())) + " " + c.Short
			if len(c.Aliases) > 0 {
				line += " " + helpAliases(c.Aliases)
			}
			b.WriteString(line + "\n")
		}
	}
	return b.String()
}

const usageTemplate = `{{helpHeader "Usage:"}}{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

{{helpHeader "Aliases:"}}
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

{{helpHeader "Examples:"}}
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}
{{groupedHelp .}}{{end}}{{if .HasAvailableLocalFlags}}

{{helpHeader "Flags:"}}
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

{{helpHeader "Global Flags:"}}
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableSubCommands}}

{{helpHint (printf "Use \"%s [command] --help\" for more information." .CommandPath)}}{{end}}
`

var rootCmd = &cobra.Command{
	Use:   "skl",
	Short: "Manage Claude Code skills as named bundles",
	Long: `skl manages your ~/.skills/ directory by loading and unloading curated
bundles of skills from a source library at ~/.config/skl/library/.

Quick start:
  skl import              # seed library from current ~/.skills/
  skl board               # vim-style: drag skills between bundles in $EDITOR
  skl load dev            # copies dev's skills into ~/.skills/
  skl unload              # fzf-pick a loaded bundle to remove
  skl ls                  # show all bundles
  skl status              # show what's loaded right now
  skl sync                # git-sync the library`,
	Version:       Version,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

func init() {
	cobra.AddTemplateFunc("helpHeader", helpHeader)
	cobra.AddTemplateFunc("helpCmdCol", helpCmdCol)
	cobra.AddTemplateFunc("helpAliases", helpAliases)
	cobra.AddTemplateFunc("helpHint", helpHint)
	cobra.AddTemplateFunc("groupedHelp", groupedHelp)
	rootCmd.SetUsageTemplate(usageTemplate)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		if errors.Is(err, ErrCancelled) || errors.Is(err, picker.ErrCancelled) {
			os.Exit(0)
		}
		fmt.Fprintln(os.Stderr, "skl:", err)
		os.Exit(1)
	}
}
