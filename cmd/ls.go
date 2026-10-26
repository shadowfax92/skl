package cmd

import (
	"fmt"
	"sort"
	"strings"

	"skl/internal/library"
	"skl/internal/state"
	"skl/internal/style"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/spf13/cobra"
)

func init() {
	lsCmd.Flags().Bool("skills", false, "List all skills (instead of bundles)")
	rootCmd.AddCommand(lsCmd)
}

var lsCmd = &cobra.Command{
	Use:         "ls",
	Aliases:     []string{"list"},
	Annotations: map[string]string{"group": "Inspect:"},
	Short:       "List bundles (or `--skills` for individual skills)",
	RunE: func(cmd *cobra.Command, args []string) error {
		showSkills, _ := cmd.Flags().GetBool("skills")

		cat, err := library.ReadCatalog()
		if err != nil {
			return err
		}
		bundles, skills := cat.Bundles, cat.Skills
		mgr, err := state.NewManager()
		if err != nil {
			return err
		}
		st, err := mgr.Load()
		if err != nil {
			return err
		}

		if len(skills) == 0 && len(bundles) == 0 {
			fmt.Println(style.Faint("No skills in library yet."))
			fmt.Println(style.Faint(`Run "skl import" to seed from ~/.skills/, or`))
			fmt.Println(style.Faint(`"skl install <git-url>" to add a remote pack.`))
			return nil
		}

		if showSkills {
			return printSkills(skills, bundles, st)
		}
		return printBundles(cat, st)
	},
}

// printBundles lists each bundle with the number of skills `skl load` would
// install. When any bundle is recursive, an INCLUDES column names the nested
// bundles it pulls in; otherwise the table keeps its original three columns.
func printBundles(cat *library.Catalog, st *state.State) error {
	bundles := cat.Bundles
	loadedBundles := map[string]bool{}
	for _, b := range st.LoadedBundles() {
		loadedBundles[b] = true
	}

	names := make([]string, 0, len(bundles))
	for n := range bundles {
		names = append(names, n)
	}
	sort.Strings(names)

	if len(names) == 0 {
		fmt.Println(style.Faint("No bundles defined. Create one with `skl bundle create <name> <skill...>`."))
		return nil
	}

	headers := []string{"BUNDLE", "SKILLS", "STATUS"}
	showIncludes := len(cat.Recursive) > 0
	if showIncludes {
		headers = append(headers, "INCLUDES")
	}

	var rows [][]string
	for _, n := range names {
		mark := style.Faint("—")
		if loadedBundles[n] {
			mark = style.OK("loaded")
		}
		row := []string{n, fmt.Sprintf("%d", len(bundles[n])), mark}
		if showIncludes {
			includes := ""
			if cat.Recursive[n] {
				includes = strings.Join(cat.Nested(n), ", ")
			}
			row = append(row, includes)
		}
		rows = append(rows, row)
	}

	t := table.New().
		Border(lipgloss.HiddenBorder()).
		Headers(headers...).
		Rows(rows...).
		StyleFunc(func(row, col int) lipgloss.Style {
			s := lipgloss.NewStyle().PaddingRight(2)
			if row == table.HeaderRow {
				return s.Bold(true).Faint(true)
			}
			return s
		})

	fmt.Println(t)
	fmt.Println(style.Faint(loadedSummaryText(len(st.Loaded))))
	return nil
}

func printSkills(skills []library.Skill, bundles map[string][]string, st *state.State) error {
	skillToBundles := map[string][]string{}
	for b, ids := range bundles {
		for _, id := range ids {
			skillToBundles[id] = append(skillToBundles[id], b)
		}
	}

	var rows [][]string
	for _, s := range skills {
		bs := skillToBundles[s.ID]
		sort.Strings(bs)
		bsStr := strings.Join(bs, ", ")
		if bsStr == "" {
			bsStr = style.Faint(library.ReservedInboxBundle)
		}
		mark := style.Faint("—")
		if _, ok := st.Loaded[s.ID]; ok {
			mark = style.OK("loaded")
		}
		src := style.Faint("local")
		if s.External {
			src = style.Faint("ext: " + s.Repo)
		}
		rows = append(rows, []string{s.ID, bsStr, mark, src})
	}

	t := table.New().
		Border(lipgloss.HiddenBorder()).
		Headers("SKILL", "BUNDLES", "STATUS", "SOURCE").
		Rows(rows...).
		StyleFunc(func(row, col int) lipgloss.Style {
			s := lipgloss.NewStyle().PaddingRight(2)
			if row == table.HeaderRow {
				return s.Bold(true).Faint(true)
			}
			return s
		})

	fmt.Println(t)
	fmt.Println(style.Faint(loadedSummaryText(len(st.Loaded))))
	return nil
}

// loadedSummaryText renders the `skl ls` footer — the count of loaded skills, pluralized.
func loadedSummaryText(n int) string {
	noun := "skills"
	if n == 1 {
		noun = "skill"
	}
	return fmt.Sprintf("%d %s loaded", n, noun)
}
