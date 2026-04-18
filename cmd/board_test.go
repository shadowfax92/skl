package cmd

import (
	"reflect"
	"strings"
	"testing"

	"skl/internal/library"
)

func TestBoardKeepsInboxDerived(t *testing.T) {
	skills := []library.Skill{
		{ID: "alpha"},
		{ID: "beta"},
	}
	bundles := map[string][]string{
		"dev":                       {"alpha"},
		library.ReservedInboxBundle: {"beta"},
	}

	markdown := generateBoardMarkdown(skills, bundles)
	if !strings.Contains(markdown, "### "+library.ReservedInboxBundle) {
		t.Fatalf("board markdown should include %q section:\n%s", library.ReservedInboxBundle, markdown)
	}

	parsed, err := parseBoardMarkdown(markdown)
	if err != nil {
		t.Fatalf("parseBoardMarkdown: %v", err)
	}
	want := map[string][]string{"dev": {"alpha"}}
	if !reflect.DeepEqual(parsed, want) {
		t.Fatalf("parsed bundles mismatch\ngot:  %#v\nwant: %#v", parsed, want)
	}

	stripped := stripReservedBundles(bundles)
	if _, ok := stripped[library.ReservedInboxBundle]; ok {
		t.Fatalf("stripReservedBundles should remove %q", library.ReservedInboxBundle)
	}
}
