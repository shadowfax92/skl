package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(initCmd)
}

var initCmd = &cobra.Command{
	Use:         "init [bash|zsh]",
	Annotations: map[string]string{"group": "Other:"},
	Short:       "Print shell integration",
	Long: `Print shell integration for commands that need to affect your shell.

Add this to your shell config:
  eval "$(skl init zsh)"`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		shell := ""
		if len(args) > 0 {
			shell = args[0]
		} else {
			shell = shellNameFromEnv(os.Getenv("SHELL"))
		}
		script, err := shellInitScript(shell)
		if err != nil {
			return err
		}
		fmt.Fprint(cmd.OutOrStdout(), script)
		return nil
	},
}

// shellInitScript prints the wrapper that lets skl commands affect the caller's shell.
func shellInitScript(shell string) (string, error) {
	switch strings.ToLower(shell) {
	case "bash", "zsh":
		return posixShellInit, nil
	case "":
		return "", fmt.Errorf("could not detect shell; pass bash or zsh")
	default:
		return "", fmt.Errorf("unsupported shell %q; supported shells: bash, zsh", shell)
	}
}

func shellNameFromEnv(shellPath string) string {
	if shellPath == "" {
		return ""
	}
	return filepath.Base(shellPath)
}

const posixShellInit = `skl() {
  if [ "$#" -gt 0 ] && [ "$1" = "cd" ]; then
    shift
    local dir code
    dir="$(command skl cd "$@")"
    code=$?
    if [ "$code" -ne 0 ]; then
      return "$code"
    fi
    if [ -t 1 ]; then
      builtin cd "$dir"
    else
      printf '%s\n' "$dir"
    fi
    return $?
  fi
  command skl "$@"
}
`
