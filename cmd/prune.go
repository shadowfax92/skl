package cmd

import (
	"fmt"
	"sort"
	"strings"

	"skl/internal/library"
	"skl/internal/live"
	"skl/internal/picker"
	"skl/internal/state"
	"skl/internal/style"

	"github.com/spf13/cobra"
)

func init() {
	pruneCmd.Flags().Bool("all", false, "Remove every skill dir in ~/.skills/")
	pruneCmd.Flags().Bool("untracked", false, "Remove only skills not loaded by skl")
	pruneCmd.MarkFlagsMutuallyExclusive("all", "untracked")
	rootCmd.AddCommand(pruneCmd)
}

var pruneCmd = &cobra.Command{
	Use:         "prune [skill...]",
	Aliases:     []string{"rm"},
	Annotations: map[string]string{"group": "Load:"},
	Short:       "Remove skills from ~/.skills/ (tracked or untracked)",
	Long: `Remove skill directories from ~/.skills/, regardless of whether skl
loaded them.

  skl prune              fzf multi-pick over every skill in ~/.skills/
  skl prune foo bar      remove specific skills by dir name
  skl prune --all        remove every skill
  skl prune --untracked  remove only skills not loaded by skl (the usual
                         post-import cleanup — wipes old manually-added
                         skills that predate your curated bundles)

State is updated for any pruned skills that skl had loaded.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		all, _ := cmd.Flags().GetBool("all")
		untrackedOnly, _ := cmd.Flags().GetBool("untracked")

		mgr, err := state.NewManager()
		if err != nil {
			return err
		}
		if err := mgr.Lock(); err != nil {
			return err
		}
		defer mgr.Unlock()

		st, err := mgr.Load()
		if err != nil {
			return err
		}

		dirs, err := live.LoadedDirs()
		if err != nil {
			return err
		}
		if len(dirs) == 0 {
			fmt.Println(style.Faint("~/.skills/ is empty."))
			return nil
		}

		stateByDir := indexStateByDir(st)

		var targets []string
		switch {
		case all:
			targets = dirs
		case untrackedOnly:
			for _, d := range dirs {
				if _, ok := stateByDir[d]; !ok {
					targets = append(targets, d)
				}
			}
		case len(args) > 0:
			targets = args
		default:
			targets, err = pickSkillsToPrune(dirs, stateByDir, st, library.Bundles)
			if err != nil {
				return err
			}
		}

		if len(targets) == 0 {
			fmt.Println(style.Faint("Nothing to prune."))
			return nil
		}

		managed, untracked := 0, 0
		for _, d := range targets {
			if err := live.RemoveSkill(d); err != nil {
				return fmt.Errorf("removing %s: %w", d, err)
			}
			if id, ok := stateByDir[d]; ok {
				st.RemoveLoaded(id)
				managed++
			} else {
				untracked++
			}
		}

		fmt.Printf("%s %d skill(s)  %s %d untracked, %d managed\n",
			style.OK("pruned"), len(targets),
			style.Faint("="), untracked, managed)
		return mgr.Save(st)
	},
}

func indexStateByDir(st *state.State) map[string]string {
	out := make(map[string]string, len(st.Loaded))
	for id, entry := range st.Loaded {
		out[entry.DirName] = id
	}
	return out
}

func pickSkillsToPrune(
	dirs []string,
	stateByDir map[string]string,
	st *state.State,
	bundlesFn func() (map[string][]string, error),
) ([]string, error) {
	bundles, err := bundlesFn()
	if err != nil {
		return nil, err
	}
	skillToBundles := map[string][]string{}
	for b, ids := range bundles {
		for _, id := range ids {
			skillToBundles[id] = append(skillToBundles[id], b)
		}
	}

	items := make([]picker.Item, 0, len(dirs))
	for _, d := range dirs {
		note := pruneLabelFor(d, stateByDir, st, skillToBundles)
		items = append(items, picker.Item{
			ID:      d,
			Display: fmt.Sprintf("%-40s  %s", d, note),
		})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })

	chosen, err := picker.Pick(items, picker.Opts{Prompt: "prune > ", Multi: true})
	if err != nil {
		return nil, err
	}
	if len(chosen) == 0 {
		return nil, ErrCancelled
	}
	return chosen, nil
}

func pruneLabelFor(dir string, stateByDir map[string]string, st *state.State, skillToBundles map[string][]string) string {
	if id, ok := stateByDir[dir]; ok {
		bs := strings.Join(st.Loaded[id].Bundles, ",")
		return fmt.Sprintf("loaded via %s", bs)
	}
	memberships := skillToBundles[dir]
	sort.Strings(memberships)
	switch len(memberships) {
	case 0:
		return "untracked (not in library)"
	case 1:
		return fmt.Sprintf("untracked (in bundle: %s)", memberships[0])
	default:
		return fmt.Sprintf("untracked (in bundles: %s)", strings.Join(memberships, ", "))
	}
}
