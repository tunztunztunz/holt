package cli

import (
	"testing"

	"github.com/tunztunztunz/holt/internal/state"
)

func TestBaseForSelection(t *testing.T) {
	t.Run("override always wins, even over recorded bases", func(t *testing.T) {
		recs := []*state.Record{{BaseBranch: "main"}, {BaseBranch: "develop"}}
		got, err := baseForSelection(recs, "release")
		if err != nil {
			t.Fatalf("baseForSelection: %v", err)
		}
		if got != "release" {
			t.Errorf("got %q, want %q", got, "release")
		}
	})

	t.Run("a single shared base is used", func(t *testing.T) {
		recs := []*state.Record{{BaseBranch: "main"}, {BaseBranch: "main"}}
		got, err := baseForSelection(recs, "")
		if err != nil {
			t.Fatalf("baseForSelection: %v", err)
		}
		if got != "main" {
			t.Errorf("got %q, want %q", got, "main")
		}
	})

	t.Run("no recorded base refuses", func(t *testing.T) {
		recs := []*state.Record{{BaseBranch: ""}, {BaseBranch: ""}}
		if _, err := baseForSelection(recs, ""); err == nil {
			t.Fatal("want error when nothing records a base, got nil")
		}
	})

	t.Run("divergent bases refuse rather than guess", func(t *testing.T) {
		recs := []*state.Record{{BaseBranch: "main"}, {BaseBranch: "develop"}}
		if _, err := baseForSelection(recs, ""); err == nil {
			t.Fatal("want error for divergent bases, got nil")
		}
	})
}
