package cmd

import (
	"reflect"
	"testing"

	"skl/internal/picker"
)

func TestPickBundleNamesUsesMultiSelectInSortedOrder(t *testing.T) {
	bundles := map[string][]string{
		"zeta": {"z"},
		"dev":  {"a", "b"},
	}

	var gotItems []picker.Item
	var gotOpts picker.Opts
	chosen, err := pickBundleNames(bundles, "show > ", func(items []picker.Item, opts picker.Opts) ([]string, error) {
		gotItems = items
		gotOpts = opts
		return []string{"dev", "zeta"}, nil
	})
	if err != nil {
		t.Fatalf("pickBundleNames: %v", err)
	}
	if !gotOpts.Multi {
		t.Fatalf("bundle picker should enable multi-select")
	}
	if gotOpts.Prompt != "show > " {
		t.Fatalf("prompt = %q, want %q", gotOpts.Prompt, "show > ")
	}
	gotIDs := []string{gotItems[0].ID, gotItems[1].ID}
	if want := []string{"dev", "zeta"}; !reflect.DeepEqual(gotIDs, want) {
		t.Fatalf("picker item order mismatch\ngot:  %#v\nwant: %#v", gotIDs, want)
	}
	if want := []string{"dev", "zeta"}; !reflect.DeepEqual(chosen, want) {
		t.Fatalf("chosen bundles mismatch\ngot:  %#v\nwant: %#v", chosen, want)
	}
}
