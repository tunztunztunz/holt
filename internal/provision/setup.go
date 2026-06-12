package provision

import (
	"os"
	"os/exec"
)

// RunCommand runs one setup/teardown line in the worktree with vars exported
func RunCommand(worktree, line string, env []string) error {
	cmd := exec.Command("bash", "-c", line)
	cmd.Dir = worktree
	cmd.Env = append(os.Environ(), env...)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
