package processor

import (
	"os"
	"testing"

	"github.com/windhooked/benthos/v3/lib/log"
	"github.com/windhooked/benthos/v3/lib/message"
	"github.com/windhooked/benthos/v3/lib/metrics"
	"github.com/windhooked/benthos/v3/lib/util/config"
	yaml "gopkg.in/yaml.v3"
)

func TestJSONValidation(t *testing.T) {
	conf := NewConfig()
	conf.JSON.Operator = "dfjjkdsgjkdfhgjfh"
	conf.JSON.Parts = []int{0}
	conf.JSON.Path = "foo.bar"
	conf.JSON.Value = []byte(`this isnt valid json`)

	testLog := log.New(os.Stdout, log.Config{LogLevel: "NONE"})

	if _, err := NewJSON(conf, nil, testLog, metrics.DudType{}); err == nil {
		t.Error("Expected error from bad operator")
	}

	conf = NewConfig()
	conf.JSON.Operator = "move"
	conf.JSON.Parts = []int{0}
	conf.JSON.Path = "foo.bar"
	conf.JSON.Value = []byte(`#%#@$his isnt valid json`)

	if _, err := NewJSON(conf, nil, testLog, metrics.DudType{}); err == nil {
		t.Error("Expected error from bad value")
	}

	conf = NewConfig()
	conf.JSON.Operator = "move"
	conf.JSON.Parts = []int{0}
	conf.JSON.Path = ""
	conf.JSON.Value = []byte(`""`)

	if _, err := NewJSON(conf, nil, testLog, metrics.DudType{}); err == nil {
		t.Error("Expected error from empty move paths")
	}

	conf = NewConfig()
	conf.JSON.Operator = "copy"
	conf.JSON.Parts = []int{0}
	conf.JSON.Path = ""
	conf.JSON.Value = []byte(`"foo.bar"`)

	if _, err := NewJSON(conf, nil, testLog, metrics.DudType{}); err == nil {
		t.Error("Expected error from empty copy path")
	}

	conf = NewConfig()
	conf.JSON.Operator = "copy"
	conf.JSON.Parts = []int{0}
	conf.JSON.Path = "foo.bar"
	conf.JSON.Value = []byte(`""`)

	if _, err := NewJSON(conf, nil, testLog, metrics.DudType{}); err == nil {
		t.Error("Expected error from empty copy destination")
	}

	conf = NewConfig()
	conf.JSON.Operator = "set"
	conf.JSON.Parts = []int{0}
	conf.JSON.Path = "foo.bar"
	conf.JSON.Value = []byte(`this isnt valid json`)

	jSet, err := NewJSON(conf, nil, testLog, metrics.DudType{})
	if err != nil {
		t.Fatal(err)
	}

	msgIn := message.New([][]byte{[]byte("this is bad json")})
	msgs, res := jSet.ProcessMessage(msgIn)
	if len(msgs) != 1 {
		t.Fatal("No passthrough for bad input data")
	}
	if res != nil {
		t.Fatal("Non-nil result")
	}
	if exp, act := "this is bad json", string(message.GetAllBytes(msgs[0])[0]); exp != act {
		t.Errorf("Wrong output from bad json: %v != %v", act, exp)
	}

	conf.JSON.Parts = []int{5}

	jSet, err = NewJSON(conf, nil, testLog, metrics.DudType{})
	if err != nil {
		t.Fatal(err)
	}

	msgIn = message.New([][]byte{[]byte("{}")})
	msgs, res = jSet.ProcessMessage(msgIn)
	if len(msgs) != 1 {
		t.Fatal("No passthrough for bad index")
	}
	if res != nil {
		t.Fatal("Non-nil result")
	}
	if exp, act := "{}", string(message.GetAllBytes(msgs[0])[0]); exp != act {
		t.Errorf("Wrong output from bad index: %v != %v", act, exp)
	}
}

func TestJSONPartBounds(t *testing.T) {
	tLog := log.New(os.Stdout, log.Config{LogLevel: "NONE"})
	tStats := metrics.DudType{}

	conf := NewConfig()
	conf.JSON.Operator = "set"
	conf.JSON.Path = "foo.bar"
	conf.JSON.Value = []byte(`{"baz":1}`)

	exp := `{"foo":{"bar":{"baz":1}}}`

	tests := map[int]int{
		-3: 0,
		-2: 1,
		-1: 2,
		0:  0,
		1:  1,
		2:  2,
	}

	for i, j := range tests {
		input := [][]byte{
			[]byte(`{"foo":{"bar":2}}`),
			[]byte(`{"foo":{"bar":2}}`),
			[]byte(`{"foo":{"bar":2}}`),
		}

		conf.JSON.Parts = []int{i}
		proc, err := NewJSON(conf, nil, tLog, tStats)
		if err != nil {
			t.Fatal(err)
		}

		msgs, res := proc.ProcessMessage(message.New(input))
		if len(msgs) != 1 {
			t.Errorf("Select Parts failed on index: %v", i)
		} else if res != nil {
			t.Errorf("Expected nil response: %v", res)
		}
		if act := string(message.GetAllBytes(msgs[0])[j]); exp != act {
			t.Errorf("Unexpected output for index %v: %v != %v", i, act, exp)
		}
		if act := string(message.GetAllBytes(msgs[0])[(j+1)%3]); exp == act {
			t.Errorf("Processor was applied to wrong index %v: %v != %v", j+1%3, act, exp)
		}
	}
}

func TestJSONFlattenArray(t *testing.T) {
	type jTest struct {
		name   string
		path   string
		value  string
		input  string
		output string
	}

	tests := []jTest{
		{
			name:   "flatten ints 1",
			path:   "foo.bar",
			input:  `{"foo":{"bar":[0,[1,2],3,4]}}`,
			output: `{"foo":{"bar":[0,1,2,3,4]}}`,
		},
		{
			name:   "flatten numbers 1",
			path:   "foo.bar",
			input:  `{"foo":{"bar":[[0],[],1.5,[2,3,4]]}}`,
			output: `{"foo":{"bar":[0,1.5,2,3,4]}}`,
		},
		{
			name:   "flatten root numbers 1",
			path:   ".",
			input:  `[0,[1.5],2,[3],4]`,
			output: `[0,1.5,2,3,4]`,
		},
		{
			name:   "flatten strings 1",
			path:   "foo.bar",
			input:  `{"foo":{"bar":[["foo"],["bar","baz"]]}}`,
			output: `{"foo":{"bar":["foo","bar","baz"]}}`,
		},
		{
			name:   "flatten mixed 1",
			path:   "foo.bar",
			input:  `{"foo":{"bar":[["foo","bar"],[5],6,null]}}`,
			output: `{"foo":{"bar":["foo","bar",5,6,null]}}`,
		},
		{
			name:   "flatten empty",
			path:   "foo.bar",
			input:  `{"foo":{"bar":[]}}`,
			output: `{"foo":{"bar":[]}}`,
		},
	}

	for _, test := range tests {
		conf := NewConfig()
		conf.JSON.Operator = "flatten_array"
		conf.JSON.Parts = []int{0}
		conf.JSON.Path = test.path
		conf.JSON.Value = []byte(test.value)

		jSet, err := NewJSON(conf, nil, log.Noop(), metrics.Noop())
		if err != nil {
			t.Fatalf("Error for test '%v': %v", test.name, err)
		}

		inMsg := message.New(
			[][]byte{
				[]byte(test.input),
			},
		)
		msgs, _ := jSet.ProcessMessage(inMsg)
		if len(msgs) != 1 {
			t.Fatalf("Test '%v' did not succeed", test.name)
		}

		if exp, act := test.output, string(message.GetAllBytes(msgs[0])[0]); exp != act {
			t.Errorf("Wrong result '%v': %v != %v", test.name, act, exp)
		}
	}
}

func TestJSONFoldStringArray(t *testing.T) {
	type jTest struct {
		name   string
		path   string
		value  string
		input  string
		output string
	}

	tests := []jTest{
		{
			name:   "fold strings 1",
			path:   "foo.bar",
			input:  `{"foo":{"bar":["foo","bar","baz"]}}`,
			output: `{"foo":{"bar":"foobarbaz"}}`,
		},
		{
			name:   "fold strings 2",
			path:   "foo.bar",
			value:  `" "`,
			input:  `{"foo":{"bar":["foo","bar","baz"]}}`,
			output: `{"foo":{"bar":"foo bar baz"}}`,
		},
		{
			name:   "fold empty",
			path:   "foo.bar",
			input:  `{"foo":{"bar":[]}}`,
			output: `{"foo":{"bar":""}}`,
		},
	}

	for _, test := range tests {
		conf := NewConfig()
		conf.JSON.Operator = "fold_string_array"
		conf.JSON.Parts = []int{0}
		conf.JSON.Path = test.path
		conf.JSON.Value = []byte(test.value)

		jSet, err := NewJSON(conf, nil, log.Noop(), metrics.Noop())
		if err != nil {
			t.Fatalf("Error for test '%v': %v", test.name, err)
		}

		inMsg := message.New(
			[][]byte{
				[]byte(test.input),
			},
		)
		msgs, _ := jSet.ProcessMessage(inMsg)
		if len(msgs) != 1 {
			t.Fatalf("Test '%v' did not succeed", test.name)
		}

		if exp, act := test.output, string(message.GetAllBytes(msgs[0])[0]); exp != act {
			t.Errorf("Wrong result '%v': %v != %v", test.name, act, exp)
		}
	}
}

func TestJSONNumberArray(t *testing.T) {
	type jTest struct {
		name   string
		path   string
		value  string
		input  string
		output string
	}

	tests := []jTest{
		{
			name:   "fold ints 1",
			path:   "foo.bar",
			input:  `{"foo":{"bar":[0,1,2,3,4]}}`,
			output: `{"foo":{"bar":10}}`,
		},
		{
			name:   "fold numbers 1",
			path:   "foo.bar",
			input:  `{"foo":{"bar":[0,1.5,2,3,4]}}`,
			output: `{"foo":{"bar":10.5}}`,
		},
		{
			name:   "fold root numbers 1",
			path:   ".",
			input:  `[0,1.5,2,3,4]`,
			output: `10.5`,
		},
		{
			name:   "fold numbers empty",
			path:   "foo.bar",
			input:  `{"foo":{"bar":[]}}`,
			output: `{"foo":{"bar":0}}`,
		},
	}

	for _, test := range tests {
		conf := NewConfig()
		conf.JSON.Operator = "fold_number_array"
		conf.JSON.Parts = []int{0}
		conf.JSON.Path = test.path
		conf.JSON.Value = []byte(test.value)

		jSet, err := NewJSON(conf, nil, log.Noop(), metrics.Noop())
		if err != nil {
			t.Fatalf("Error for test '%v': %v", test.name, err)
		}

		inMsg := message.New(
			[][]byte{
				[]byte(test.input),
			},
		)
		msgs, _ := jSet.ProcessMessage(inMsg)
		if len(msgs) != 1 {
			t.Fatalf("Test '%v' did not succeed", test.name)
		}

		if exp, act := test.output, string(message.GetAllBytes(msgs[0])[0]); exp != act {
			t.Errorf("Wrong result '%v': %v != %v", test.name, act, exp)
		}
	}
}

func TestJSONFlatten(t *testing.T) {
	type jTest struct {
		name   string
		path   string
		input  string
		output string
	}

	tests := []jTest{
		{
			name:   "flatten 1",
			path:   ".",
			input:  `{"foo":{"bar":"baz"}}`,
			output: `{"foo.bar":"baz"}`,
		},
		{
			name:   "flatten 2",
			path:   ".",
			input:  `{"foo":[{"bar":"1"},{"bar":"2"}]}`,
			output: `{"foo.0.bar":"1","foo.1.bar":"2"}`,
		},
		{
			name:   "flatten 3",
			path:   "",
			input:  `[{"bar":"1"},{"bar":"2"}]`,
			output: `{"0.bar":"1","1.bar":"2"}`,
		},
		{
			name:   "flatten 4",
			path:   "",
			input:  `[["1"],["2","3"]]`,
			output: `{"0.0":"1","1.0":"2","1.1":"3"}`,
		},
		{
			name:   "flatten nested 1",
			path:   "inner",
			input:  `{"inner":{"foo":{"bar":"baz"}}}`,
			output: `{"inner":{"foo.bar":"baz"}}`,
		},
		{
			name:   "flatten nested 2",
			path:   "inner",
			input:  `{"also":"this","inner":{"foo":[{"bar":"1"},{"bar":"2"}]}}`,
			output: `{"also":"this","inner":{"foo.0.bar":"1","foo.1.bar":"2"}}`,
		},
	}

	for _, test := range tests {
		conf := NewConfig()
		conf.JSON.Operator = "flatten"
		conf.JSON.Parts = []int{0}
		conf.JSON.Path = test.path

		jSet, err := NewJSON(conf, nil, log.Noop(), metrics.Noop())
		if err != nil {
			t.Fatalf("Error for test '%v': %v", test.name, err)
		}

		inMsg := message.New(
			[][]byte{
				[]byte(test.input),
			},
		)
		msgs, _ := jSet.ProcessMessage(inMsg)
		if len(msgs) != 1 {
			t.Fatalf("Test '%v' did not succeed", test.name)
		}

		if exp, act := test.output, string(message.GetAllBytes(msgs[0])[0]); exp != act {
			t.Errorf("Wrong result '%v': %v != %v", test.name, act, exp)
		}
	}
}

func TestJSONAppend(t *testing.T) {
	tLog := log.New(os.Stdout, log.Config{LogLevel: "NONE"})
	tStats := metrics.DudType{}

	type jTest struct {
		name   string
		path   string
		value  string
		input  string
		output string
	}

	tests := []jTest{
		{
			name:   "append 1",
			path:   "foo.bar",
			value:  `{"baz":1}`,
			input:  `{"foo":{"bar":5}}`,
			output: `{"foo":{"bar":[5,{"baz":1}]}}`,
		},
		{
			name:   "append in array 1",
			path:   "foo.1.bar",
			value:  `{"baz":1}`,
			input:  `{"foo":[{"ignored":true},{"bar":5}]}`,
			output: `{"foo":[{"ignored":true},{"bar":[5,{"baz":1}]}]}`,
		},
		{
			name:   "append nil 1",
			path:   "foo.bar",
			value:  `{"baz":1}`,
			input:  `{"foo":{"bar":null}}`,
			output: `{"foo":{"bar":[null,{"baz":1}]}}`,
		},
		{
			name:   "append nil 2",
			path:   "foo.bar",
			value:  `{"baz":1}`,
			input:  `{"foo":{"bar":[null]}}`,
			output: `{"foo":{"bar":[null,{"baz":1}]}}`,
		},
		{
			name:   "append empty 1",
			path:   "foo.bar",
			value:  `{"baz":1}`,
			input:  `{"foo":{}}`,
			output: `{"foo":{"bar":[{"baz":1}]}}`,
		},
		{
			name:   "append collision 1",
			path:   "foo.bar",
			value:  `{"baz":1}`,
			input:  `{"foo":0}`,
			output: `{"foo":0}`,
		},
		{
			name:   "append array 1",
			path:   "foo.bar",
			value:  `[1,2,3]`,
			input:  `{"foo":{"bar":[0]}}`,
			output: `{"foo":{"bar":[0,1,2,3]}}`,
		},
		{
			name:   "append array 2",
			path:   "foo.bar",
			value:  `[1,2,3]`,
			input:  `{"foo":{"bar":0}}`,
			output: `{"foo":{"bar":[0,1,2,3]}}`,
		},
	}

	for _, test := range tests {
		conf := NewConfig()
		conf.JSON.Operator = "append"
		conf.JSON.Parts = []int{0}
		conf.JSON.Path = test.path
		conf.JSON.Value = []byte(test.value)

		jSet, err := NewJSON(conf, nil, tLog, tStats)
		if err != nil {
			t.Fatalf("Error for test '%v': %v", test.name, err)
		}

		inMsg := message.New(
			[][]byte{
				[]byte(test.input),
			},
		)
		msgs, _ := jSet.ProcessMessage(inMsg)
		if len(msgs) != 1 {
			t.Fatalf("Test '%v' did not succeed", test.name)
		}

		if exp, act := test.output, string(message.GetAllBytes(msgs[0])[0]); exp != act {
			t.Errorf("Wrong result '%v': %v != %v", test.name, act, exp)
		}
	}
}

func TestJSONSplit(t *testing.T) {
	type jTest struct {
		name   string
		path   string
		value  string
		input  string
		output string
	}

	tests := []jTest{
		{
			name:   "split 1",
			path:   "foo.bar",
			value:  `","`,
			input:  `{"foo":{"bar":"1,2,3"}}`,
			output: `{"foo":{"bar":["1","2","3"]}}`,
		},
		{
			name:   "split 2",
			path:   "foo.bar",
			value:  `"-"`,
			input:  `{"foo":{"bar":"1-2-3"}}`,
			output: `{"foo":{"bar":["1","2","3"]}}`,
		},
		{
			name:   "split 3",
			path:   "foo.bar",
			value:  `"-"`,
			input:  `{"foo":{"bar":20}}`,
			output: `{"foo":{"bar":20}}`,
		},
		{
			name:   "split 4",
			path:   "foo.bar",
			value:  `","`,
			input:  `{"foo":{"bar":"1"}}`,
			output: `{"foo":{"bar":["1"]}}`,
		},
		{
			name:   "split 5",
			path:   "foo.bar",
			value:  `","`,
			input:  `{"foo":{"bar":","}}`,
			output: `{"foo":{"bar":["",""]}}`,
		},
	}

	for _, test := range tests {
		conf := NewConfig()
		conf.JSON.Operator = "split"
		conf.JSON.Parts = []int{0}
		conf.JSON.Path = test.path
		conf.JSON.Value = []byte(test.value)

		jSet, err := NewJSON(conf, nil, log.Noop(), metrics.Noop())
		if err != nil {
			t.Fatalf("Error for test '%v': %v", test.name, err)
		}

		inMsg := message.New(
			[][]byte{
				[]byte(test.input),
			},
		)
		msgs, _ := jSet.ProcessMessage(inMsg)
		if len(msgs) != 1 {
			t.Fatalf("Test '%v' did not succeed", test.name)
		}

		if exp, act := test.output, string(message.GetAllBytes(msgs[0])[0]); exp != act {
			t.Errorf("Wrong result '%v': %v != %v", test.name, act, exp)
		}
	}
}

func TestJSONMove(t *testing.T) {
	type jTest struct {
		name   string
		path   string
		value  string
		input  string
		output string
	}

	tests := []jTest{
		{
			name:   "move 1",
			path:   "foo.bar",
			value:  `"bar.baz"`,
			input:  `{"foo":{"bar":5}}`,
			output: `{"bar":{"baz":5},"foo":{}}`,
		},
		{
			name:   "move 2",
			path:   "foo.bar",
			value:  `"bar.baz"`,
			input:  `{"foo":{"bar":5},"bar":{"qux":6}}`,
			output: `{"bar":{"baz":5,"qux":6},"foo":{}}`,
		},
		{
			name:   "move to same path 1",
			path:   "foo.bar",
			value:  `"foo.bar"`,
			input:  `{"foo":{"bar":5},"bar":{"qux":6}}`,
			output: `{"bar":{"qux":6},"foo":{"bar":5}}`,
		},
		{
			name:   "move from root 1",
			path:   ".",
			value:  `"bar.baz"`,
			input:  `{"foo":{"bar":5}}`,
			output: `{"bar":{"baz":{"foo":{"bar":5}}}}`,
		},
		{
			name:   "move to root 1",
			path:   "foo",
			value:  `""`,
			input:  `{"foo":{"bar":5}}`,
			output: `{"bar":5}`,
		},
	}

	for _, test := range tests {
		conf := NewConfig()
		conf.JSON.Operator = "move"
		conf.JSON.Parts = []int{0}
		conf.JSON.Path = test.path
		conf.JSON.Value = []byte(test.value)

		jSet, err := NewJSON(conf, nil, log.Noop(), metrics.Noop())
		if err != nil {
			t.Fatalf("Error for test '%v': %v", test.name, err)
		}

		inMsg := message.New(
			[][]byte{
				[]byte(test.input),
			},
		)
		msgs, _ := jSet.ProcessMessage(inMsg)
		if len(msgs) != 1 {
			t.Fatalf("Test '%v' did not succeed", test.name)
		}

		if exp, act := test.output, string(message.GetAllBytes(msgs[0])[0]); exp != act {
			t.Errorf("Wrong result '%v': %v != %v", test.name, act, exp)
		}
	}
}

func TestJSONExplode(t *testing.T) {
	type jTest struct {
		name   string
		path   string
		input  string
		output string
	}

	tests := []jTest{
		{
			name:   "explode 1",
			path:   "foo",
			input:  `{"foo":[1,2,3],"id":"bar"}`,
			output: `[{"foo":1,"id":"bar"},{"foo":2,"id":"bar"},{"foo":3,"id":"bar"}]`,
		},
		{
			name:   "explode 2",
			path:   "foo.bar",
			input:  `{"foo":{"also":"this","bar":[{"key":"value1"},{"key":"value2"},{"key":"value3"}]},"id":"baz"}`,
			output: `[{"foo":{"also":"this","bar":{"key":"value1"}},"id":"baz"},{"foo":{"also":"this","bar":{"key":"value2"}},"id":"baz"},{"foo":{"also":"this","bar":{"key":"value3"}},"id":"baz"}]`,
		},
		{
			name:   "explode 3",
			path:   "foo",
			input:  `{"foo":{"a":1,"b":2,"c":3},"id":"bar"}`,
			output: `{"a":{"foo":1,"id":"bar"},"b":{"foo":2,"id":"bar"},"c":{"foo":3,"id":"bar"}}`,
		},
		{
			name:   "explode 4",
			path:   "foo.bar",
			input:  `{"foo":{"also":"this","bar":{"key1":["a","b"],"key2":{"c":3,"d":4}}},"id":"baz"}`,
			output: `{"key1":{"foo":{"also":"this","bar":["a","b"]},"id":"baz"},"key2":{"foo":{"also":"this","bar":{"c":3,"d":4}},"id":"baz"}}`,
		},
	}

	for _, test := range tests {
		conf := NewConfig()
		conf.JSON.Operator = "explode"
		conf.JSON.Parts = []int{0}
		conf.JSON.Path = test.path

		jSet, err := NewJSON(conf, nil, log.Noop(), metrics.Noop())
		if err != nil {
			t.Fatalf("Error for test '%v': %v", test.name, err)
		}

		inMsg := message.New(
			[][]byte{
				[]byte(test.input),
			},
		)
		msgs, _ := jSet.ProcessMessage(inMsg)
		if len(msgs) != 1 {
			t.Fatalf("Test '%v' did not succeed", test.name)
		}

		if exp, act := test.output, string(message.GetAllBytes(msgs[0])[0]); exp != act {
			t.Errorf("Wrong result '%v': %v != %v", test.name, act, exp)
		}
	}
}

func TestJSONCopy(t *testing.T) {
	tLog := log.New(os.Stdout, log.Config{LogLevel: "NONE"})
	tStats := metrics.DudType{}

	type jTest struct {
		name   string
		path   string
		value  string
		input  string
		output string
	}

	tests := []jTest{
		{
			name:   "copy 1",
			path:   "foo.bar",
			value:  `"bar.baz"`,
			input:  `{"foo":{"bar":5}}`,
			output: `{"bar":{"baz":5},"foo":{"bar":5}}`,
		},
		{
			name:   "copy 2",
			path:   "foo.bar",
			value:  `"bar.baz"`,
			input:  `{"foo":{"bar":5},"bar":{"qux":6}}`,
			output: `{"bar":{"baz":5,"qux":6},"foo":{"bar":5}}`,
		},
	}

	for _, test := range tests {
		conf := NewConfig()
		conf.JSON.Operator = "copy"
		conf.JSON.Parts = []int{0}
		conf.JSON.Path = test.path
		conf.JSON.Value = []byte(test.value)

		jSet, err := NewJSON(conf, nil, tLog, tStats)
		if err != nil {
			t.Fatalf("Error for test '%v': %v", test.name, err)
		}

		inMsg := message.New(
			[][]byte{
				[]byte(test.input),
			},
		)
		msgs, _ := jSet.ProcessMessage(inMsg)
		if len(msgs) != 1 {
			t.Fatalf("Test '%v' did not succeed", test.name)
		}

		if exp, act := test.output, string(message.GetAllBytes(msgs[0])[0]); exp != act {
			t.Errorf("Wrong result '%v': %v != %v", test.name, act, exp)
		}
	}
}

func TestJSONClean(t *testing.T) {
	tLog := log.New(os.Stdout, log.Config{LogLevel: "NONE"})
	tStats := metrics.DudType{}

	type jTest struct {
		name   string
		path   string
		input  string
		output string
	}

	tests := []jTest{
		{
			name:   "clean nothing",
			path:   "foo.bar",
			input:  `{"foo":{"bar":5}}`,
			output: `{"foo":{"bar":5}}`,
		},
		{
			name:   "clean array",
			path:   "foo.bar",
			input:  `{"foo":{"bar":[]}}`,
			output: `{"foo":{}}`,
		},
		{
			name:   "clean array 2",
			path:   "foo.bar",
			input:  `{"foo":{"b":[1],"bar":[]}}`,
			output: `{"foo":{"b":[1]}}`,
		},
		{
			name:   "clean array 3",
			path:   "foo",
			input:  `{"foo":{"b":[1],"bar":[]}}`,
			output: `{"foo":{"b":[1]}}`,
		},
		{
			name:   "clean object",
			path:   "foo.bar",
			input:  `{"foo":{"bar":{}}}`,
			output: `{"foo":{}}`,
		},
		{
			name:   "clean object 2",
			path:   "foo.bar",
			input:  `{"foo":{"b":{"1":1},"bar":{}}}`,
			output: `{"foo":{"b":{"1":1}}}`,
		},
		{
			name:   "clean object 3",
			path:   "foo",
			input:  `{"foo":{"b":{"1":1},"bar":{}}}`,
			output: `{"foo":{"b":{"1":1}}}`,
		},
		{
			name:   "clean array from root",
			path:   "",
			input:  `{"foo":{"b":"b","bar":[]}}`,
			output: `{"foo":{"b":"b"}}`,
		},
		{
			name:   "clean object from root",
			path:   "",
			input:  `{"foo":{"b":"b","bar":{}}}`,
			output: `{"foo":{"b":"b"}}`,
		},
		{
			name:   "clean everything object",
			path:   "",
			input:  `{"foo":{"bar":{}}}`,
			output: `{}`,
		},
		{
			name:   "clean everything array",
			path:   "",
			input:  `[{"foo":{"bar":{}}},[]]`,
			output: `[]`,
		},
		{
			name:   "clean everything string",
			path:   "",
			input:  `""`,
			output: `null`,
		},
		{
			name:   "clean arrays",
			path:   "",
			input:  `[[],1,"",2,{},"test",{"foo":{}}]`,
			output: `[1,2,"test"]`,
		},
	}

	for _, test := range tests {
		conf := NewConfig()
		conf.JSON.Operator = "clean"
		conf.JSON.Parts = []int{0}
		conf.JSON.Path = test.path

		jSet, err := NewJSON(conf, nil, tLog, tStats)
		if err != nil {
			t.Fatalf("Error for test '%v': %v", test.name, err)
		}

		inMsg := message.New(
			[][]byte{
				[]byte(test.input),
			},
		)
		msgs, _ := jSet.ProcessMessage(inMsg)
		if len(msgs) != 1 {
			t.Fatalf("Test '%v' did not succeed", test.name)
		}

		if exp, act := test.output, string(message.GetAllBytes(msgs[0])[0]); exp != act {
			t.Errorf("Wrong result '%v': %v != %v", test.name, act, exp)
		}
	}
}

func TestJSONSet(t *testing.T) {
	type jTest struct {
		name   string
		path   string
		value  string
		input  string
		output string
	}

	tests := []jTest{
		{
			name:   "set 1",
			path:   "foo.bar",
			value:  `{"baz":1}`,
			input:  `{"foo":{"bar":5}}`,
			output: `{"foo":{"bar":{"baz":1}}}`,
		},
		{
			name:   "set 2",
			path:   "foo",
			value:  `5`,
			input:  `{"foo":{"bar":5}}`,
			output: `{"foo":5}`,
		},
		{
			name:   "set 3",
			path:   "foo",
			value:  `"5"`,
			input:  `{"foo":{"bar":5}}`,
			output: `{"foo":"5"}`,
		},
		{
			name: "set 4",
			path: "foo.bar",
			value: `{
					"baz": 1
				}`,
			input:  `{"foo":{"bar":5}}`,
			output: `{"foo":{"bar":{"baz":1}}}`,
		},
		{
			name:   "set 5",
			path:   "foo.bar",
			value:  `{"baz":"${!echo:foo}"}`,
			input:  `{"foo":{"bar":5}}`,
			output: `{"foo":{"bar":{"baz":"foo"}}}`,
		},
		{
			name:   "set 6",
			path:   "foo.bar",
			value:  `${!echo:10}`,
			input:  `{"foo":{"bar":5}}`,
			output: `{"foo":{"bar":10}}`,
		},
		{
			name:   "set root 1",
			path:   "",
			value:  `{"baz":1}`,
			input:  `"hello world"`,
			output: `{"baz":1}`,
		},
		{
			name:   "set root 2",
			path:   ".",
			value:  `{"baz":1}`,
			input:  `{"foo":2}`,
			output: `{"baz":1}`,
		},
		{
			name:   "set interpolate 1",
			path:   "foo",
			value:  `{"baz":"${!json("bar")}"}`,
			input:  `{"foo":2,"bar":"hello world this is a string"}`,
			output: `{"bar":"hello world this is a string","foo":{"baz":"hello world this is a string"}}`,
		},
		{
			name:   "set interpolate 2",
			path:   ".",
			value:  `{"${!json("key")}":{"value":"${!json("value")}"}}`,
			input:  `{"key":"dynamic","value":{"foo":"bar"}}`,
			output: `{"dynamic":{"value":"{\"foo\":\"bar\"}"}}`,
		},
		{
			name:   "set null 1",
			path:   "foo.bar",
			value:  `null`,
			input:  `{"foo":{"bar":5}}`,
			output: `{"foo":{"bar":null}}`,
		},
		{
			name:   "set null 2",
			path:   "foo.bar",
			value:  `null`,
			input:  `{"foo":{"bar":{"baz":"yelp"}}}`,
			output: `{"foo":{"bar":null}}`,
		},
		{
			name:   "set unicode 1",
			path:   "foo.bar",
			value:  `"contains 🦄 emoji"`,
			input:  `{"foo":{"bar":{"baz":"yelp"}}}`,
			output: `{"foo":{"bar":"contains 🦄 emoji"}}`,
		},
		{
			name:   "set unicode 2",
			path:   "foo.bar",
			value:  `{"value":{"unicode":"contains 🦄 emoji"}}`,
			input:  `{"foo":{"bar":{"baz":"yelp"}}}`,
			output: `{"foo":{"bar":{"value":{"unicode":"contains 🦄 emoji"}}}}`,
		},
		{
			name:   "set unicode 3",
			path:   "foo.bar",
			value:  `{"value":"${!json("foo.bar.baz")}"}`,
			input:  `{"foo":{"bar":{"baz":"foo 🦄 bar"}}}`,
			output: `{"foo":{"bar":{"value":"foo 🦄 bar"}}}`,
		},
	}

	for _, test := range tests {
		conf := NewConfig()
		conf.JSON.Operator = "set"
		conf.JSON.Parts = []int{0}
		conf.JSON.Path = test.path
		conf.JSON.Value = []byte(test.value)

		jSet, err := NewJSON(conf, nil, log.Noop(), metrics.Noop())
		if err != nil {
			t.Fatalf("Error for test '%v': %v", test.name, err)
		}

		inMsg := message.New(
			[][]byte{
				[]byte(test.input),
			},
		)
		msgs, _ := jSet.ProcessMessage(inMsg)
		if len(msgs) != 1 {
			t.Fatalf("Test '%v' did not succeed", test.name)
		}

		if exp, act := test.output, string(message.GetAllBytes(msgs[0])[0]); exp != act {
			t.Errorf("Wrong result '%v': %v != %v", test.name, act, exp)
		}
	}
}

func TestJSONSetEdge(t *testing.T) {
	conf := NewConfig()
	conf.JSON.Operator = "set"
	conf.JSON.Path = "foo"
	conf.JSON.Value = []byte(`"bar"`)

	jSet, err := NewJSON(conf, nil, log.Noop(), metrics.Noop())
	if err != nil {
		t.Fatal(err)
	}

	inMsg := message.New([][]byte{[]byte(`{}`)})
	msgs, _ := jSet.ProcessMessage(inMsg)
	if len(msgs) != 1 {
		t.Fatalf("Wrong count of result messages: %v", len(msgs))
	}
	if exp, act := `{"foo":"bar"}`, string(msgs[0].Get(0).Get()); exp != act {
		t.Errorf("Wrong result: %v != %v", act, exp)
	}

	msgs, _ = jSet.ProcessMessage(msgs[0])
	if len(msgs) != 1 {
		t.Fatalf("Wrong count of result messages: %v", len(msgs))
	}
	if exp, act := `{"foo":"bar"}`, string(msgs[0].Get(0).Get()); exp != act {
		t.Errorf("Wrong result: %v != %v", act, exp)
	}
}

func TestJSONConfigYAML(t *testing.T) {
	tLog := log.New(os.Stdout, log.Config{LogLevel: "NONE"})
	tStats := metrics.DudType{}

	input := `{"foo":{"bar":5}}`

	tests := map[string]string{
		`value: 10`:            `{"foo":{"bar":10}}`,
		`value: "hello world"`: `{"foo":{"bar":"hello world"}}`,
		`value: hello world`:   `{"foo":{"bar":"hello world"}}`,
		`
value:
  baz: 10`: `{"foo":{"bar":{"baz":10}}}`,
		`
value:
  baz:
    - first
    - 2
    - third`: `{"foo":{"bar":{"baz":["first",2,"third"]}}}`,
		`
value:
  baz:
    deeper: look at me
  here: 11`: `{"foo":{"bar":{"baz":{"deeper":"look at me"},"here":11}}}`,
	}

	for config, exp := range tests {
		conf := NewConfig()
		conf.JSON.Operator = "set"
		conf.JSON.Parts = []int{}
		conf.JSON.Path = "foo.bar"

		if err := yaml.Unmarshal([]byte(config), &conf.JSON); err != nil {
			t.Fatal(err)
		}

		jSet, err := NewJSON(conf, nil, tLog, tStats)
		if err != nil {
			t.Fatalf("Error creating proc '%v': %v", config, err)
		}

		inMsg := message.New(
			[][]byte{
				[]byte(input),
			},
		)
		msgs, _ := jSet.ProcessMessage(inMsg)
		if len(msgs) != 1 {
			t.Fatalf("Test did not succeed with config: %v", config)
		}

		if act := string(message.GetAllBytes(msgs[0])[0]); exp != act {
			t.Errorf("Wrong result '%v': %v != %v", config, act, exp)
		}
	}
}

func TestJSONConfigYAMLMarshal(t *testing.T) {
	tLog := log.New(os.Stdout, log.Config{LogLevel: "NONE"})
	tStats := metrics.DudType{}

	tests := []string{
		`parts:
  - 0
operator: set
path: foo.bar
value:
  baz:
    deeper: look at me
  here: 11
`,
		`parts:
  - 0
operator: set
path: foo.bar
value: null
`,
		`parts:
  - 0
operator: set
path: foo.bar
value:
  foo: null
`,
		`parts:
  - 0
operator: set
path: foo.bar
value:
  baz:
    deeper:
      - first
      - second
      - third
  here: 11
`,
		`parts:
  - 5
operator: set
path: foo.bar.baz
value: 5
`,
		`parts:
  - 0
operator: set
path: foo.bar
value: hello world
`,
		`parts:
  - 0
operator: set
path: foo.bar
value:
  root:
    - values:
        - nested: true
      with: array
`,
		`parts:
  - 0
operator: set
path: foo.bar
value:
  foo:
    bar:
      baz:
        value: true
`,
	}

	for _, testconfig := range tests {
		conf := NewConfig()
		if err := yaml.Unmarshal([]byte(testconfig), &conf.JSON); err != nil {
			t.Error(err)
			continue
		}

		if act, err := config.MarshalYAML(conf.JSON); err != nil {
			t.Error(err)
		} else if string(act) != testconfig {
			t.Errorf("Marshalled config does not match: %v != %v", string(act), testconfig)
		}

		if _, err := NewJSON(conf, nil, tLog, tStats); err != nil {
			t.Errorf("Error creating proc '%v': %v", testconfig, err)
		}
	}
}

func TestJSONSelect(t *testing.T) {
	tLog := log.New(os.Stdout, log.Config{LogLevel: "NONE"})
	tStats := metrics.DudType{}

	type jTest struct {
		name   string
		path   string
		input  string
		output string
	}

	tests := []jTest{
		{
			name:   "select obj",
			path:   "foo.bar",
			input:  `{"foo":{"bar":{"baz":1}}}`,
			output: `{"baz":1}`,
		},
		{
			name:   "select array",
			path:   "foo.bar",
			input:  `{"foo":{"bar":["baz","qux"]}}`,
			output: `["baz","qux"]`,
		},
		{
			name:   "select obj as str",
			path:   "foo.bar",
			input:  `{"foo":{"bar":"{\"baz\":1}"}}`,
			output: `{"baz":1}`,
		},
		{
			name:   "select str",
			path:   "foo.bar",
			input:  `{"foo":{"bar":"hello world"}}`,
			output: `hello world`,
		},
		{
			name:   "select float",
			path:   "foo.bar",
			input:  `{"foo":{"bar":0.123}}`,
			output: `0.123`,
		},
		{
			name:   "select int",
			path:   "foo.bar",
			input:  `{"foo":{"bar":123}}`,
			output: `123`,
		},
		{
			name:   "select bool",
			path:   "foo.bar",
			input:  `{"foo":{"bar":true}}`,
			output: `true`,
		},
	}

	for _, test := range tests {
		conf := NewConfig()
		conf.JSON.Operator = "select"
		conf.JSON.Parts = []int{0}
		conf.JSON.Path = test.path

		jSet, err := NewJSON(conf, nil, tLog, tStats)
		if err != nil {
			t.Fatalf("Error for test '%v': %v", test.name, err)
		}

		inMsg := message.New(
			[][]byte{
				[]byte(test.input),
			},
		)
		msgs, _ := jSet.ProcessMessage(inMsg)
		if len(msgs) != 1 {
			t.Fatalf("Test '%v' did not succeed", test.name)
		}

		if exp, act := test.output, string(message.GetAllBytes(msgs[0])[0]); exp != act {
			t.Errorf("Wrong result '%v': %v != %v", test.name, act, exp)
		}
	}
}

func TestJSONDeletePartBounds(t *testing.T) {
	tLog := log.New(os.Stdout, log.Config{LogLevel: "NONE"})
	tStats := metrics.DudType{}

	conf := NewConfig()
	conf.JSON.Path = "foo.bar"
	conf.JSON.Operator = "delete"

	exp := `{"foo":{}}`

	tests := map[int]int{
		-3: 0,
		-2: 1,
		-1: 2,
		0:  0,
		1:  1,
		2:  2,
	}

	for i, j := range tests {
		input := [][]byte{
			[]byte(`{"foo":{"bar":2}}`),
			[]byte(`{"foo":{"bar":2}}`),
			[]byte(`{"foo":{"bar":2}}`),
		}

		conf.JSON.Parts = []int{i}
		proc, err := NewJSON(conf, nil, tLog, tStats)
		if err != nil {
			t.Fatal(err)
		}

		msgs, res := proc.ProcessMessage(message.New(input))
		if len(msgs) != 1 {
			t.Errorf("Select Parts failed on index: %v", i)
		} else if res != nil {
			t.Errorf("Expected nil response: %v", res)
		}
		if act := string(message.GetAllBytes(msgs[0])[j]); exp != act {
			t.Errorf("Unexpected output for index %v: %v != %v", i, act, exp)
		}
	}
}

func TestJSONDelete(t *testing.T) {
	tLog := log.New(os.Stdout, log.Config{LogLevel: "NONE"})
	tStats := metrics.DudType{}

	type jTest struct {
		name   string
		path   string
		input  string
		output string
	}

	tests := []jTest{
		{
			name:   "del field 1",
			path:   "foo.bar",
			input:  `{"foo":{"bar":5}}`,
			output: `{"foo":{}}`,
		},
		{
			name:   "del obj field 1",
			path:   "foo.bar",
			input:  `{"foo":{"bar":{"baz":5}}}`,
			output: `{"foo":{}}`,
		},
		{
			name:   "del array field 1",
			path:   "foo.bar",
			input:  `{"foo":{"bar":[5]}}`,
			output: `{"foo":{}}`,
		},
	}

	for _, test := range tests {
		conf := NewConfig()
		conf.JSON.Parts = []int{0}
		conf.JSON.Operator = "delete"
		conf.JSON.Path = test.path

		jSet, err := NewJSON(conf, nil, tLog, tStats)
		if err != nil {
			t.Fatalf("Error for test '%v': %v", test.name, err)
		}

		inMsg := message.New(
			[][]byte{
				[]byte(test.input),
			},
		)
		msgs, _ := jSet.ProcessMessage(inMsg)
		if len(msgs) != 1 {
			t.Fatalf("Test '%v' did not succeed", test.name)
		}

		if exp, act := test.output, string(message.GetAllBytes(msgs[0])[0]); exp != act {
			t.Errorf("Wrong result '%v': %v != %v", test.name, act, exp)
		}
	}
}
