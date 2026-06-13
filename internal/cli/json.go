package cli

import (
	"encoding/json"
	"os"
)

type envelope struct {
	Acre     string   `json:"acre"`
	Command  string   `json:"command"`
	Data     any      `json:"data"`
	Warnings []string `json:"warnings,omitempty"`
}

func emitJSON(command string, data any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", " ")
	return enc.Encode(envelope{
		Acre:    Version,
		Command: command,
		Data:    data,
	})
}
