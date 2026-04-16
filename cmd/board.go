package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"

	"skl/internal/library"
	"skl/internal/style"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(boardCmd)
}

const boardUnbundled = "(unbundled)"

const boardHeader = `# skl board — edit and save to apply
#
# Each "### <name>" is a bundle. Skills go beneath as "- <skill-id>".
# Move skills between sections to reorganize bundles.
# Add a new bundle by adding a "### <name>" heading.
# Delete a bundle by removing its entire section.
# A skill can appear in multiple bundles (just list it under each).
# Skills under "### (unbundled)" stay in the library but join no bundle.
# Lines starting with "#" or blank lines are ignored.
# Quit without saving (e.g. ":cq" in vim) to abort all changes.
#

`

var boardCmd = &cobra.Command{
	Use:         "board",
	Aliases:     []string{"b"},
	Annotations: map[string]string{"group": "Interactive:"},
	Short:       "Edit bundles in $EDITOR (vim-style grouping)",
	RunE: func(cmd *cobra.Command, args []string) error {
		skills, err := library.Skills()
		if err != nil {
			return err
		}
		if len(skills) == 0 {
			fmt.Println(style.Faint(`No skills in library. Run "skl import" first.`))
			return nil
		}

		oldBundles, err := library.Bundles()
		if err != nil {
			return err
		}

		original := generateBoardMarkdown(skills, oldBundles)
		edited, err := openInEditor("skl-board", original)
		if err != nil {
			return err
		}
		if edited == original {
			fmt.Println(style.Faint("No changes."))
			return nil
		}

		newBundles, err := parseBoardMarkdown(edited)
		if err != nil {
			return err
		}
		if err := validateBoardSkills(newBundles, skills); err != nil {
			return err
		}

		summary := diffBundles(oldBundles, newBundles)
		if err := library.WriteBundles(newBundles); err != nil {
			return err
		}
		printBoardSummary(summary)
		return nil
	},
}

type boardSummary struct {
	bundlesCreated []string
	bundlesDeleted []string
	skillsAdded    int
	skillsRemoved  int
}

func generateBoardMarkdown(skills []library.Skill, bundles map[string][]string) string {
	var b strings.Builder
	b.WriteString(boardHeader)

	names := make([]string, 0, len(bundles))
	for n := range bundles {
		if n == boardUnbundled {
			continue
		}
		names = append(names, n)
	}
	sort.Strings(names)

	bundled := map[string]bool{}
	for _, n := range names {
		ids := append([]string(nil), bundles[n]...)
		sort.Strings(ids)
		b.WriteString("### " + n + "\n")
		for _, id := range ids {
			b.WriteString("- " + id + "\n")
			bundled[id] = true
		}
		b.WriteString("\n")
	}

	var unbundled []string
	for _, s := range skills {
		if !bundled[s.ID] {
			unbundled = append(unbundled, s.ID)
		}
	}
	sort.Strings(unbundled)
	b.WriteString("### " + boardUnbundled + "\n")
	for _, id := range unbundled {
		b.WriteString("- " + id + "\n")
	}
	return b.String()
}

func parseBoardMarkdown(content string) (map[string][]string, error) {
	out := map[string][]string{}
	current := ""
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "### ") {
			current = strings.TrimSpace(strings.TrimPrefix(trimmed, "### "))
			if current != boardUnbundled {
				if _, ok := out[current]; !ok {
					out[current] = []string{}
				}
			}
			continue
		}
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		if !strings.HasPrefix(trimmed, "-") && !strings.HasPrefix(trimmed, "*") {
			continue
		}
		id := strings.TrimSpace(trimmed[1:])
		if id == "" || current == "" || current == boardUnbundled {
			continue
		}
		out[current] = append(out[current], id)
	}
	return out, nil
}

func validateBoardSkills(bundles map[string][]string, lib []library.Skill) error {
	libIDs := make(map[string]bool, len(lib))
	for _, s := range lib {
		libIDs[s.ID] = true
	}
	for name, ids := range bundles {
		seen := map[string]bool{}
		for _, id := range ids {
			if !libIDs[id] {
				return fmt.Errorf("bundle %q references unknown skill %q", name, id)
			}
			if seen[id] {
				return fmt.Errorf("bundle %q lists skill %q twice", name, id)
			}
			seen[id] = true
		}
	}
	return nil
}

func diffBundles(oldB, newB map[string][]string) boardSummary {
	var s boardSummary
	for name := range newB {
		if _, ok := oldB[name]; !ok {
			s.bundlesCreated = append(s.bundlesCreated, name)
		}
	}
	for name := range oldB {
		if _, ok := newB[name]; !ok {
			s.bundlesDeleted = append(s.bundlesDeleted, name)
		}
	}
	sort.Strings(s.bundlesCreated)
	sort.Strings(s.bundlesDeleted)

	for name, ids := range newB {
		oldSet := setOf(oldB[name])
		for _, id := range ids {
			if !oldSet[id] {
				s.skillsAdded++
			}
		}
	}
	for name, ids := range oldB {
		newSet := setOf(newB[name])
		for _, id := range ids {
			if !newSet[id] {
				s.skillsRemoved++
			}
		}
	}
	return s
}

func setOf(in []string) map[string]bool {
	out := make(map[string]bool, len(in))
	for _, s := range in {
		out[s] = true
	}
	return out
}

func printBoardSummary(s boardSummary) {
	if len(s.bundlesCreated) == 0 && len(s.bundlesDeleted) == 0 && s.skillsAdded == 0 && s.skillsRemoved == 0 {
		fmt.Println(style.Faint("No effective changes."))
		return
	}
	if len(s.bundlesCreated) > 0 {
		fmt.Printf("%s %s\n", style.OK("created bundle(s):"), strings.Join(s.bundlesCreated, ", "))
	}
	if len(s.bundlesDeleted) > 0 {
		fmt.Printf("%s %s\n", style.OK("deleted bundle(s):"), strings.Join(s.bundlesDeleted, ", "))
	}
	if s.skillsAdded > 0 {
		fmt.Printf("%s %d skill assignment(s)\n", style.OK("added"), s.skillsAdded)
	}
	if s.skillsRemoved > 0 {
		fmt.Printf("%s %d skill assignment(s)\n", style.OK("removed"), s.skillsRemoved)
	}
}

func openInEditor(prefix, content string) (string, error) {
	f, err := os.CreateTemp("", prefix+"-*.md")
	if err != nil {
		return "", err
	}
	name := f.Name()
	defer os.Remove(name)
	if _, err := f.WriteString(content); err != nil {
		f.Close()
		return "", err
	}
	if err := f.Close(); err != nil {
		return "", err
	}

	editor := resolveEditor()
	cmd := exec.Command(editor, name)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("editor %q exited with error: %w", editor, err)
	}

	edited, err := os.ReadFile(name)
	if err != nil {
		return "", err
	}
	return string(edited), nil
}

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
