package query

import (
	"testing"

	"github.com/windhooked/benthos/v3/lib/bloblang/x/parser"
	"github.com/stretchr/testify/assert"
)

func TestFunctionParserErrors(t *testing.T) {
	tests := map[string]struct {
		input      string
		err        string
		deprecated bool
	}{
		"bad function": {
			input:      `not a function`,
			deprecated: true,
			err:        `char 3: expected: function-parameters`,
		},
		"bad function 2": {
			input: `not_a_function()`,
			err:   `char 0: unrecognised function 'not_a_function'`,
		},
		"bad args 2": {
			input:      `json("foo`,
			deprecated: true,
			err:        `char 5: required one of: [boolean number quoted-string match function null array object variable-path field-path]`,
		},
		"bad args 3": {
			input: `json(`,
			err:   `char 5: required one of: [boolean number quoted-string match function null array object variable-path field-path]`,
		},
		"bad args 4": {
			input: `json(0,`,
			err:   `char 7: required one of: [boolean number quoted-string match function null array object variable-path field-path]`,
		},
		"bad args 5": {
			input:      `json`,
			deprecated: true,
			err:        `char 4: expected: function-parameters`,
		},
		"bad args 7": {
			input: `json(5)`,
			err:   `char 0: expected string param, received int64`,
		},
		"bad args 8": {
			input: `json(false)`,
			err:   `char 0: expected string param, received bool`,
		},
		"bad operators": {
			input: `json("foo") + `,
			err:   `char 14: expected one of: [match function boolean number quoted-string null array object variable-path field-path]`,
		},
		"bad expression": {
			input: `(json("foo") `,
			err:   `char 13: expected: )`,
		},
		"bad expression 2": {
			input: `(json("foo") + `,
			err:   `char 15: expected one of: [match function boolean number quoted-string null array object variable-path field-path]`,
		},
		"bad expression 3": {
			input: `(json("foo") + meta("bar") `,
			err:   `char 27: expected: )`,
		},
		"bad method": {
			input: `json("foo").not_a_thing()`,
			err:   `char 12: unrecognised method 'not_a_thing'`,
		},
		"bad method 2": {
			input:      `json("foo").not_a_thing()`,
			deprecated: true,
			err:        `char 12: unrecognised method 'not_a_thing'`,
		},
		"bad method args 2": {
			input: `json("foo").from(`,
			err:   `char 17: required one of: [boolean number quoted-string match function null array object variable-path field-path]`,
		},
		"bad method args 3": {
			input: `json("foo").from()`,
			err:   `char 12: expected 1 parameters, received: 0`,
		},
		"bad method args 4": {
			input: `json("foo").from("nah")`,
			err:   `char 12: expected int param, received string`,
		},
		"bad map args": {
			input: `json("foo").map()`,
			err:   `char 12: expected 1 parameters, received: 0`,
		},
		"gibberish": {
			input: `json("foo").(=)`,
			err:   `char 13: required one of: [match function boolean number quoted-string null array object variable-path field-path]`,
		},
		"gibberish 2": {
			input: `json("foo").(1 + )`,
			err:   `char 17: required one of: [match function boolean number quoted-string null array object variable-path field-path]`,
		},
		"bad match": {
			input: `match json("foo")`,
			err:   `char 17: required: {`,
		},
		"bad match 2": {
			input: `match json("foo") what is this?`,
			err:   `char 18: required: {`,
		},
	}

	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, err := tryParse(test.input, test.deprecated)
			_ = assert.Error(t, err) &&
				assert.Equal(t, test.err, err.Error())
		})
	}
}

func TestFunctionParserLimits(t *testing.T) {
	tests := map[string]struct {
		input      string
		remaining  string
		deprecated bool
	}{
		"nothing": {
			input:     `json("foo") + meta("bar")`,
			remaining: ``,
		},
		"space before": {
			input:     `   json("foo") + meta("bar")`,
			remaining: ``,
		},
		"space before 2": {
			input:     `   json("foo")   +    meta("bar")`,
			remaining: ``,
		},
		"unfinished comment": {
			input:     `json("foo") + meta("bar") # Here's a comment`,
			remaining: ` # Here's a comment`,
		},
		"extra text": {
			input:     `json("foo") and this`,
			remaining: ` and this`,
		},
		"extra text 2": {
			input:     `json("foo") + meta("bar") and this`,
			remaining: ` and this`,
		},
		"extra text 3": {
			input:     `json("foo")+meta("bar")and this`,
			remaining: `and this`,
		},
		"extra text 4": {
			input:     `json("foo")+meta("bar")         and this`,
			remaining: `         and this`,
		},
		"squiggly bracket": {
			input:     `json("foo")}`,
			remaining: `}`,
		},
		"normal bracket": {
			input:     `json("foo"))`,
			remaining: `)`,
		},
		"normal bracket 2": {
			input:     `json("foo"))))`,
			remaining: `)))`,
		},
		"normal bracket 3": {
			input:     `json("foo")) + json("bar")`,
			remaining: `) + json("bar")`,
		},
		"path literals": {
			input:     `this.foo bar baz`,
			remaining: ` bar baz`,
		},
		"path literals 2": {
			input:     `this.foo . bar baz`,
			remaining: ` . bar baz`,
		},
		"brackets at root": {
			input:     `(json().foo | "fallback").from_all()`,
			remaining: ``,
		},
		"brackets after root": {
			input:     `this.root.(json().foo | "fallback").from_all()`,
			remaining: ``,
		},
		"brackets after root 2": {
			input:     `this.root.(json().foo | "fallback").from_all().bar.baz`,
			remaining: ``,
		},
		"this at root": {
			input:     `this.foo.bar and then this`,
			remaining: ` and then this`,
		},
		"path literal at root": {
			input:     `foo.bar and then this`,
			remaining: ` and then this`,
		},
		"match expression": {
			input: `match null {
	"foo" == "bar" => "baz"
	5 > 10 => "or this"
}
not this`,
			remaining: "\nnot this",
		},
		"operators and line breaks": {
			input: `(5 * 8) +
	6 -
	5 and also this`,
			remaining: " and also this",
		},
	}

	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var res parser.Result
			if test.deprecated {
				res = ParseDeprecated([]rune(test.input))
			} else {
				res = Parse([]rune(test.input))
			}
			_ = assert.NoError(t, res.Err) &&
				assert.Equal(t, test.remaining, string(res.Remaining))
		})
	}
}
