package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"skl/internal/library"
	"skl/internal/picker"
	"skl/internal/style"

	"github.com/spf13/cobra"
)

var Version = "dev"

var ErrCancelled = errors.New("")

var rootLLMTxt bool

func helpHeader(s string) string { return style.Header(s) }
func helpCmdCol(s string) string { return style.Cmd(s) }
func helpHint(s string) string   { return style.Hint(s) }
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
	Short: "Manage Claude Code skills as folder bundles",
	Long: `Folder-based skill loadouts for ~/.skills/.

Try:
  skl ls
  skl load <bundle>
  skl status`,
	Version:       Version,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if rootLLMTxt {
			out, err := llmTxt()
			if err != nil {
				return err
			}
			fmt.Fprint(cmd.OutOrStdout(), out)
			return nil
		}
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
	rootCmd.Flags().BoolVar(&rootLLMTxt, "llm-txt", false, "Print simple instructions for coding agents")
}

// llmTxt prints a compact, machine-friendly guide to the skl library layout.
// It uses the resolved library path so an agent can operate on the right
// directory without guessing where the user's skills live.
func llmTxt() (string, error) {
	lib, err := library.LibraryPath()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(`SKL LLM GUIDE

Library: %s
Live skills dir: ~/.skills

How skl works:
- The library is the source of truth.
- skl load <bundle> copies skill folders from the library into ~/.skills.
- skl unload <bundle> removes skills that no loaded bundle still claims.

How to organize skills:
- Folder bundles live directly under the library.
- A skill is any directory containing SKILL.md.
- Put bundled skills at <bundle>/<skill>/SKILL.md.
- Example: dev/cso/SKILL.md is loaded by: skl load dev
- legacy unbundled skills live at skills/<skill>/SKILL.md.

External repos:
- Put third-party repos under external/<repo>/.
- Put or keep skills at external/<repo>/<skill>/SKILL.md.
- Example: external/gstack/agent/SKILL.md is loaded by: skl load external/gstack
- Nested .git directories are intentionally ignored by skl sync.
- To update an external repo, run git commands inside external/<repo>/.

Useful commands:
- skl ls                  list folder bundles
- skl ls --skills         list every skill ID
- skl load <bundle>       load a folder bundle
- skl unload <bundle>     unload a folder bundle
- skl status              show loaded skills
- skl config              show paths
`, lib), nil
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
