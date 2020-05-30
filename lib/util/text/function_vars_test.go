package text

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/windhooked/benthos/v3/lib/message"
	"github.com/windhooked/benthos/v3/lib/types"
)

func TestFunctionVarDetection(t *testing.T) {
	tests := map[string]bool{
		"foo ${!foo_bar} baz":                          true,
		"foo ${!foo_bar} baz ${!foo_baz}":              true,
		"foo $!foo} baz $!but_not_this}":               false,
		"foo ${!baz ${!or_this":                        false,
		"foo ${baz} ${or_this}":                        false,
		"nothing $ here boss {!}":                      false,
		"foo ${!foo_bar:arg1} baz":                     true,
		"foo ${!foo_bar:} baz":                         false,
		"foo ${!foo_bar:arg1} baz ${!foo_baz:arg2}":    true,
		"foo $!foo:arg2} baz $!but_not_this:}":         false,
		"nothing $ here boss {!:argnope}":              false,
		"foo ${{!foo_bar}} baz":                        true,
		"foo ${{!foo_bar:default}} baz":                true,
		"foo ${{!foo_bar:default} baz":                 false,
		"foo {{!foo_bar:default}} baz":                 false,
		"foo {{!}} baz":                                false,
		"foo ${!foo_bar:arg1,} baz ${!foo_baz:arg2,5}": true,
	}

	for in, exp := range tests {
		act := ContainsFunctionVariables([]byte(in))
		if act != exp {
			t.Errorf("Wrong result for '%v': %v != %v", in, act, exp)
		}
	}
}

func TestMetadataFunction(t *testing.T) {
	msg := message.New([][]byte{[]byte("foo")})
	msg.Get(0).Metadata().Set("foo", "bar")
	msg.Get(0).Metadata().Set("baz", "qux")
	msg.Get(0).Metadata().Set("duck,1", "quack")

	act := string(ReplaceFunctionVariables(
		msg, []byte(`foo ${!metadata:foo} baz`),
	))
	if exp := "foo bar baz"; act != exp {
		t.Errorf("Wrong result: %v != %v", act, exp)
	}

	act = string(ReplaceFunctionVariables(
		msg, []byte(`foo ${!metadata:bar} baz`),
	))
	if exp := "foo  baz"; act != exp {
		t.Errorf("Wrong result: %v != %v", act, exp)
	}

	act = string(ReplaceFunctionVariables(
		msg, []byte(`foo ${!metadata} bar`),
	))
	if exp := `foo  bar`; act != exp {
		t.Errorf("Wrong result: %v != %v", act, exp)
	}

	act = string(ReplaceFunctionVariables(
		msg, []byte(`foo ${!metadata:duck,1,} baz`),
	))
	if exp := "foo quack baz"; act != exp {
		t.Errorf("Wrong result: %v != %v", act, exp)
	}
}

func TestErrorFunction(t *testing.T) {
	msg := message.New([][]byte{[]byte("foo"), []byte("bar")})
	msg.Get(0).Metadata().Set(types.FailFlagKey, "test error")

	act := string(ReplaceFunctionVariables(
		msg, []byte(`foo ${!error} baz`),
	))
	if exp := "foo test error baz"; act != exp {
		t.Errorf("Wrong result: %v != %v", act, exp)
	}

	act = string(ReplaceFunctionVariables(
		msg, []byte(`foo ${!error:1} baz`),
	))
	if exp := "foo  baz"; act != exp {
		t.Errorf("Wrong result: %v != %v", act, exp)
	}
}

func TestFunctionEscaped(t *testing.T) {
	msg := message.New([][]byte{[]byte(`{}`)})
	msg.Get(0).Metadata().Set("foo", `{"foo":"bar"}`)

	act := string(ReplaceFunctionVariablesEscaped(
		msg, []byte(`{"metadata":"${!metadata:foo}"}`),
	))
	if exp := `{"metadata":"{\"foo\":\"bar\"}"}`; act != exp {
		t.Errorf("Wrong result: %v != %v", act, exp)
	}

	act = string(ReplaceFunctionVariablesEscaped(
		msg, []byte(`"${!metadata:foo}"`),
	))
	if exp := `"{\"foo\":\"bar\"}"`; act != exp {
		t.Errorf("Wrong result: %v != %v", act, exp)
	}

	act = string(ReplaceFunctionVariablesEscaped(
		msg, []byte(`"${!metadata_json_object}"`),
	))
	if exp := `"{\"foo\":\"{\\\"foo\\\":\\\"bar\\\"}\"}"`; act != exp {
		t.Errorf("Wrong result: %v != %v", act, exp)
	}

	act = string(ReplaceFunctionVariablesEscaped(
		msg, []byte(`"${!metadata:bar}"`),
	))
	if exp := `""`; act != exp {
		t.Errorf("Wrong result: %v != %v", act, exp)
	}
}

func TestMetadataFunctionIndex(t *testing.T) {
	msg := message.New([][]byte{
		[]byte("foo"),
		[]byte("bar"),
	})
	msg.Get(0).Metadata().Set("foo", "bar")
	msg.Get(1).Metadata().Set("foo", "bar2")

	act := string(ReplaceFunctionVariables(
		msg, []byte(`foo ${!metadata:foo,0} baz`),
	))
	if exp := "foo bar baz"; act != exp {
		t.Errorf("Wrong result: %v != %v", act, exp)
	}

	act = string(ReplaceFunctionVariables(
		msg, []byte(`foo ${!metadata:foo,1} baz`),
	))
	if exp := "foo bar2 baz"; act != exp {
		t.Errorf("Wrong result: %v != %v", act, exp)
	}
}

func TestMetadataMapFunction(t *testing.T) {
	msg := message.New([][]byte{
		[]byte("foo"),
		[]byte("bar"),
	})
	msg.Get(0).Metadata().Set("foo", "bar")
	msg.Get(0).Metadata().Set("bar", "baz")
	msg.Get(1).Metadata().Set("foo", "bar2")

	act := string(ReplaceFunctionVariables(
		msg, []byte(`foo ${!metadata_json_object} baz`),
	))
	if exp := `foo {"bar":"baz","foo":"bar"} baz`; act != exp {
		t.Errorf("Wrong result: %v != %v", act, exp)
	}

	act = string(ReplaceFunctionVariables(
		msg, []byte(`foo ${!metadata_json_object:0} baz`),
	))
	if exp := `foo {"bar":"baz","foo":"bar"} baz`; act != exp {
		t.Errorf("Wrong result: %v != %v", act, exp)
	}

	act = string(ReplaceFunctionVariables(
		msg, []byte(`foo ${!metadata_json_object:1} baz`),
	))
	if exp := `foo {"foo":"bar2"} baz`; act != exp {
		t.Errorf("Wrong result: %v != %v", act, exp)
	}
}

func TestContentFunctionIndex(t *testing.T) {
	msg := message.New([][]byte{
		[]byte("foo"),
		[]byte("bar"),
	})

	act := string(ReplaceFunctionVariables(
		msg, []byte(`${!content:0} bar baz`),
	))
	if exp := "foo bar baz"; act != exp {
		t.Errorf("Wrong result: %v != %v", act, exp)
	}

	act = string(ReplaceFunctionVariables(
		msg, []byte(`foo ${!content:1} baz`),
	))
	if exp := "foo bar baz"; act != exp {
		t.Errorf("Wrong result: %v != %v", act, exp)
	}
}

func TestBatchSizeFunction(t *testing.T) {
	act := string(ReplaceFunctionVariables(
		message.New(make([][]byte, 0)), []byte(`${!batch_size} bar baz`),
	))
	if exp := "0 bar baz"; act != exp {
		t.Errorf("Wrong result: %v != %v", act, exp)
	}

	act = string(ReplaceFunctionVariables(
		message.New(make([][]byte, 1)), []byte(`${!batch_size} bar baz`),
	))
	if exp := "1 bar baz"; act != exp {
		t.Errorf("Wrong result: %v != %v", act, exp)
	}

	act = string(ReplaceFunctionVariables(
		message.New(make([][]byte, 2)), []byte(`${!batch_size} bar baz`),
	))
	if exp := "2 bar baz"; act != exp {
		t.Errorf("Wrong result: %v != %v", act, exp)
	}
}

func TestJSONFunction(t *testing.T) {
	type testCase struct {
		name   string
		input  []string
		arg    string
		result string
	}

	tests := []testCase{
		{
			name: "json func 1",
			input: []string{
				`{"foo":{"bar":"baz"}}`,
			},
			arg:    "foo ${!json_field:foo.bar,0} baz",
			result: "foo baz baz",
		},
		{
			name: "json func 2",
			input: []string{
				`{"foo":{"bar":"baz"}}`,
			},
			arg:    "foo ${!json_field:foo.bar,1} baz",
			result: "foo null baz",
		},
		{
			name: "json func 3",
			input: []string{
				`{"foo":{"bar":"baz"}}`,
			},
			arg:    "foo ${!json_field:foo.baz,0} baz",
			result: "foo null baz",
		},
		{
			name: "json func 4",
			input: []string{
				`{"foo":{"bar":{"baz":1}}}`,
			},
			arg:    "foo ${!json_field:foo.bar,0} baz",
			result: `foo {"baz":1} baz`,
		},
		{
			name: "json func 5",
			input: []string{
				`{"foo":{"bar":{"baz":1}}}`,
			},
			arg:    "foo ${!json_field:foo.bar,0} baz",
			result: `foo {"baz":1} baz`,
		},
		{
			name: "json func 6",
			input: []string{
				`{"foo":{"bar":5}}`,
			},
			arg:    "foo ${!json_field:foo.bar} baz",
			result: `foo 5 baz`,
		},
		{
			name: "json func 7",
			input: []string{
				`{"foo":{"bar":false}}`,
			},
			arg:    "foo ${!json_field:foo.bar} baz",
			result: `foo false baz`,
		},
		{
			name: "json func 8",
			input: []string{
				`{"foo":{"bar,1":false}}`,
			},
			arg:    "foo ${!json_field:foo.bar,1,} baz",
			result: `foo false baz`,
		},
		{
			name: "json func 9",
			input: []string{
				`{"foo":{"bar,1,":false}}`,
			},
			arg:    "foo ${!json_field:foo.bar,1,,} baz",
			result: `foo false baz`,
		},
		{
			name: "json func 10",
			input: []string{
				`{"foo":{"bar,1,test":false}}`,
			},
			arg:    "foo ${!json_field:foo.bar,1,test,} baz",
			result: `foo false baz`,
		},
	}

	for _, test := range tests {
		exp := test.result
		parts := [][]byte{}
		for _, input := range test.input {
			parts = append(parts, []byte(input))
		}
		act := string(ReplaceFunctionVariables(
			message.New(parts),
			[]byte(test.arg),
		))
		if act != exp {
			t.Errorf("Wrong result for test '%v': %v != %v", test.name, act, exp)
		}
	}
}

func TestJSONFunctionFor(t *testing.T) {
	type testCase struct {
		name   string
		input  []string
		index  int
		arg    string
		result string
	}

	tests := []testCase{
		{
			name: "json func 1",
			input: []string{
				`{"foo":{"bar":"baz1"}}`,
				`{"foo":{"bar":"baz2"}}`,
				`{"foo":{"bar":"baz3"}}`,
			},
			index:  0,
			arg:    "foo.bar: ${!json_field:foo.bar}",
			result: "foo.bar: baz1",
		},
		{
			name: "json func 2",
			input: []string{
				`{"foo":{"bar":"baz1"}}`,
				`{"foo":{"bar":"baz2"}}`,
				`{"foo":{"bar":"baz3"}}`,
			},
			index:  1,
			arg:    "foo.bar: ${!json_field:foo.bar}",
			result: "foo.bar: baz2",
		},
		{
			name: "json func 3",
			input: []string{
				`{"foo":{"bar":"baz1"}}`,
				`{"foo":{"bar":"baz2"}}`,
				`{"foo":{"bar":"baz3"}}`,
			},
			index:  2,
			arg:    "foo.bar: ${!json_field:foo.bar}",
			result: "foo.bar: baz3",
		},
		{
			name: "json func 4",
			input: []string{
				`{"foo":{"bar":"baz1"}}`,
				`{"foo":{"bar":"baz2"}}`,
				`{"foo":{"bar":"baz3"}}`,
			},
			index:  3,
			arg:    "foo.bar: ${!json_field:foo.bar}",
			result: "foo.bar: null",
		},
		{
			name: "json func 5",
			input: []string{
				`{"foo":{"bar":"baz1"}}`,
				`{"foo":{"bar":"baz2"}}`,
				`{"foo":{"bar":"baz3"}}`,
			},
			index:  -1,
			arg:    "foo.bar: ${!json_field:foo.bar}",
			result: "foo.bar: baz3",
		},
		{
			name: "json func 6",
			input: []string{
				`{"foo":{"bar":"baz1"}}`,
				`{"foo":{"bar":"baz2"}}`,
				`{"foo":{"bar":"baz3"}}`,
			},
			index:  1,
			arg:    "foo.bar: ${!json_field:foo.bar,0}",
			result: "foo.bar: baz1",
		},
	}

	for _, test := range tests {
		exp := test.result
		parts := [][]byte{}
		for _, input := range test.input {
			parts = append(parts, []byte(input))
		}
		act := string(ReplaceFunctionVariablesFor(
			message.New(parts),
			test.index,
			[]byte(test.arg),
		))
		if act != exp {
			t.Errorf("Wrong result for test '%v': %v != %v", test.name, act, exp)
		}
	}
}

func TestFunctionSwapping(t *testing.T) {
	hostname, _ := os.Hostname()

	exp := fmt.Sprintf("foo %v baz", hostname)
	act := string(ReplaceFunctionVariables(nil, []byte("foo ${!hostname} baz")))
	if act != exp {
		t.Errorf("Wrong result: %v != %v", act, exp)
	}

	exp = "foo ${!} baz"
	act = string(ReplaceFunctionVariables(nil, []byte("foo ${!} baz")))
	if act != exp {
		t.Errorf("Wrong result: %v != %v", act, exp)
	}

	exp = "foo ${!does_not_exist} baz"
	act = string(ReplaceFunctionVariables(nil, []byte("foo ${!does_not_exist} baz")))
	if act != exp {
		t.Errorf("Wrong result: %v != %v", act, exp)
	}

	now := time.Now()
	tStamp := string(ReplaceFunctionVariables(nil, []byte("${!timestamp_unix_nano}")))

	nanoseconds, err := strconv.ParseInt(tStamp, 10, 64)
	if err != nil {
		t.Fatal(err)
	}
	tThen := time.Unix(0, nanoseconds)

	if tThen.Sub(now).Seconds() > 5.0 {
		t.Errorf("Timestamps too far out of sync: %v and %v", tThen, now)
	}

	now = time.Now()
	tStamp = string(ReplaceFunctionVariables(nil, []byte("${!timestamp_unix}")))

	seconds, err := strconv.ParseInt(tStamp, 10, 64)
	if err != nil {
		t.Fatal(err)
	}
	tThen = time.Unix(seconds, 0)

	if tThen.Sub(now).Seconds() > 5.0 {
		t.Errorf("Timestamps too far out of sync: %v and %v", tThen, now)
	}

	now = time.Now()
	tStamp = string(ReplaceFunctionVariables(nil, []byte("${!timestamp_unix:10}")))

	var secondsF float64
	secondsF, err = strconv.ParseFloat(tStamp, 64)
	if err != nil {
		t.Fatal(err)
	}
	tThen = time.Unix(int64(secondsF), 0)

	if tThen.Sub(now).Seconds() > 5.0 {
		t.Errorf("Timestamps too far out of sync: %v and %v", tThen, now)
	}

	now = time.Now()
	tStamp = string(ReplaceFunctionVariables(nil, []byte("${!timestamp}")))

	tThen, err = time.Parse("Mon Jan 2 15:04:05 -0700 MST 2006", tStamp)
	if err != nil {
		t.Fatal(err)
	}

	if tThen.Sub(now).Seconds() > 5.0 {
		t.Errorf("Timestamps too far out of sync: %v and %v", tThen, now)
	}

	now = time.Now()
	tStamp = string(ReplaceFunctionVariables(nil, []byte("${!timestamp_utc}")))

	tThen, err = time.Parse("Mon Jan 2 15:04:05 -0700 MST 2006", tStamp)
	if err != nil {
		t.Fatal(err)
	}

	if tThen.Sub(now).Seconds() > 5.0 {
		t.Errorf("Timestamps too far out of sync: %v and %v", tThen, now)
	}
	if !strings.Contains(tStamp, "UTC") {
		t.Errorf("Non-UTC timezone detected: %v", tStamp)
	}
}

func TestFunctionEscape(t *testing.T) {
	tests := map[string]string{
		"foo ${{!echo:bar}} bar":      "foo ${!echo:bar} bar",
		"foo ${{!notafunction}} bar":  "foo ${!notafunction} bar",
		"foo ${{{!notafunction}} bar": "foo ${{{!notafunction}} bar",
		"foo ${!notafunction}} bar":   "foo ${!notafunction}} bar",
	}

	for input, exp := range tests {
		act := string(ReplaceFunctionVariables(nil, []byte(input)))
		if exp != act {
			t.Errorf("Wrong results for input (%v): %v != %v", input, act, exp)
		}
	}
}

func TestEchoFunction(t *testing.T) {
	tests := map[string]string{
		"foo ${!echo:bar}":              "foo bar",
		"foo ${!echo}":                  "foo ",
		"foo ${!echo:bar} ${!echo:baz}": "foo bar baz",
	}

	for input, exp := range tests {
		act := string(ReplaceFunctionVariables(nil, []byte(input)))
		if exp != act {
			t.Errorf("Wrong results for input (%v): %v != %v", input, act, exp)
		}
	}
}

func TestCountersFunction(t *testing.T) {
	tests := [][2]string{
		{"foo1: ${!count:foo}", "foo1: 1"},
		{"bar1: ${!count:bar}", "bar1: 1"},
		{"foo2: ${!count:foo} ${!count:foo}", "foo2: 2 3"},
		{"bar2: ${!count:bar} ${!count:bar}", "bar2: 2 3"},
		{"foo3: ${!count:foo} ${!count:foo}", "foo3: 4 5"},
		{"bar3: ${!count:bar} ${!count:bar}", "bar3: 4 5"},
	}

	for _, test := range tests {
		input := test[0]
		exp := test[1]
		act := string(ReplaceFunctionVariables(nil, []byte(input)))
		if exp != act {
			t.Errorf("Wrong results for input (%v): %v != %v", input, act, exp)
		}
	}
}

func TestCountersSharedFunction(t *testing.T) {
	tests := []string{
		"foo1: ${!count:foo}",
		"bar1: ${!count:bar}",
		"foo2: ${!count:foo} ${!count:foo}",
		"bar2: ${!count:bar} ${!count:bar}",
		"foo3: ${!count:foo} ${!count:foo}",
		"bar3: ${!count:bar} ${!count:bar}",
	}

	startChan := make(chan struct{})
	wg := sync.WaitGroup{}
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-startChan
			for _, input := range tests {
				act := string(ReplaceFunctionVariables(nil, []byte(input)))
				if act == input {
					t.Errorf("Unchanged input: %v", act)
				}
			}
		}()
	}
	close(startChan)
	wg.Wait()
}

func TestUUIDV4Function(t *testing.T) {
	results := map[string]struct{}{}

	for i := 0; i < 100; i++ {
		result := string(ReplaceFunctionVariables(nil, []byte(`${!uuid_v4}`)))
		if _, exists := results[result]; exists {
			t.Errorf("Duplicate UUID generated: %v", result)
		}
		results[result] = struct{}{}
	}
}
