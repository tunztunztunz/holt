package cli

import (
	"os"

	"charm.land/huh/v2"
)

func confirm(prompt string) bool {
	if !isTTY(os.Stdin) {
		return false
	}
	var ok bool
	form := huh.NewForm(huh.NewGroup(
		huh.NewConfirm().Title(prompt).Value(&ok),
	)).WithOutput(os.Stderr).WithInput(os.Stdin)

	if err := form.Run(); err != nil {
		return false
	}
	return ok
}
