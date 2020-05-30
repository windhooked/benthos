package processor

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/windhooked/benthos/v3/lib/log"
	"github.com/windhooked/benthos/v3/lib/message"
	"github.com/windhooked/benthos/v3/lib/metrics"
)

func TestUnarchiveBadAlgo(t *testing.T) {
	conf := NewConfig()
	conf.Unarchive.Format = "does not exist"

	testLog := log.New(os.Stdout, log.Config{LogLevel: "NONE"})

	_, err := NewUnarchive(conf, nil, testLog, metrics.DudType{})
	if err == nil {
		t.Error("Expected error from bad algo")
	}
}

func TestUnarchiveTar(t *testing.T) {
	conf := NewConfig()
	conf.Unarchive.Format = "tar"

	testLog := log.New(os.Stdout, log.Config{LogLevel: "NONE"})

	input := [][]byte{
		[]byte("hello world first part"),
		[]byte("hello world second part"),
		[]byte("third part"),
		[]byte("fourth"),
		[]byte("5"),
	}

	exp := [][]byte{}
	expNames := []string{}

	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	for i := range input {
		exp = append(exp, input[i])

		hdr := &tar.Header{
			Name: fmt.Sprintf("testfile%v", i),
			Mode: 0600,
			Size: int64(len(input[i])),
		}
		expNames = append(expNames, hdr.Name)
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write(input[i]); err != nil {
			t.Fatal(err)
		}
	}

	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}

	input = [][]byte{buf.Bytes()}

	if reflect.DeepEqual(input, exp) {
		t.Fatal("Input and exp output are the same")
	}

	proc, err := NewUnarchive(conf, nil, testLog, metrics.DudType{})
	if err != nil {
		t.Fatal(err)
	}

	msgs, res := proc.ProcessMessage(message.New(input))
	if len(msgs) != 1 {
		t.Errorf("Unarchive failed: %v", res)
	} else if res != nil {
		t.Errorf("Expected nil response: %v", res)
	}
	if act := message.GetAllBytes(msgs[0]); !reflect.DeepEqual(exp, act) {
		t.Errorf("Unexpected output: %s != %s", act, exp)
	}
	for i := 0; i < msgs[0].Len(); i++ {
		if name := msgs[0].Get(i).Metadata().Get("archive_filename"); name != expNames[i] {
			t.Errorf("Unexpected name %d: %s != %s", i, name, expNames[i])
		}
	}
}

func TestUnarchiveZip(t *testing.T) {
	conf := NewConfig()
	conf.Unarchive.Format = "zip"

	testLog := log.New(os.Stdout, log.Config{LogLevel: "NONE"})

	input := [][]byte{
		[]byte("hello world first part"),
		[]byte("hello world second part"),
		[]byte("third part"),
		[]byte("fourth"),
		[]byte("5"),
	}

	exp := [][]byte{}
	expNames := []string{}

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	for i := range input {
		exp = append(exp, input[i])

		name := fmt.Sprintf("testfile%v", i)
		expNames = append(expNames, name)
		if fw, err := zw.Create(name); err != nil {
			t.Fatal(err)
		} else {
			if _, err := fw.Write(input[i]); err != nil {
				t.Fatal(err)
			}
		}
	}

	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}

	input = [][]byte{buf.Bytes()}

	if reflect.DeepEqual(input, exp) {
		t.Fatal("Input and exp output are the same")
	}

	proc, err := NewUnarchive(conf, nil, testLog, metrics.DudType{})
	if err != nil {
		t.Fatal(err)
	}

	msgs, res := proc.ProcessMessage(message.New(input))
	if len(msgs) != 1 {
		t.Errorf("Unarchive failed: %v", res)
	} else if res != nil {
		t.Errorf("Expected nil response: %v", res)
	}
	if act := message.GetAllBytes(msgs[0]); !reflect.DeepEqual(exp, act) {
		t.Errorf("Unexpected output: %s != %s", act, exp)
	}
	for i := 0; i < msgs[0].Len(); i++ {
		if name := msgs[0].Get(i).Metadata().Get("archive_filename"); name != expNames[i] {
			t.Errorf("Unexpected name %d: %s != %s", i, name, expNames[i])
		}
	}
}

func TestUnarchiveLines(t *testing.T) {
	conf := NewConfig()
	conf.Unarchive.Format = "lines"

	testLog := log.New(os.Stdout, log.Config{LogLevel: "NONE"})

	exp := [][]byte{
		[]byte("hello world first part"),
		[]byte("hello world second part"),
		[]byte("third part"),
		[]byte("fourth"),
		[]byte("5"),
	}

	proc, err := NewUnarchive(conf, nil, testLog, metrics.DudType{})
	if err != nil {
		t.Fatal(err)
	}

	msgs, res := proc.ProcessMessage(message.New([][]byte{
		[]byte(`hello world first part
hello world second part
third part
fourth
5`),
	}))
	if len(msgs) != 1 {
		t.Error("Unarchive failed")
	} else if res != nil {
		t.Errorf("Expected nil response: %v", res)
	}
	if act := message.GetAllBytes(msgs[0]); !reflect.DeepEqual(exp, act) {
		t.Errorf("Unexpected output: %s != %s", act, exp)
	}
}

func TestUnarchiveJSONDocuments(t *testing.T) {
	conf := NewConfig()
	conf.Unarchive.Format = "json_documents"

	exp := [][]byte{
		[]byte(`{"foo":"bar"}`),
		[]byte(`5`),
		[]byte(`"testing 123"`),
		[]byte(`["root","is","an","array"]`),
		[]byte(`{"bar":"baz"}`),
		[]byte(`true`),
	}

	proc, err := NewUnarchive(conf, nil, log.Noop(), metrics.Noop())
	if err != nil {
		t.Fatal(err)
	}

	msgs, res := proc.ProcessMessage(message.New([][]byte{
		[]byte(`{"foo":"bar"} 5 "testing 123" ["root", "is", "an", "array"] {"bar": "baz"} true`),
	}))
	if len(msgs) != 1 {
		t.Error("Unarchive failed")
	} else if res != nil {
		t.Errorf("Expected nil response: %v", res)
	}
	if act := message.GetAllBytes(msgs[0]); !reflect.DeepEqual(exp, act) {
		t.Errorf("Unexpected output: %s != %s", act, exp)
	}
}

func TestUnarchiveJSONArray(t *testing.T) {
	conf := NewConfig()
	conf.Unarchive.Format = "json_array"

	exp := [][]byte{
		[]byte(`{"foo":"bar"}`),
		[]byte(`5`),
		[]byte(`"testing 123"`),
		[]byte(`["nested","array"]`),
		[]byte(`true`),
	}

	proc, err := NewUnarchive(conf, nil, log.Noop(), metrics.Noop())
	if err != nil {
		t.Fatal(err)
	}

	msgs, res := proc.ProcessMessage(message.New([][]byte{
		[]byte(`[{"foo":"bar"},5,"testing 123",["nested","array"],true]`),
	}))
	if len(msgs) != 1 {
		t.Error("Unarchive failed")
	} else if res != nil {
		t.Errorf("Expected nil response: %v", res)
	}
	if act := message.GetAllBytes(msgs[0]); !reflect.DeepEqual(exp, act) {
		t.Errorf("Unexpected output: %s != %s", act, exp)
	}
}

func TestUnarchiveJSONMap(t *testing.T) {
	conf := NewConfig()
	conf.Unarchive.Format = "json_map"

	exp := [][]byte{
		[]byte(`{"foo":"bar"}`),
		[]byte(`5`),
		[]byte(`"testing 123"`),
		[]byte(`["nested","array"]`),
		[]byte(`true`),
	}
	expKeys := []string{
		"a", "b", "c", "d", "e",
	}

	proc, err := NewUnarchive(conf, nil, log.Noop(), metrics.Noop())
	if err != nil {
		t.Fatal(err)
	}

	msgs, res := proc.ProcessMessage(message.New([][]byte{
		[]byte(`{"a":{"foo":"bar"},"b":5,"c":"testing 123","d":["nested","array"],"e":true}`),
	}))
	if len(msgs) != 1 {
		t.Error("Unarchive failed")
	} else if res != nil {
		t.Errorf("Expected nil response: %v", res)
	} else if msgs[0].Len() != len(exp) {
		t.Errorf("Incorrect number of messages: %d != %d", msgs[0].Len(), len(exp))
	}

	// we need to be careful of the random order the map will be generated in
	// so we can't just test for byte equivalence of the whole array
	found := make([]bool, msgs[0].Len())
	for i := 0; i < msgs[0].Len(); i++ {
		key := msgs[0].Get(i).Metadata().Get("archive_key")
		seq := -1
		for j := 0; j < len(expKeys); j++ {
			if expKeys[j] == key {
				seq = j
			}
		}
		if seq < 0 {
			t.Errorf("Unexpected output: incorrect key %s found in position %d", key, i)
		}
		if found[seq] {
			t.Errorf("Unexpected output: duplicate key %s found in position %d", key, i)
		}
		found[seq] = true
		if act := msgs[0].Get(i).Get(); !reflect.DeepEqual(exp[seq], act) {
			t.Errorf("Unexpected output: %s != %s", act, exp[seq])
		}
	}
	for i := 0; i < msgs[0].Len(); i++ {
		if !found[i] {
			t.Errorf("Missing output: message for key %s not found", expKeys[i])
		}
	}
}

func TestUnarchiveBinary(t *testing.T) {
	conf := NewConfig()
	conf.Unarchive.Format = "binary"

	testLog := log.New(os.Stdout, log.Config{LogLevel: "NONE"})
	proc, err := NewUnarchive(conf, nil, testLog, metrics.DudType{})
	if err != nil {
		t.Error(err)
		return
	}

	msgs, _ := proc.ProcessMessage(
		message.New([][]byte{[]byte("wat this isnt good")}),
	)
	if exp, act := 1, len(msgs); exp != act {
		t.Fatalf("Wrong count: %v != %v", act, exp)
	}
	if exp, act := 1, msgs[0].Len(); exp != act {
		t.Fatalf("Wrong count: %v != %v", act, exp)
	}
	if !HasFailed(msgs[0].Get(0)) {
		t.Error("Expected fail")
	}

	testMsg := message.New([][]byte{[]byte("hello"), []byte("world")})
	testMsgBlob := message.ToBytes(testMsg)

	if msgs, _ := proc.ProcessMessage(message.New([][]byte{testMsgBlob})); len(msgs) == 1 {
		if !reflect.DeepEqual(message.GetAllBytes(testMsg), message.GetAllBytes(msgs[0])) {
			t.Errorf("Returned message did not match: %v != %v", msgs, testMsg)
		}
	} else {
		t.Error("Failed on good message")
	}
}

func TestUnarchiveIndexBounds(t *testing.T) {
	conf := NewConfig()
	conf.Unarchive.Format = "tar"

	testLog := log.New(os.Stdout, log.Config{LogLevel: "NONE"})

	input := [][]byte{
		[]byte("0"),
		[]byte("1"),
		[]byte("2"),
		[]byte("3"),
		[]byte("4"),
	}

	for i := range input {
		var buf bytes.Buffer
		tw := tar.NewWriter(&buf)

		hdr := &tar.Header{
			Name: fmt.Sprintf("testfile%v", i),
			Mode: 0600,
			Size: int64(len(input[i])),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write(input[i]); err != nil {
			t.Fatal(err)
		}

		if err := tw.Close(); err != nil {
			t.Fatal(err)
		}

		input[i] = buf.Bytes()
	}

	type result struct {
		index int
		value string
	}

	tests := map[int]result{
		-5: {
			index: 0,
			value: "0",
		},
		-4: {
			index: 1,
			value: "1",
		},
		-3: {
			index: 2,
			value: "2",
		},
		-2: {
			index: 3,
			value: "3",
		},
		-1: {
			index: 4,
			value: "4",
		},
		0: {
			index: 0,
			value: "0",
		},
		1: {
			index: 1,
			value: "1",
		},
		2: {
			index: 2,
			value: "2",
		},
		3: {
			index: 3,
			value: "3",
		},
		4: {
			index: 4,
			value: "4",
		},
	}

	for i, result := range tests {
		conf.Unarchive.Parts = []int{i}
		proc, err := NewUnarchive(conf, nil, testLog, metrics.DudType{})
		if err != nil {
			t.Fatal(err)
		}

		msgs, res := proc.ProcessMessage(message.New(input))
		if len(msgs) != 1 {
			t.Errorf("Unarchive failed on index: %v", i)
		} else if res != nil {
			t.Errorf("Expected nil response: %v", res)
		}
		if exp, act := result.value, string(message.GetAllBytes(msgs[0])[result.index]); exp != act {
			t.Errorf("Unexpected output for index %v: %v != %v", i, act, exp)
		}
		if exp, act := result.value, string(message.GetAllBytes(msgs[0])[(result.index+1)%5]); exp == act {
			t.Errorf("Processor was applied to wrong index %v: %v != %v", (result.index+1)%5, act, exp)
		}
	}
}
