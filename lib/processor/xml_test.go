package processor

import (
	"testing"

	"github.com/windhooked/benthos/v3/lib/log"
	"github.com/windhooked/benthos/v3/lib/message"
	"github.com/windhooked/benthos/v3/lib/metrics"
)

func TestXMLCases(t *testing.T) {
	type testCase struct {
		name   string
		input  string
		output string
	}
	tests := []testCase{
		{
			name: "basic 1",
			input: `<root>
  <next>foo1</next>
</root>`,
			output: `{"root":{"next":"foo1"}}`,
		},
		{
			name: "basic 2",
			input: `<root>
  <next>foo1</next>
  <inner>
  	<thing>10</thing>
  </inner>
</root>`,
			output: `{"root":{"inner":{"thing":"10"},"next":"foo1"}}`,
		},
		{
			name: "with array 1",
			input: `<root>
  <next>foo1</next>
  <next>foo2</next>
  <next>foo3</next>
</root>`,
			output: `{"root":{"next":["foo1","foo2","foo3"]}}`,
		},
		{
			name: "with attributes 1",
			input: `<root isRooted="true">
  <next withinRoot="yes">foo1</next>
  <inner>
  	<thing someAttr="is boring" someAttr2="is also boring">10</thing>
  </inner>
</root>`,
			output: `{"root":{"-isRooted":"true","inner":{"thing":{"#text":"10","-someAttr":"is boring","-someAttr2":"is also boring"}},"next":{"#text":"foo1","-withinRoot":"yes"}}}`,
		},
		{
			name: "array with attributes 1",
			input: `<root>
  <title>This is a title</title>
  <description tone="boring">This is a description</description>
  <elements id="1">foo1</elements>
  <elements id="2">foo2</elements>
  <elements>foo3</elements>
</root>`,
			output: `{"root":{"description":{"#text":"This is a description","-tone":"boring"},"elements":[{"#text":"foo1","-id":"1"},{"#text":"foo2","-id":"2"},"foo3"],"title":"This is a title"}}`,
		},
	}

	conf := NewConfig()
	proc, err := NewXML(conf, nil, log.Noop(), metrics.Noop())
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			msgsOut, res := proc.ProcessMessage(message.New([][]byte{[]byte(test.input)}))
			if res != nil {
				tt.Fatal(res.Error())
			}
			if len(msgsOut) != 1 {
				tt.Fatalf("Wrong count of result messages: %v != 1", len(msgsOut))
			}
			if exp, act := test.output, string(msgsOut[0].Get(0).Get()); exp != act {
				tt.Errorf("Wrong result: %v != %v", act, exp)
			}
		})
	}
}
