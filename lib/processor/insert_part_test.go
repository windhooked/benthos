package processor

import (
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/windhooked/benthos/v3/lib/log"
	"github.com/windhooked/benthos/v3/lib/message"
	"github.com/windhooked/benthos/v3/lib/metrics"
)

func TestInsertBoundaries(t *testing.T) {
	conf := NewConfig()
	conf.InsertPart.Content = "hello world"

	testLog := log.New(os.Stdout, log.Config{LogLevel: "NONE"})

	for i := 0; i < 10; i++ {
		for j := -5; j <= 5; j++ {
			conf.InsertPart.Index = j
			proc, err := NewInsertPart(conf, nil, testLog, metrics.DudType{})
			if err != nil {
				t.Error(err)
				return
			}

			var parts [][]byte
			for k := 0; k < i; k++ {
				parts = append(parts, []byte("foo"))
			}

			msgs, res := proc.ProcessMessage(message.New(parts))
			if len(msgs) != 1 {
				t.Error("Insert Part failed")
			} else if res != nil {
				t.Errorf("Expected nil response: %v", res)
			}
			if exp, act := i+1, len(message.GetAllBytes(msgs[0])); exp != act {
				t.Errorf("Wrong count of result parts: %v != %v", act, exp)
			}
		}
	}
}

func TestInsertPart(t *testing.T) {
	conf := NewConfig()
	conf.InsertPart.Content = "hello world"

	testLog := log.New(os.Stdout, log.Config{LogLevel: "NONE"})

	type test struct {
		index int
		in    [][]byte
		out   [][]byte
	}

	tests := []test{
		{
			index: 0,
			in: [][]byte{
				[]byte("0"),
				[]byte("1"),
			},
			out: [][]byte{
				[]byte("hello world"),
				[]byte("0"),
				[]byte("1"),
			},
		},
		{
			index: 0,
			in:    [][]byte{},
			out: [][]byte{
				[]byte("hello world"),
			},
		},
		{
			index: 1,
			in: [][]byte{
				[]byte("0"),
				[]byte("1"),
			},
			out: [][]byte{
				[]byte("0"),
				[]byte("hello world"),
				[]byte("1"),
			},
		},
		{
			index: 2,
			in: [][]byte{
				[]byte("0"),
				[]byte("1"),
			},
			out: [][]byte{
				[]byte("0"),
				[]byte("1"),
				[]byte("hello world"),
			},
		},
		{
			index: 3,
			in: [][]byte{
				[]byte("0"),
				[]byte("1"),
			},
			out: [][]byte{
				[]byte("0"),
				[]byte("1"),
				[]byte("hello world"),
			},
		},
		{
			index: -1,
			in: [][]byte{
				[]byte("0"),
				[]byte("1"),
			},
			out: [][]byte{
				[]byte("0"),
				[]byte("1"),
				[]byte("hello world"),
			},
		},
		{
			index: -2,
			in: [][]byte{
				[]byte("0"),
				[]byte("1"),
			},
			out: [][]byte{
				[]byte("0"),
				[]byte("hello world"),
				[]byte("1"),
			},
		},
		{
			index: -3,
			in: [][]byte{
				[]byte("0"),
				[]byte("1"),
			},
			out: [][]byte{
				[]byte("hello world"),
				[]byte("0"),
				[]byte("1"),
			},
		},
		{
			index: -4,
			in: [][]byte{
				[]byte("0"),
				[]byte("1"),
			},
			out: [][]byte{
				[]byte("hello world"),
				[]byte("0"),
				[]byte("1"),
			},
		},
	}

	for _, test := range tests {
		conf.InsertPart.Index = test.index
		proc, err := NewInsertPart(conf, nil, testLog, metrics.DudType{})
		if err != nil {
			t.Error(err)
			return
		}

		msgs, res := proc.ProcessMessage(message.New(test.in))
		if len(msgs) != 1 {
			t.Errorf("Insert Part failed on: %s", test.in)
		} else if res != nil {
			t.Errorf("Expected nil response: %v", res)
		}
		if exp, act := test.out, message.GetAllBytes(msgs[0]); !reflect.DeepEqual(exp, act) {
			t.Errorf("Unexpected output for %s at index %v: %s != %s", test.in, test.index, act, exp)
		}
	}
}

func TestInsertPartInterpolation(t *testing.T) {
	conf := NewConfig()
	conf.InsertPart.Content = "hello ${!hostname()} world"

	hostname, _ := os.Hostname()

	testLog := log.New(os.Stdout, log.Config{LogLevel: "NONE"})
	proc, err := NewInsertPart(conf, nil, testLog, metrics.DudType{})
	if err != nil {
		t.Error(err)
		return
	}

	type test struct {
		in  [][]byte
		out [][]byte
	}

	tests := []test{
		{
			in: [][]byte{
				[]byte("0"),
				[]byte("1"),
			},
			out: [][]byte{
				[]byte("0"),
				[]byte("1"),
				[]byte(fmt.Sprintf("hello %v world", hostname)),
			},
		},
		{
			in: [][]byte{},
			out: [][]byte{
				[]byte(fmt.Sprintf("hello %v world", hostname)),
			},
		},
	}

	for _, test := range tests {
		msgs, res := proc.ProcessMessage(message.New(test.in))
		if len(msgs) != 1 {
			t.Errorf("Insert Part failed on: %s", test.in)
		} else if res != nil {
			t.Errorf("Expected nil response: %v", res)
		}
		if exp, act := test.out, message.GetAllBytes(msgs[0]); !reflect.DeepEqual(exp, act) {
			t.Errorf("Unexpected output for %s: %s != %s", test.in, act, exp)
		}
	}
}
