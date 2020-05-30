package processor

import (
	"reflect"
	"testing"

	"github.com/windhooked/benthos/v3/lib/condition"
	"github.com/windhooked/benthos/v3/lib/log"
	"github.com/windhooked/benthos/v3/lib/message"
	"github.com/windhooked/benthos/v3/lib/metrics"
)

//------------------------------------------------------------------------------

func TestForEachEmpty(t *testing.T) {
	conf := NewConfig()
	conf.Type = "for_each"

	proc, err := New(conf, nil, log.Noop(), metrics.Noop())
	if err != nil {
		t.Fatal(err)
	}

	exp := [][]byte{
		[]byte("foo bar baz"),
	}
	msgs, res := proc.ProcessMessage(message.New(exp))
	if res != nil {
		t.Fatal(res.Error())
	}

	if len(msgs) != 1 {
		t.Errorf("Wrong count of result msgs: %v", len(msgs))
	}
	if act := message.GetAllBytes(msgs[0]); !reflect.DeepEqual(exp, act) {
		t.Errorf("Wrong results: %s != %s", act, exp)
	}
}

func TestForEachBasic(t *testing.T) {
	encodeConf := NewConfig()
	encodeConf.Type = "encode"
	encodeConf.Encode.Parts = []int{0}

	conf := NewConfig()
	conf.Type = "for_each"
	conf.ForEach = append(conf.ForEach, encodeConf)

	proc, err := New(conf, nil, log.Noop(), metrics.Noop())
	if err != nil {
		t.Fatal(err)
	}

	parts := [][]byte{
		[]byte("foo bar baz"),
		[]byte("1 2 3 4"),
		[]byte("hello foo world"),
	}
	exp := [][]byte{
		[]byte("Zm9vIGJhciBiYXo="),
		[]byte("MSAyIDMgNA=="),
		[]byte("aGVsbG8gZm9vIHdvcmxk"),
	}
	msgs, res := proc.ProcessMessage(message.New(parts))
	if res != nil {
		t.Fatal(res.Error())
	}

	if len(msgs) != 1 {
		t.Errorf("Wrong count of result msgs: %v", len(msgs))
	}
	if act := message.GetAllBytes(msgs[0]); !reflect.DeepEqual(exp, act) {
		t.Errorf("Wrong results: %s != %s", act, exp)
	}
}

func TestForEachFilterSome(t *testing.T) {
	cond := condition.NewConfig()
	cond.Type = "text"
	cond.Text.Arg = "foo"
	cond.Text.Operator = "contains"

	filterConf := NewConfig()
	filterConf.Type = "filter"
	filterConf.Filter.Config = cond

	conf := NewConfig()
	conf.Type = "for_each"
	conf.ForEach = append(conf.ForEach, filterConf)

	proc, err := New(conf, nil, log.Noop(), metrics.Noop())
	if err != nil {
		t.Fatal(err)
	}

	parts := [][]byte{
		[]byte("foo bar baz"),
		[]byte("1 2 3 4"),
		[]byte("hello foo world"),
	}
	exp := [][]byte{
		[]byte("foo bar baz"),
		[]byte("hello foo world"),
	}
	msgs, res := proc.ProcessMessage(message.New(parts))
	if res != nil {
		t.Fatal(res.Error())
	}

	if len(msgs) != 1 {
		t.Errorf("Wrong count of result msgs: %v", len(msgs))
	}
	if act := message.GetAllBytes(msgs[0]); !reflect.DeepEqual(exp, act) {
		t.Errorf("Wrong results: %s != %s", act, exp)
	}
}

func TestForEachMultiProcs(t *testing.T) {
	encodeConf := NewConfig()
	encodeConf.Type = "encode"
	encodeConf.Encode.Parts = []int{0}

	cond := condition.NewConfig()
	cond.Type = "text"
	cond.Text.Arg = "foo"
	cond.Text.Operator = "contains"

	filterConf := NewConfig()
	filterConf.Type = "filter"
	filterConf.Filter.Config = cond

	conf := NewConfig()
	conf.Type = "for_each"
	conf.ForEach = append(conf.ForEach, filterConf)
	conf.ForEach = append(conf.ForEach, encodeConf)

	proc, err := New(conf, nil, log.Noop(), metrics.Noop())
	if err != nil {
		t.Fatal(err)
	}

	parts := [][]byte{
		[]byte("foo bar baz"),
		[]byte("1 2 3 4"),
		[]byte("hello foo world"),
	}
	exp := [][]byte{
		[]byte("Zm9vIGJhciBiYXo="),
		[]byte("aGVsbG8gZm9vIHdvcmxk"),
	}
	msgs, res := proc.ProcessMessage(message.New(parts))
	if res != nil {
		t.Fatal(res.Error())
	}

	if len(msgs) != 1 {
		t.Errorf("Wrong count of result msgs: %v", len(msgs))
	}
	if act := message.GetAllBytes(msgs[0]); !reflect.DeepEqual(exp, act) {
		t.Errorf("Wrong results: %s != %s", act, exp)
	}
}

func TestForEachFilterAll(t *testing.T) {
	cond := condition.NewConfig()
	cond.Type = "text"
	cond.Text.Arg = "foo"
	cond.Text.Operator = "contains"

	filterConf := NewConfig()
	filterConf.Type = "filter"
	filterConf.Filter.Config = cond

	conf := NewConfig()
	conf.Type = "for_each"
	conf.ForEach = append(conf.ForEach, filterConf)

	proc, err := New(conf, nil, log.Noop(), metrics.Noop())
	if err != nil {
		t.Fatal(err)
	}

	parts := [][]byte{
		[]byte("bar baz"),
		[]byte("1 2 3 4"),
		[]byte("hello world"),
	}
	msgs, res := proc.ProcessMessage(message.New(parts))
	if res == nil {
		t.Fatal("expected empty response")
	}
	if err = res.Error(); err != nil {
		t.Error(err)
	}
	if len(msgs) != 0 {
		t.Errorf("Wrong count of result msgs: %v", len(msgs))
	}
}

//------------------------------------------------------------------------------

func TestProcessBatchEmpty(t *testing.T) {
	conf := NewConfig()
	conf.Type = "process_batch"

	proc, err := New(conf, nil, log.Noop(), metrics.Noop())
	if err != nil {
		t.Fatal(err)
	}

	exp := [][]byte{
		[]byte("foo bar baz"),
	}
	msgs, res := proc.ProcessMessage(message.New(exp))
	if res != nil {
		t.Fatal(res.Error())
	}

	if len(msgs) != 1 {
		t.Errorf("Wrong count of result msgs: %v", len(msgs))
	}
	if act := message.GetAllBytes(msgs[0]); !reflect.DeepEqual(exp, act) {
		t.Errorf("Wrong results: %s != %s", act, exp)
	}
}

func TestProcessBatchBasic(t *testing.T) {
	encodeConf := NewConfig()
	encodeConf.Type = "encode"
	encodeConf.Encode.Parts = []int{0}

	conf := NewConfig()
	conf.Type = "process_batch"
	conf.ProcessBatch = append(conf.ProcessBatch, encodeConf)

	proc, err := New(conf, nil, log.Noop(), metrics.Noop())
	if err != nil {
		t.Fatal(err)
	}

	parts := [][]byte{
		[]byte("foo bar baz"),
		[]byte("1 2 3 4"),
		[]byte("hello foo world"),
	}
	exp := [][]byte{
		[]byte("Zm9vIGJhciBiYXo="),
		[]byte("MSAyIDMgNA=="),
		[]byte("aGVsbG8gZm9vIHdvcmxk"),
	}
	msgs, res := proc.ProcessMessage(message.New(parts))
	if res != nil {
		t.Fatal(res.Error())
	}

	if len(msgs) != 1 {
		t.Errorf("Wrong count of result msgs: %v", len(msgs))
	}
	if act := message.GetAllBytes(msgs[0]); !reflect.DeepEqual(exp, act) {
		t.Errorf("Wrong results: %s != %s", act, exp)
	}
}

func TestProcessBatchFilterSome(t *testing.T) {
	cond := condition.NewConfig()
	cond.Type = "text"
	cond.Text.Arg = "foo"
	cond.Text.Operator = "contains"

	filterConf := NewConfig()
	filterConf.Type = "filter"
	filterConf.Filter.Config = cond

	conf := NewConfig()
	conf.Type = "process_batch"
	conf.ProcessBatch = append(conf.ProcessBatch, filterConf)

	proc, err := New(conf, nil, log.Noop(), metrics.Noop())
	if err != nil {
		t.Fatal(err)
	}

	parts := [][]byte{
		[]byte("foo bar baz"),
		[]byte("1 2 3 4"),
		[]byte("hello foo world"),
	}
	exp := [][]byte{
		[]byte("foo bar baz"),
		[]byte("hello foo world"),
	}
	msgs, res := proc.ProcessMessage(message.New(parts))
	if res != nil {
		t.Fatal(res.Error())
	}

	if len(msgs) != 1 {
		t.Errorf("Wrong count of result msgs: %v", len(msgs))
	}
	if act := message.GetAllBytes(msgs[0]); !reflect.DeepEqual(exp, act) {
		t.Errorf("Wrong results: %s != %s", act, exp)
	}
}

func TestProcessBatchMultiProcs(t *testing.T) {
	encodeConf := NewConfig()
	encodeConf.Type = "encode"
	encodeConf.Encode.Parts = []int{0}

	cond := condition.NewConfig()
	cond.Type = "text"
	cond.Text.Arg = "foo"
	cond.Text.Operator = "contains"

	filterConf := NewConfig()
	filterConf.Type = "filter"
	filterConf.Filter.Config = cond

	conf := NewConfig()
	conf.Type = "process_batch"
	conf.ProcessBatch = append(conf.ProcessBatch, filterConf)
	conf.ProcessBatch = append(conf.ProcessBatch, encodeConf)

	proc, err := New(conf, nil, log.Noop(), metrics.Noop())
	if err != nil {
		t.Fatal(err)
	}

	parts := [][]byte{
		[]byte("foo bar baz"),
		[]byte("1 2 3 4"),
		[]byte("hello foo world"),
	}
	exp := [][]byte{
		[]byte("Zm9vIGJhciBiYXo="),
		[]byte("aGVsbG8gZm9vIHdvcmxk"),
	}
	msgs, res := proc.ProcessMessage(message.New(parts))
	if res != nil {
		t.Fatal(res.Error())
	}

	if len(msgs) != 1 {
		t.Errorf("Wrong count of result msgs: %v", len(msgs))
	}
	if act := message.GetAllBytes(msgs[0]); !reflect.DeepEqual(exp, act) {
		t.Errorf("Wrong results: %s != %s", act, exp)
	}
}

func TestProcessBatchFilterAll(t *testing.T) {
	cond := condition.NewConfig()
	cond.Type = "text"
	cond.Text.Arg = "foo"
	cond.Text.Operator = "contains"

	filterConf := NewConfig()
	filterConf.Type = "filter"
	filterConf.Filter.Config = cond

	conf := NewConfig()
	conf.Type = "process_batch"
	conf.ProcessBatch = append(conf.ProcessBatch, filterConf)

	proc, err := New(conf, nil, log.Noop(), metrics.Noop())
	if err != nil {
		t.Fatal(err)
	}

	parts := [][]byte{
		[]byte("bar baz"),
		[]byte("1 2 3 4"),
		[]byte("hello world"),
	}
	msgs, res := proc.ProcessMessage(message.New(parts))
	if res == nil {
		t.Fatal("expected empty response")
	}
	if err = res.Error(); err != nil {
		t.Error(err)
	}
	if len(msgs) != 0 {
		t.Errorf("Wrong count of result msgs: %v", len(msgs))
	}
}

//------------------------------------------------------------------------------
