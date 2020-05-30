package client

import (
	"github.com/windhooked/benthos/v3/lib/util/aws/session"
	"github.com/windhooked/benthos/v3/lib/x/docs"
)

//------------------------------------------------------------------------------

// FieldSpecs returns field specs for a lambda client config.
func FieldSpecs() docs.FieldSpecs {
	return docs.FieldSpecs{
		docs.FieldCommon("function", "The function to invoke."),
		docs.FieldAdvanced("rate_limit", "An optional [`rate_limit`](/docs/components/rate_limits/about) to throttle invocations by."),
	}.Merge(session.FieldSpecs()).Add(
		docs.FieldAdvanced("timeout", "The maximum period of time to wait before abandoning an invocation."),
		docs.FieldAdvanced("retries", "The maximum number of retry attempts for each message."),
	)
}

//------------------------------------------------------------------------------
