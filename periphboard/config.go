package periphboard

import (
	"fmt"

	"go.viam.com/rdk/components/board"
	"go.viam.com/rdk/utils"
)

// A Config describes the configuration of a board and all of its connected parts.
type Config struct {
	SPIs              []board.SPIConfig              `json:"spis,omitempty"`
	Analogs           []board.AnalogConfig           `json:"analogs,omitempty"`
	Attributes        utils.AttributeMap             `json:"attributes,omitempty"`
}

// Validate ensures all parts of the config are valid.
func (conf *Config) Validate(path string) ([]string, error) {
	for idx, c := range conf.SPIs {
		if err := c.Validate(fmt.Sprintf("%s.%s.%d", path, "spis", idx)); err != nil {
			return nil, err
		}
	}
	for idx, c := range conf.Analogs {
		if err := c.Validate(fmt.Sprintf("%s.%s.%d", path, "analogs", idx)); err != nil {
			return nil, err
		}
	}
	return nil, nil
}
