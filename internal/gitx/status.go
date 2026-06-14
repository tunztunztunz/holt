package gitx

import (
	"strconv"
	"strings"
)

type Status struct {
	Dirty  bool
	Ahead  int
	Behind int
}

// WorktreeStatus reports the worktree's dirty state and how far it has diverged
// from base: Behind = commits on base not yet in this branch, Ahead = commits
// here not yet in base. base is compared with `...` (merge-base), so a freshly
// branched worktree reads 0/0 until base moves. If base is "" or unresolvable,
// ahead/behind stay 0.
func WorktreeStatus(worktree, base string) (Status, error) {
	var s Status

	out, err := run(worktree, "status", "--porcelain")
	if err != nil {
		return s, err
	}
	s.Dirty = strings.TrimSpace(out) != ""

	counts, err := run(worktree, "rev-list", "--left-right", "--count", base+"...HEAD")
	if err == nil {
		if f := strings.Fields(counts); len(f) == 2 {
			s.Behind, err = strconv.Atoi(f[0])
			if err != nil {
				return s, err
			}
			s.Ahead, err = strconv.Atoi(f[1])
			if err != nil {
				return s, err
			}
		}
	}

	return s, nil
}
