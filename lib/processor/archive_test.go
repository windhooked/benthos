package processor

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"reflect"
	"testing"

	"github.com/windhooked/benthos/v3/lib/log"
	"github.com/windhooked/benthos/v3/lib/message"
	"github.com/windhooked/benthos/v3/lib/metrics"
	"github.com/windhooked/benthos/v3/lib/types"
)

func TestArchiveBadAlgo(t *testing.T) {
	conf := NewConfig()
	conf.Archive.Format = "does not exist"

	testLog := log.New(os.Stdout, log.Config{LogLevel: "NONE"})

	_, err := NewArchive(conf, nil, testLog, metrics.DudType{})
	if err == nil {
		t.Error("Expected error from bad algo")
	}
}

func TestArchiveTar(t *testing.T) {
	conf := NewConfig()
	conf.Archive.Format = "tar"
	conf.Archive.Path = "foo-${!meta(\"path\")}"

	testLog := log.New(os.Stdout, log.Config{LogLevel: "NONE"})

	exp := [][]byte{
		[]byte("hello world first part"),
		[]byte("hello world second part"),
		[]byte("third part"),
		[]byte("fourth"),
		[]byte("5"),
	}

	proc, err := NewArchive(conf, nil, testLog, metrics.DudType{})
	if err != nil {
		t.Fatal(err)
	}

	msg := message.New(exp)
	msg.Iter(func(i int, p types.Part) error {
		p.Metadata().Set("path", fmt.Sprintf("bar%v", i))
		return nil
	})
	msgs, res := proc.ProcessMessage(msg)
	if len(msgs) != 1 {
		t.Error("Archive failed")
	} else if res != nil {
		t.Errorf("Expected nil response: %v", res)
	}
	if msgs[0].Len() != 1 {
		t.Fatal("More parts than expected")
	}

	act := [][]byte{}

	buf := bytes.NewBuffer(msgs[0].Get(0).Get())
	tr := tar.NewReader(buf)
	i := 0
	for {
		var hdr *tar.Header
		hdr, err = tr.Next()
		if err == io.EOF {
			// end of tar archive
			break
		}
		if err != nil {
			t.Fatal(err)
		}

		newPartBuf := bytes.Buffer{}
		if _, err = newPartBuf.ReadFrom(tr); err != nil {
			t.Fatal(err)
		}

		act = append(act, newPartBuf.Bytes())
		if exp, act := fmt.Sprintf("foo-bar%v", i), hdr.FileInfo().Name(); exp != act {
			t.Errorf("Wrong filename: %v != %v", act, exp)
		}
		i++
	}

	if !reflect.DeepEqual(exp, act) {
		t.Errorf("Unexpected output: %s != %s", act, exp)
	}
}

func TestArchiveZip(t *testing.T) {
	conf := NewConfig()
	conf.Archive.Format = "zip"

	testLog := log.New(os.Stdout, log.Config{LogLevel: "NONE"})

	exp := [][]byte{
		[]byte("hello world first part"),
		[]byte("hello world second part"),
		[]byte("third part"),
		[]byte("fourth"),
		[]byte("5"),
	}

	proc, err := NewArchive(conf, nil, testLog, metrics.DudType{})
	if err != nil {
		t.Fatal(err)
	}

	msgs, res := proc.ProcessMessage(message.New(exp))
	if len(msgs) != 1 {
		t.Error("Archive failed")
	} else if res != nil {
		t.Errorf("Expected nil response: %v", res)
	}
	if msgs[0].Len() != 1 {
		t.Fatal("More parts than expected")
	}

	act := [][]byte{}

	buf := bytes.NewReader(msgs[0].Get(0).Get())
	zr, err := zip.NewReader(buf, int64(buf.Len()))
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range zr.File {
		fr, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}

		newPartBuf := bytes.Buffer{}
		if _, err = newPartBuf.ReadFrom(fr); err != nil {
			t.Fatal(err)
		}

		act = append(act, newPartBuf.Bytes())
	}

	if !reflect.DeepEqual(exp, act) {
		t.Errorf("Unexpected output: %s != %s", act, exp)
	}
}

func TestArchiveLines(t *testing.T) {
	conf := NewConfig()
	conf.Archive.Format = "lines"

	testLog := log.New(os.Stdout, log.Config{LogLevel: "NONE"})

	proc, err := NewArchive(conf, nil, testLog, metrics.DudType{})
	if err != nil {
		t.Fatal(err)
	}

	msgs, res := proc.ProcessMessage(message.New([][]byte{
		[]byte("hello world first part"),
		[]byte("hello world second part"),
		[]byte("third part"),
		[]byte("fourth"),
		[]byte("5"),
	}))
	if len(msgs) != 1 {
		t.Error("Archive failed")
	} else if res != nil {
		t.Errorf("Expected nil response: %v", res)
	}
	if msgs[0].Len() != 1 {
		t.Fatal("More parts than expected")
	}

	exp := [][]byte{
		[]byte(`hello world first part
hello world second part
third part
fourth
5`),
	}
	if act := message.GetAllBytes(msgs[0]); !reflect.DeepEqual(exp, act) {
		t.Errorf("Unexpected output: %s != %s", act, exp)
	}
}

func TestArchiveJSONArray(t *testing.T) {
	conf := NewConfig()
	conf.Archive.Format = "json_array"

	proc, err := NewArchive(conf, nil, log.Noop(), metrics.Noop())
	if err != nil {
		t.Fatal(err)
	}

	msgs, res := proc.ProcessMessage(message.New([][]byte{
		[]byte(`{"foo":"bar"}`),
		[]byte(`5`),
		[]byte(`"testing 123"`),
		[]byte(`["nested","array"]`),
		[]byte(`true`),
	}))
	if len(msgs) != 1 {
		t.Error("Archive failed")
	} else if res != nil {
		t.Errorf("Expected nil response: %v", res)
	}
	if msgs[0].Len() != 1 {
		t.Fatal("More parts than expected")
	}

	exp := [][]byte{[]byte(
		`[{"foo":"bar"},5,"testing 123",["nested","array"],true]`,
	)}
	if act := message.GetAllBytes(msgs[0]); !reflect.DeepEqual(exp, act) {
		t.Errorf("Unexpected output: %s != %s", act, exp)
	}
}

func TestArchiveBinary(t *testing.T) {
	conf := NewConfig()
	conf.Archive.Format = "binary"

	testLog := log.New(os.Stdout, log.Config{LogLevel: "NONE"})
	proc, err := NewArchive(conf, nil, testLog, metrics.DudType{})
	if err != nil {
		t.Error(err)
		return
	}

	testMsg := message.New([][]byte{[]byte("hello"), []byte("world")})
	testMsgBlob := message.ToBytes(testMsg)

	if msgs, _ := proc.ProcessMessage(testMsg); len(msgs) == 1 {
		if lParts := msgs[0].Len(); lParts != 1 {
			t.Errorf("Wrong number of parts returned: %v != %v", lParts, 1)
		}
		if !reflect.DeepEqual(testMsgBlob, msgs[0].Get(0).Get()) {
			t.Errorf("Returned message did not match: %s != %s", msgs[0].Get(0).Get(), testMsgBlob)
		}
	} else {
		t.Error("Failed on good message")
	}
}

func TestArchiveEmpty(t *testing.T) {
	conf := NewConfig()

	testLog := log.New(os.Stdout, log.Config{LogLevel: "NONE"})
	proc, err := NewArchive(conf, nil, testLog, metrics.DudType{})
	if err != nil {
		t.Error(err)
		return
	}

	msgs, _ := proc.ProcessMessage(message.New([][]byte{}))
	if len(msgs) != 0 {
		t.Error("Expected failure with zero part message")
	}
}
