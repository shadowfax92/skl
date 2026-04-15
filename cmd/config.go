package cmd

import (
	"fmt"

	"skl/internal/config"
	"skl/internal/library"
	"skl/internal/style"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(configCmd)
}

var configCmd = &cobra.Command{
	Use:         "config",
	Annotations: map[string]string{"group": "Other:"},
	Short:       "Show config and library paths",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfgPath, err := config.DefaultConfigPath()
		if err != nil {
			return err
		}
		libPath, err := library.LibraryPath()
		if err != nil {
			return err
		}
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		fmt.Println(style.Header("Paths:"))
		fmt.Printf("  config:  %s\n", cfgPath)
		fmt.Printf("  library: %s\n", libPath)
		fmt.Println()
		fmt.Println(style.Header("Config:"))
		if cfg.Sync.Remote == "" {
			fmt.Printf("  sync.remote:    %s\n", style.Faint("(unset — run `skl remote <url>`)"))
		} else {
			fmt.Printf("  sync.remote:    %s\n", cfg.Sync.Remote)
		}
		if len(cfg.DefaultBundles) == 0 {
			fmt.Printf("  default_bundles: %s\n", style.Faint("(none)"))
		} else {
			fmt.Printf("  default_bundles: %v\n", cfg.DefaultBundles)
		}
		return nil
	},
}
