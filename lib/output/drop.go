package output

import (
	"github.com/windhooked/benthos/v3/lib/log"
	"github.com/windhooked/benthos/v3/lib/metrics"
	"github.com/windhooked/benthos/v3/lib/output/writer"
	"github.com/windhooked/benthos/v3/lib/types"
)

//------------------------------------------------------------------------------

func init() {
	Constructors[TypeDrop] = TypeSpec{
		constructor: NewDrop,
		Summary: `
Drops all messages.`,
	}
}

//------------------------------------------------------------------------------

// NewDrop creates a new Drop output type.
func NewDrop(conf Config, mgr types.Manager, log log.Modular, stats metrics.Type) (Type, error) {
	return NewWriter(
		TypeDrop, writer.NewDrop(conf.Drop, log, stats), log, stats,
	)
}

//------------------------------------------------------------------------------
