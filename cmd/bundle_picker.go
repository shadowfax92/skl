package cmd

import (
	"fmt"
	"sort"

	"skl/internal/picker"
)

type pickItemsFunc func([]picker.Item, picker.Opts) ([]string, error)

// pickBundles prompts for one or more bundle names using the shared bundle picker.
func pickBundles(bundles map[string][]string, prompt string) ([]string, error) {
	return pickBundleNames(bundles, prompt, picker.Pick)
}

func pickBundleNames(bundles map[string][]string, prompt string, pick pickItemsFunc) ([]string, error) {
	if len(bundles) == 0 {
		return nil, fmt.Errorf("no bundles defined")
	}

	names := make([]string, 0, len(bundles))
	for name := range bundles {
		names = append(names, name)
	}
	sort.Strings(names)

	items := make([]picker.Item, 0, len(names))
	for _, name := range names {
		items = append(items, picker.Item{
			ID:      name,
			Display: fmt.Sprintf("%s\t(%d skills)", name, len(bundles[name])),
		})
	}

	chosen, err := pick(items, picker.Opts{Prompt: prompt, Multi: true})
	if err != nil {
		return nil, err
	}
	if len(chosen) == 0 {
		return nil, ErrCancelled
	}
	return chosen, nil
}
