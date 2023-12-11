package periphboard

import (
	"fmt"

	"go.viam.com/rdk/components/board"
	"go.viam.com/rdk/utils"
)

// A Config describes the configuration of a board and all of its connected parts.
type Config struct {
	Attributes utils.AttributeMap `json:"attributes,omitempty"`
}

// Validate ensures all parts of the config are valid.
func (conf *Config) Validate(path string) ([]string, error) {
	return nil, nil
}
