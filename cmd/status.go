package cmd

import (
	"fmt"
	"sort"
	"strings"

	"skl/internal/live"
	"skl/internal/state"
	"skl/internal/style"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(statusCmd)
}

var statusCmd = &cobra.Command{
	Use:         "status",
	Aliases:     []string{"st"},
	Annotations: map[string]string{"group": "Inspect:"},
	Short:       "Show what's currently in ~/.skills/",
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := state.NewManager()
		if err != nil {
			return err
		}
		st, err := mgr.Load()
		if err != nil {
			return err
		}
		liveDirs, err := live.LoadedDirs()
		if err != nil {
			return err
		}

		liveSet := map[string]bool{}
		for _, d := range liveDirs {
			liveSet[d] = true
		}

		var managed []string
		var drift []string
		for id, entry := range st.Loaded {
			if liveSet[entry.DirName] {
				managed = append(managed, id)
			} else {
				drift = append(drift, id)
			}
		}

		stateDirs := map[string]bool{}
		for _, e := range st.Loaded {
			stateDirs[e.DirName] = true
		}
		var untracked []string
		for _, d := range liveDirs {
			if !stateDirs[d] {
				untracked = append(untracked, d)
			}
		}

		sort.Strings(managed)
		sort.Strings(drift)
		sort.Strings(untracked)

		printSection(style.Header("Loaded by skl:"), managed, func(id string) string {
			e := st.Loaded[id]
			bs := strings.Join(e.Bundles, ", ")
			age := style.Faint(state.RelativeTime(e.LoadedAt) + " ago")
			return fmt.Sprintf("  %s  %s  %s", id, style.Faint("["+bs+"]"), age)
		})
		printSection(style.Header("Untracked in ~/.skills/:"), untracked, func(d string) string {
			return "  " + d
		})
		printSection(style.Warn("Drift (state says loaded, missing on disk):"), drift, func(id string) string {
			return "  " + id
		})

		if len(managed) == 0 && len(untracked) == 0 && len(drift) == 0 {
			fmt.Println(style.Faint("~/.skills/ is empty (excluding system entries)."))
		}
		return nil
	},
}

func printSection(title string, items []string, fmtFn func(string) string) {
	if len(items) == 0 {
		return
	}
	fmt.Println(title)
	for _, it := range items {
		fmt.Println(fmtFn(it))
	}
	fmt.Println()
}
