package pipeline

import (
	"github.com/windhooked/benthos/v3/lib/types"
)

// Type is implemented by all pipeline implementations.
type Type interface {
	types.Pipeline
}
