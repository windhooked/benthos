package message

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/windhooked/benthos/v3/lib/message/metadata"
)

//------------------------------------------------------------------------------

func TestGetAllBytes(t *testing.T) {
	rawBytes := [][]byte{
		[]byte("foo"),
		[]byte("bar"),
		[]byte("baz"),
	}
	m := New(rawBytes)
	if exp, act := rawBytes, GetAllBytes(m); !reflect.DeepEqual(exp, act) {
		t.Errorf("Wrong result: %s != %s", act, exp)
	}
}

func TestSetAllMetadata(t *testing.T) {
	meta := metadata.New(map[string]string{
		"foo": "bar",
	})
	m := New([][]byte{
		[]byte("foo"),
		[]byte("bar"),
		[]byte("baz"),
	})
	SetAllMetadata(m, meta)
	if exp, act := "bar", m.Get(0).Metadata().Get("foo"); exp != act {
		t.Errorf("Wrong result: %v != %v", act, exp)
	}
	if exp, act := "bar", m.Get(1).Metadata().Get("foo"); exp != act {
		t.Errorf("Wrong result: %v != %v", act, exp)
	}
	if exp, act := "bar", m.Get(2).Metadata().Get("foo"); exp != act {
		t.Errorf("Wrong result: %v != %v", act, exp)
	}
}

func TestCloneGeneric(t *testing.T) {
	var original interface{}
	var cloned interface{}

	err := json.Unmarshal([]byte(`{
		"root":{
			"first":{
				"value1": 1,
				"value2": 1.2,
				"value3": false,
				"value4": "hello world"
			},
			"second": [
				1,
				1.2,
				false,
				"hello world"
			]
		}
	}`), &original)
	if err != nil {
		t.Fatal(err)
	}

	if cloned, err = cloneGeneric(original); err != nil {
		t.Fatal(err)
	}

	if exp, act := original, cloned; !reflect.DeepEqual(exp, act) {
		t.Fatalf("Wrong cloned contents: %v != %v", act, exp)
	}

	target := cloned.(map[string]interface{})
	target = target["root"].(map[string]interface{})
	target = target["first"].(map[string]interface{})
	target["value1"] = 2

	target = original.(map[string]interface{})
	target = target["root"].(map[string]interface{})
	target = target["first"].(map[string]interface{})
	if exp, act := float64(1), target["value1"].(float64); exp != act {
		t.Errorf("Original value was mutated: %v != %v", act, exp)
	}
}

func TestCloneGenericYAML(t *testing.T) {
	var original interface{} = map[interface{}]interface{}{
		"root": map[interface{}]interface{}{
			"first": map[interface{}]interface{}{
				"value1": 1,
				"value2": 1.2,
				"value3": false,
				"value4": "hello world",
			},
			"second": []interface{}{
				1, 1.2, false, "hello world",
			},
		},
	}
	var cloned interface{}
	var err error

	if cloned, err = cloneGeneric(original); err != nil {
		t.Fatal(err)
	}

	if exp, act := original, cloned; !reflect.DeepEqual(exp, act) {
		t.Fatalf("Wrong cloned contents: %v != %v", act, exp)
	}

	target := cloned.(map[interface{}]interface{})
	target = target["root"].(map[interface{}]interface{})
	target = target["first"].(map[interface{}]interface{})
	target["value1"] = 2

	target = original.(map[interface{}]interface{})
	target = target["root"].(map[interface{}]interface{})
	target = target["first"].(map[interface{}]interface{})
	if exp, act := 1, target["value1"].(int); exp != act {
		t.Errorf("Original value was mutated: %v != %v", act, exp)
	}
}

//------------------------------------------------------------------------------

var benchResult float64

func BenchmarkCloneGeneric(b *testing.B) {
	var generic, cloned interface{}
	err := json.Unmarshal([]byte(`{
		"root":{
			"first":{
				"value1": 1,
				"value2": 1.2,
				"value3": false,
				"value4": "hello world"
			},
			"second": [
				1,
				1.2,
				false,
				"hello world"
			]
		}
	}`), &generic)
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if cloned, err = cloneGeneric(generic); err != nil {
			b.Fatal(err)
		}
	}
	b.StopTimer()

	target := cloned.(map[string]interface{})
	target = target["root"].(map[string]interface{})
	target = target["first"].(map[string]interface{})
	benchResult = target["value1"].(float64)
	if exp, act := float64(1), benchResult; exp != act {
		b.Errorf("Wrong result: %v != %v", act, exp)
	}
}

func BenchmarkCloneJSON(b *testing.B) {
	var generic, cloned interface{}
	err := json.Unmarshal([]byte(`{
		"root":{
			"first":{
				"value1": 1,
				"value2": 1.2,
				"value3": false,
				"value4": "hello world"
			},
			"second": [
				1,
				1.2,
				false,
				"hello world"
			]
		}
	}`), &generic)
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var interBytes []byte
		if interBytes, err = json.Marshal(generic); err != nil {
			b.Fatal(err)
		}
		if err = json.Unmarshal(interBytes, &cloned); err != nil {
			b.Fatal(err)
		}
	}
	b.StopTimer()

	target := cloned.(map[string]interface{})
	target = target["root"].(map[string]interface{})
	target = target["first"].(map[string]interface{})
	benchResult = target["value1"].(float64)
	if exp, act := float64(1), benchResult; exp != act {
		b.Errorf("Wrong result: %v != %v", act, exp)
	}
}

//------------------------------------------------------------------------------
