package test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/windhooked/benthos/v3/lib/condition"
	"github.com/windhooked/benthos/v3/lib/log"
	"github.com/windhooked/benthos/v3/lib/metrics"
	"github.com/windhooked/benthos/v3/lib/processor"
	"github.com/windhooked/benthos/v3/lib/types"
	"github.com/fatih/color"
	yaml "gopkg.in/yaml.v3"
)

type mockProvider map[string][]types.Processor

func (m mockProvider) Provide(ptr string, env map[string]string) ([]types.Processor, error) {
	if procs, ok := m[ptr]; ok {
		return procs, nil
	}
	return nil, errors.New("processors not found")
}

func TestCase(t *testing.T) {
	color.NoColor = true

	provider := mockProvider{}

	procConf := processor.NewConfig()
	procConf.Type = processor.TypeNoop
	proc, err := processor.New(procConf, nil, log.Noop(), metrics.Noop())
	if err != nil {
		t.Fatal(err)
	}
	provider["/pipeline/processors"] = []types.Processor{proc}

	procConf = processor.NewConfig()
	procConf.Type = processor.TypeText
	procConf.Text.Operator = "to_upper"
	if proc, err = processor.New(procConf, nil, log.Noop(), metrics.Noop()); err != nil {
		t.Fatal(err)
	}
	provider["/input/broker/inputs/0/processors"] = []types.Processor{proc}

	procConf = processor.NewConfig()
	procConf.Type = processor.TypeFilter
	procConf.Filter.Type = condition.TypeStatic
	procConf.Filter.Static = false
	if proc, err = processor.New(procConf, nil, log.Noop(), metrics.Noop()); err != nil {
		t.Fatal(err)
	}
	provider["/input/broker/inputs/1/processors"] = []types.Processor{proc}

	type testCase struct {
		name     string
		conf     string
		expected []CaseFailure
	}

	tests := []testCase{
		{
			name: "positive 1",
			conf: `
name: positive 1
input_batch:
- content: foo bar
output_batches:
-
  - content_equals: "foo bar"
`,
		},
		{
			name: "positive 2",
			conf: `
name: positive 2
target_processors: /input/broker/inputs/0/processors
input_batch:
- content: foo bar
output_batches:
-
  - content_equals: "FOO BAR"
`,
		},
		{
			name: "positive 3",
			conf: `
name: positive 3
target_processors: /input/broker/inputs/1/processors
input_batch:
- content: foo bar
output_batches: []`,
		},
		{
			name: "negative 1",
			conf: `
name: negative 1
input_batch:
- content: foo bar
output_batches:
-
  - content_equals: "foo baz"
`,
			expected: []CaseFailure{
				{
					Name:     "negative 1",
					TestLine: 2,
					Reason:   "batch 0 message 0: content_equals: content mismatch\n  expected: foo baz\n  received: foo bar",
				},
			},
		},
		{
			name: "negative 2",
			conf: `
name: negative 2
input_batch:
- content: foo bar
- content: foo baz
  metadata:
    foo: baz
output_batches:
-
  - content_equals: "foo bar"
  - content_equals: "bar baz"
    metadata_equals:
      foo: bar
`,
			expected: []CaseFailure{
				{
					Name:     "negative 2",
					TestLine: 2,
					Reason:   "batch 0 message 1: content_equals: content mismatch\n  expected: bar baz\n  received: foo baz",
				},
				{
					Name:     "negative 2",
					TestLine: 2,
					Reason:   "batch 0 message 1: metadata_equals: metadata key 'foo' mismatch\n  expected: bar\n  received: baz",
				},
			},
		},
		{
			name: "negative batches count 1",
			conf: `
name: negative batches count 1
input_batch:
- content: foo bar
output_batches:
-
  - content_equals: "foo bar"
-
  - content_equals: "foo bar"
`,
			expected: []CaseFailure{
				{
					Name:     "negative batches count 1",
					TestLine: 2,
					Reason:   "wrong batch count, expected 2, got 1",
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			c := NewCase()
			if err = yaml.Unmarshal([]byte(test.conf), &c); err != nil {
				tt.Fatal(err)
			}
			fails, err := c.Execute(provider)
			if err != nil {
				tt.Fatal(err)
			}
			if exp, act := test.expected, fails; !reflect.DeepEqual(exp, act) {
				tt.Errorf("Wrong results: %v != %v", act, exp)
			}
		})
	}
}
