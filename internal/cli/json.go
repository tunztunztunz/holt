package cli

import (
	"encoding/json"
	"os"
)

type envelope struct {
	Holt     string   `json:"holt"`
	Command  string   `json:"command"`
	Data     any      `json:"data"`
	Warnings []string `json:"warnings,omitempty"`
}

func emitJSON(command string, data any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", " ")
	return enc.Encode(envelope{
		Holt:    Version,
		Command: command,
		Data:    data,
	})
}
