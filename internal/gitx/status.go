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

func WorktreeStatus(worktree string) (Status, error) {
	var s Status

	out, err := run(worktree, "status", "--porcelain")
	if err != nil {
		return s, err
	}
	s.Dirty = strings.TrimSpace(out) != ""

	counts, err := run(worktree, "rev-list", "--left-right", "--count", "@{upstream}...HEAD")
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
