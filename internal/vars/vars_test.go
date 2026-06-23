package vars

import (
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/tunztunztunz/holt/internal/config"
)

func TestResolve(t *testing.T) {
	t.Run("defaults expand to project-tree slug", func(t *testing.T) {
		p := &config.Profile{SiteName: "$PROJECT-$TREE", WorktreesDir: ".."}

		got, err := Resolve("/x/myrepo", "feature/foo", p)
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}

		want := &Vars{
			RepoRoot: "/x/myrepo",
			Project:  "myrepo",
			Branch:   "feature/foo",
			Tree:     "feature-foo", // slashes become dashes
			SiteName: "myrepo-feature-foo",
			Worktree: "/x/myrepo-feature-foo", // ".." sibling of the repo
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("Vars mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("site_name that expands to a non-slug is rejected", func(t *testing.T) {
		p := &config.Profile{SiteName: "$PROJECT/$TREE", WorktreesDir: ".."}
		if _, err := Resolve("/x/myrepo", "foo", p); err == nil {
			t.Fatal("want error for slug containing '/', got nil")
		}
	})

	t.Run("undefined variable in site_name surfaces", func(t *testing.T) {
		p := &config.Profile{SiteName: "$BOGUS", WorktreesDir: ".."}
		if _, err := Resolve("/x/myrepo", "foo", p); err == nil {
			t.Fatal("want error for undefined $BOGUS, got nil")
		}
	})
}

func TestExpand(t *testing.T) {
	t.Parallel()

	v := &Vars{Project: "proj", Branch: "feature/foo", Tree: "feature-foo"}

	t.Run("bare and braced forms", func(t *testing.T) {
		for _, tmpl := range []string{"$PROJECT", "${PROJECT}"} {
			got, err := v.Expand(tmpl)
			if err != nil {
				t.Fatalf("Expand(%q): %v", tmpl, err)
			}
			if got != "proj" {
				t.Errorf("Expand(%q) = %q, want %q", tmpl, got, "proj")
			}
		}
	})

	t.Run("undefined variable errors", func(t *testing.T) {
		if _, err := v.Expand("$BOGUS"); err == nil {
			t.Fatal("want error for $BOGUS, got nil")
		}
	})

	t.Run("PORT reflects a value assigned after Resolve", func(t *testing.T) {
		v := &Vars{Port: 3000} // Expand rebuilds its table each call, so this is visible
		got, err := v.Expand("$PORT")
		if err != nil {
			t.Fatalf("Expand: %v", err)
		}
		if got != "3000" {
			t.Errorf("Expand($PORT) = %q, want %q", got, "3000")
		}
	})
}

func TestEnviron(t *testing.T) {
	t.Parallel()

	v := &Vars{
		RepoRoot: "/r",
		Project:  "proj",
		Branch:   "b",
		Tree:     "b",
		SiteName: "proj-b",
		Worktree: "/wt",
		Port:     4000,
	}

	want := []string{
		"REPO_ROOT=/r",
		"PROJECT=proj",
		"BRANCH=b",
		"TREE=b",
		"SITE_NAME=proj-b",
		"WORKTREE=/wt",
		"PORT=4000",
	}
	if diff := cmp.Diff(want, v.Environ()); diff != "" {
		t.Errorf("Environ mismatch (-want +got):\n%s", diff)
	}
}

func TestWorktreePath(t *testing.T) {
	t.Run("relative dir joins under the repo", func(t *testing.T) {
		if got := worktreePath("/repo", "sub", "site"); got != "/repo/sub/site" {
			t.Errorf("got %q, want %q", got, "/repo/sub/site")
		}
	})

	t.Run("absolute dir is used as-is", func(t *testing.T) {
		if got := worktreePath("/repo", "/abs/wt", "site"); got != "/abs/wt/site" {
			t.Errorf("got %q, want %q", got, "/abs/wt/site")
		}
	})

	t.Run("~/ expands to the home directory", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)

		want := filepath.Join(home, "wt", "site")
		if got := worktreePath("/repo", "~/wt", "site"); got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

func TestIsValidSlug(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in   string
		want bool
	}{
		{"myrepo-feature-foo", true},
		{"a.b_c-1", true},
		{"a/b", false},
		{"a b", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := isValidSlug(tt.in); got != tt.want {
			t.Errorf("isValidSlug(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}
