package field

import (
	"strconv"

	"github.com/windhooked/benthos/v3/lib/bloblang/x/query"
)

//------------------------------------------------------------------------------

type resolver interface {
	ResolveString(index int, msg Message, escaped, legacy bool) string
	ResolveBytes(index int, msg Message, escaped, legacy bool) []byte
}

//------------------------------------------------------------------------------

type staticResolver string

func (s staticResolver) ResolveString(index int, msg Message, escaped, legacy bool) string {
	return string(s)
}

func (s staticResolver) ResolveBytes(index int, msg Message, escaped, legacy bool) []byte {
	return []byte(s)
}

//------------------------------------------------------------------------------

type queryResolver struct {
	fn query.Function
}

func (q queryResolver) ResolveString(index int, msg Message, escaped, legacy bool) string {
	return query.ExecToString(q.fn, query.FunctionContext{
		Index:  index,
		Msg:    msg,
		Legacy: legacy,
	})
}

func (q queryResolver) ResolveBytes(index int, msg Message, escaped, legacy bool) []byte {
	bs := query.ExecToBytes(q.fn, query.FunctionContext{
		Index:  index,
		Msg:    msg,
		Legacy: legacy,
	})
	if escaped {
		bs = escapeBytes(bs)
	}
	return bs
}

func escapeBytes(in []byte) []byte {
	quoted := strconv.Quote(string(in))
	if len(quoted) < 3 {
		return in
	}
	return []byte(quoted[1 : len(quoted)-1])
}

//------------------------------------------------------------------------------
