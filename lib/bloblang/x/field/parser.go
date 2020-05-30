package field

import (
	"fmt"

	"github.com/windhooked/benthos/v3/lib/bloblang/x/parser"
	"github.com/windhooked/benthos/v3/lib/bloblang/x/query"
)

//------------------------------------------------------------------------------

func intoStaticResolver(p parser.Type) parser.Type {
	return func(input []rune) parser.Result {
		res := p(input)
		if str, ok := res.Payload.(string); ok {
			res.Payload = staticResolver(str)
		}
		return res
	}
}

func aFunction(input []rune) parser.Result {
	if len(input) < 3 || input[0] != '$' || input[1] != '{' || input[2] != '!' {
		return parser.Result{
			Payload:   nil,
			Err:       parser.ExpectedError{"${!"},
			Remaining: input,
		}
	}
	i := 3
	for ; i < len(input); i++ {
		if input[i] == '}' {
			res := query.ParseDeprecated(input[3:i])
			if res.Err == nil {
				if len(res.Remaining) > 0 {
					return parser.Result{
						Err: parser.ErrAtPosition(
							i-len(res.Remaining),
							fmt.Errorf("unexpected contents at end of expression: %v", string(res.Remaining)),
						),
						Remaining: input,
					}
				}
				res.Remaining = input[i+1:]
				res.Payload = queryResolver{fn: res.Payload.(query.Function)}
			} else {
				res.Err = parser.ErrAtPosition(3, res.Err).Expand(func(err error) error {
					// Scrap underlying expected error.
					return fmt.Errorf("%v", err.Error())
				})
				res.Remaining = input
			}
			return res
		}
	}
	return parser.Result{
		Payload:   staticResolver(string(input)),
		Err:       nil,
		Remaining: nil,
	}
}

func escapedBlock(input []rune) parser.Result {
	if len(input) < 4 || input[0] != '$' || input[1] != '{' || input[2] != '{' || input[3] != '!' {
		return parser.Result{
			Payload:   nil,
			Err:       parser.ExpectedError{"${{!"},
			Remaining: input,
		}
	}
	i := 4
	for ; i < len(input)-1; i++ {
		if input[i] == '}' && input[i+1] == '}' {
			return parser.Result{
				Payload:   staticResolver("${!" + string(input[4:i]) + "}"),
				Err:       nil,
				Remaining: input[i+2:],
			}
		}
	}
	return parser.Result{
		Payload:   staticResolver(string(input)),
		Err:       nil,
		Remaining: nil,
	}
}

//------------------------------------------------------------------------------

func parse(expr string) (*expression, error) {
	var resolvers []resolver

	p := parser.OneOf(
		escapedBlock,
		aFunction,
		intoStaticResolver(parser.Char('$')),
		intoStaticResolver(parser.NotChar('$')),
	)

	remaining := []rune(expr)
	i := 0
	for len(remaining) > 0 {
		res := p(remaining)
		if res.Err != nil {
			return nil, fmt.Errorf("failed to parse expression: %v", parser.ErrAtPosition(i, res.Err))
		}
		i = len(remaining) - len(res.Remaining)
		remaining = res.Remaining
		resolvers = append(resolvers, res.Payload.(resolver))
	}

	return buildExpression(resolvers), nil
}

//------------------------------------------------------------------------------
