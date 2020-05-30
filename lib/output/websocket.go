package output

import (
	"github.com/windhooked/benthos/v3/lib/log"
	"github.com/windhooked/benthos/v3/lib/metrics"
	"github.com/windhooked/benthos/v3/lib/output/writer"
	"github.com/windhooked/benthos/v3/lib/types"
	"github.com/windhooked/benthos/v3/lib/util/http/auth"
	"github.com/windhooked/benthos/v3/lib/x/docs"
)

//------------------------------------------------------------------------------

func init() {
	Constructors[TypeWebsocket] = TypeSpec{
		constructor: NewWebsocket,
		Summary: `
Sends messages to an HTTP server via a websocket connection.`,
		FieldSpecs: docs.FieldSpecs{
			docs.FieldCommon("url", "The URL to connect to."),
		}.Merge(auth.FieldSpecs()),
	}
}

//------------------------------------------------------------------------------

// NewWebsocket creates a new Websocket output type.
func NewWebsocket(conf Config, mgr types.Manager, log log.Modular, stats metrics.Type) (Type, error) {
	w, err := writer.NewWebsocket(conf.Websocket, log, stats)
	if err != nil {
		return nil, err
	}
	return NewWriter(TypeWebsocket, w, log, stats)
}

//------------------------------------------------------------------------------
