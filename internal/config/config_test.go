package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestApplyDefaults(t *testing.T) {
	t.Parallel()

	t.Run("empty profile gets the documented defaults", func(t *testing.T) {
		var p Profile
		p.applyDefaults()

		want := Profile{
			Version:      1,
			SiteName:     "$PROJECT-$TREE",
			WorktreesDir: "..",
			Guards:       []string{"uncommitted", "unmerged"},
		}
		if diff := cmp.Diff(want, p); diff != "" {
			t.Errorf("defaults mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("explicit values are preserved", func(t *testing.T) {
		p := Profile{
			Version:      2,
			SiteName:     "custom",
			WorktreesDir: "~/wt",
			Guards:       []string{"unpushed"},
		}
		want := p
		p.applyDefaults()

		if diff := cmp.Diff(want, p); diff != "" {
			t.Errorf("applyDefaults overwrote set values (-want +got):\n%s", diff)
		}
	})
}

func TestValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		profile Profile
		wantErr bool
	}{
		{"empty profile", Profile{}, false},
		{"valid port + default strategy", Profile{Port: &PortBlock{Range: [2]int{4000, 4999}}}, false},
		{"valid hash strategy", Profile{Port: &PortBlock{Range: [2]int{4000, 4999}, Strategy: PortHash}}, false},
		{"valid free strategy", Profile{Port: &PortBlock{Range: [2]int{4000, 4999}, Strategy: PortFree}}, false},
		{"port below 1024", Profile{Port: &PortBlock{Range: [2]int{100, 200}}}, true},
		{"port above 65535", Profile{Port: &PortBlock{Range: [2]int{1024, 70000}}}, true},
		{"reversed range", Profile{Port: &PortBlock{Range: [2]int{5000, 4000}}}, true},
		{"unknown strategy", Profile{Port: &PortBlock{Range: [2]int{4000, 4999}, Strategy: "weird"}}, true},
		{"unknown guard", Profile{Guards: []string{"bogus"}}, true},
		{"copy escapes worktree", Profile{Copy: []string{"../x"}}, true},
		{"copy stays local", Profile{Copy: []string{"a/b"}}, false},
		{"env.file escapes worktree", Profile{Env: []EnvBlock{{File: "../x"}}}, true},
		{"link escapes worktree", Profile{Link: []string{"../x"}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.profile.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoad(t *testing.T) {
	t.Run("valid file parses and gets defaults", func(t *testing.T) {
		dir := writeProfile(t, "site_name: $PROJECT-$TREE\ncopy:\n  - .env\n")

		p, err := Load(dir)
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if p.Version != 1 { // default applied
			t.Errorf("Version = %d, want 1", p.Version)
		}
		if diff := cmp.Diff([]string{".env"}, p.Copy); diff != "" {
			t.Errorf("Copy mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("missing file is a clear error", func(t *testing.T) {
		if _, err := Load(t.TempDir()); err == nil || !strings.Contains(err.Error(), "no holt.yml") {
			t.Fatalf("want 'no holt.yml' error, got %v", err)
		}
	})

	t.Run("unknown field is rejected", func(t *testing.T) {
		dir := writeProfile(t, "bogus_field: 1\n")
		if _, err := Load(dir); err == nil || !strings.Contains(err.Error(), "invalid holt.yml") {
			t.Fatalf("want 'invalid holt.yml' error, got %v", err)
		}
	})
}

// writeProfile writes holt.yml into a fresh temp dir and returns the dir.
func writeProfile(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "holt.yml"), []byte(body), 0o644); err != nil {
		t.Fatalf("write holt.yml: %v", err)
	}
	return dir
}
