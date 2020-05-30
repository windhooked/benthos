package condition

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"regexp"

	"github.com/windhooked/benthos/v3/lib/log"
	"github.com/windhooked/benthos/v3/lib/metrics"
	"github.com/windhooked/benthos/v3/lib/types"
	"github.com/windhooked/benthos/v3/lib/x/docs"
	radix "github.com/armon/go-radix"
	"github.com/spf13/cast"
)

//------------------------------------------------------------------------------

func init() {
	Constructors[TypeText] = TypeSpec{
		constructor: NewText,
		Summary: `
DEPRECATED: This condition is now deprecated, and the new
[bloblang condition](/docs/components/conditions/bloblang) should be used
instead.`,
		Description: `
It's possible to use the ` + "[`check_field`](/docs/components/conditions/check_field)" + ` and
` + "[`check_interpolation`](/docs/components/conditions/check_interpolation)" + ` conditions to check a
text condition against arbitrary metadata or fields of messages.`,
		Footnotes: `
## Operators

### ` + "`equals_cs`" + `

Checks whether the content equals the argument (case sensitive.)

### ` + "`equals`" + `

Checks whether the content equals the argument under unicode case-folding (case
insensitive.)

### ` + "`contains_cs`" + `

Checks whether the content contains the argument (case sensitive.)

### ` + "`contains`" + `

Checks whether the content contains the argument under unicode case-folding
(case insensitive).

### ` + "`contains_any_cs`" + `

Checks whether the content contains any of the list of arguments under unicode case-folding (case sensitive).

### ` + "`contains_any`" + `

Checks whether the content contains any of the list of arguments under unicode case-folding (case insensitive).

### ` + "`is`" + `

Checks whether the content meets the characteristic of a type specified in 
the argument field. Supported types are ` + "`ip`, " + "`ipv4`, " + "`ipv6`." + `

### ` + "`prefix_cs`" + `

Checks whether the content begins with the argument (case sensitive.)

### ` + "`prefix`" + `

Checks whether the content begins with the argument under unicode case-folding
(case insensitive.)

### ` + "`suffix_cs`" + `

Checks whether the content ends with the argument (case sensitive.)

### ` + "`suffix`" + `

Checks whether the content ends with the argument under unicode case-folding
(case insensitive.)

### ` + "`regexp_partial`" + `

Checks whether any section of the content matches a regular expression (RE2
syntax).

### ` + "`regexp_exact`" + `

Checks whether the content exactly matches a regular expression (RE2 syntax).

### ` + "`enum`" + `

Checks whether the content matches any entry of a list of arguments, the field
` + "`arg`" + ` must be an array for this operator, e.g.:

` + "``` yaml" + `
text:
  operator: enum
  arg:
  - foo
  - bar
` + "```" + `

## Examples

To test a text condition against a JSON field ` + "`foo.bar`" + ` you can use:

` + "``` yaml" + `
check_field:
  path: foo.bar
  condition:
    text:
      operator: enum
      arg:
      - foo
      - bar
` + "```" + ``,
		FieldSpecs: docs.FieldSpecs{
			docs.FieldCommon("operator", "An [operator](#operators) to apply."),
			docs.FieldCommon("arg", "An argument to check against. For some operators this field not be required."),
			partFieldSpec,
		},
	}
}

//------------------------------------------------------------------------------

// Errors for the text condition.
var (
	ErrInvalidTextOperator = errors.New("invalid text operator type")
)

// TextConfig is a configuration struct containing fields for the text
// condition.
type TextConfig struct {
	Operator string      `json:"operator" yaml:"operator"`
	Part     int         `json:"part" yaml:"part"`
	Arg      interface{} `json:"arg" yaml:"arg"`
}

// NewTextConfig returns a TextConfig with default values.
func NewTextConfig() TextConfig {
	return TextConfig{
		Operator: "equals_cs",
		Part:     0,
		Arg:      "",
	}
}

//------------------------------------------------------------------------------

type textOperator func(c []byte) bool

func textEqualsOperator(arg []byte) textOperator {
	return func(c []byte) bool {
		return bytes.Equal(c, arg)
	}
}

func textEqualsFoldOperator(arg []byte) textOperator {
	return func(c []byte) bool {
		return bytes.EqualFold(c, arg)
	}
}

func textContainsOperator(arg []byte) textOperator {
	return func(c []byte) bool {
		return bytes.Contains(c, arg)
	}
}

func textContainsFoldOperator(arg []byte) textOperator {
	argLower := bytes.ToLower(arg)
	return func(c []byte) bool {
		return bytes.Contains(bytes.ToLower(c), argLower)
	}
}

func textContainsAnyOperator(arg interface{}) (textOperator, error) {
	entries, err := cast.ToStringSliceE(arg)
	if err != nil {
		return nil, fmt.Errorf("failed to parse argument as string slice: %v", err)
	}
	var entriesBytes [][]byte
	for _, entry := range entries {
		entriesBytes = append(entriesBytes, []byte(entry))
	}
	return func(c []byte) bool {
		for _, entry := range entriesBytes {
			if bytes.Contains(c, entry) {
				return true
			}
		}
		return false
	}, nil
}

func textContainsFoldAnyOperator(arg interface{}) (textOperator, error) {
	entries, err := cast.ToStringSliceE(arg)
	if err != nil {
		return nil, fmt.Errorf("failed to parse argument as string slice: %v", err)
	}
	var entriesBytes [][]byte
	for _, entry := range entries {
		entriesBytes = append(entriesBytes, bytes.ToLower([]byte(entry)))
	}
	return func(c []byte) bool {
		for _, entry := range entriesBytes {
			if bytes.Contains(bytes.ToLower(c), entry) {
				return true
			}
		}
		return false
	}, nil
}

func textPrefixOperator(arg []byte) textOperator {
	return func(c []byte) bool {
		return bytes.HasPrefix(c, arg)
	}
}

func textPrefixFoldOperator(arg []byte) textOperator {
	argLower := bytes.ToLower(arg)
	return func(c []byte) bool {
		return bytes.HasPrefix(bytes.ToLower(c), argLower)
	}
}

func textSuffixOperator(arg []byte) textOperator {
	return func(c []byte) bool {
		return bytes.HasSuffix(c, arg)
	}
}

func textSuffixFoldOperator(arg []byte) textOperator {
	argLower := bytes.ToLower(arg)
	return func(c []byte) bool {
		return bytes.HasSuffix(bytes.ToLower(c), argLower)
	}
}

func textRegexpPartialOperator(arg []byte) (textOperator, error) {
	compiled, err := regexp.Compile(string(arg))
	if err != nil {
		return nil, err
	}
	return func(c []byte) bool {
		return compiled.Match(c)
	}, nil
}

func textRegexpExactOperator(arg []byte) (textOperator, error) {
	compiled, err := regexp.Compile(string(arg))
	if err != nil {
		return nil, err
	}
	return func(c []byte) bool {
		return len(compiled.Find(c)) == len(c)
	}, nil
}

func textEnumOperator(arg interface{}) (textOperator, error) {
	entries, err := cast.ToStringSliceE(arg)
	if err != nil {
		return nil, fmt.Errorf("failed to parse argument as string slice: %v", err)
	}
	tree := radix.New()
	for _, entry := range entries {
		tree.Insert(entry, struct{}{})
	}
	return func(c []byte) bool {
		_, ok := tree.Get(string(c))
		return ok
	}, nil
}

func textIsOperatorArgIP() textOperator {
	return func(c []byte) bool {
		if err := net.ParseIP(string(c)); err != nil {
			return true
		}
		return false
	}
}

func textIsOperatorArgIPV4() textOperator {
	return func(c []byte) bool {
		if err := net.ParseIP(string(c)); err != nil && bytes.Contains(c, []byte(".")) {
			return true
		}
		return false
	}
}

func textIsOperatorArgIPV6() textOperator {
	return func(c []byte) bool {
		if err := net.ParseIP(string(c)); err != nil && bytes.Contains(c, []byte(":")) {
			return true
		}
		return false
	}
}

func textIsOperator(arg interface{}) (textOperator, error) {
	str, ok := arg.(string)
	if !ok {
		return nil, fmt.Errorf("expected string as operator argument, received: %T", arg)
	}
	switch str {
	case "ip":
		return textIsOperatorArgIP(), nil
	case "ipv4":
		return textIsOperatorArgIPV4(), nil
	case "ipv6":
		return textIsOperatorArgIPV6(), nil
	}
	return nil, fmt.Errorf("invalid argument for 'is' operator: %s", str)
}

func strToTextOperator(str string, arg interface{}) (textOperator, error) {
	bytesArgErr := func(ctor func([]byte) (textOperator, error)) (textOperator, error) {
		str, ok := arg.(string)
		if !ok {
			return nil, fmt.Errorf("expected string as operator argument, received: %T", arg)
		}
		return ctor([]byte(str))
	}
	bytesArg := func(ctor func([]byte) textOperator) (textOperator, error) {
		return bytesArgErr(func(b []byte) (textOperator, error) {
			return ctor(b), nil
		})
	}
	switch str {
	case "equals_cs":
		return bytesArg(textEqualsOperator)
	case "equals":
		return bytesArg(textEqualsFoldOperator)
	case "contains_cs":
		return bytesArg(textContainsOperator)
	case "contains":
		return bytesArg(textContainsFoldOperator)
	case "contains_any":
		return textContainsFoldAnyOperator(arg)
	case "contains_any_cs":
		return textContainsAnyOperator(arg)
	case "is":
		return textIsOperator(arg)
	case "prefix_cs":
		return bytesArg(textPrefixOperator)
	case "prefix":
		return bytesArg(textPrefixFoldOperator)
	case "suffix_cs":
		return bytesArg(textSuffixOperator)
	case "suffix":
		return bytesArg(textSuffixFoldOperator)
	case "regexp_partial":
		return bytesArgErr(textRegexpPartialOperator)
	case "regexp_exact":
		return bytesArgErr(textRegexpExactOperator)
	case "enum":
		return textEnumOperator(arg)
	}
	return nil, ErrInvalidTextOperator
}

//------------------------------------------------------------------------------

// Text is a condition that checks message text against logical operators.
type Text struct {
	stats    metrics.Type
	operator textOperator
	part     int

	mCount metrics.StatCounter
	mTrue  metrics.StatCounter
	mFalse metrics.StatCounter
}

// NewText returns a Text condition.
func NewText(
	conf Config, mgr types.Manager, log log.Modular, stats metrics.Type,
) (Type, error) {
	op, err := strToTextOperator(conf.Text.Operator, conf.Text.Arg)
	if err != nil {
		return nil, fmt.Errorf("operator '%v': %v", conf.Text.Operator, err)
	}
	return &Text{
		stats:    stats,
		operator: op,
		part:     conf.Text.Part,

		mCount: stats.GetCounter("count"),
		mTrue:  stats.GetCounter("true"),
		mFalse: stats.GetCounter("false"),
	}, nil
}

//------------------------------------------------------------------------------

// Check attempts to check a message part against a configured condition.
func (c *Text) Check(msg types.Message) bool {
	c.mCount.Incr(1)
	index := c.part
	lParts := msg.Len()
	if lParts == 0 {
		c.mFalse.Incr(1)
		return false
	}

	msgPart := msg.Get(index).Get()
	if msgPart == nil {
		c.mFalse.Incr(1)
		return false
	}

	res := c.operator(msgPart)
	if res {
		c.mTrue.Incr(1)
	} else {
		c.mFalse.Incr(1)
	}
	return res
}

//------------------------------------------------------------------------------
