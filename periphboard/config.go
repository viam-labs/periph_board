package periphboard

import (
	"go.viam.com/rdk/resource"
	"go.viam.com/rdk/utils"
)

// A Config describes the configuration of a board and all of its connected parts.
type Config struct {
	resource.TriviallyValidateConfig

	Attributes utils.AttributeMap `json:"attributes,omitempty"`
}
