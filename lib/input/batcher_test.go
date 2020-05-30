package input

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/windhooked/benthos/v3/lib/log"
	"github.com/windhooked/benthos/v3/lib/message"
	"github.com/windhooked/benthos/v3/lib/message/batch"
	"github.com/windhooked/benthos/v3/lib/metrics"
	"github.com/windhooked/benthos/v3/lib/response"
	"github.com/windhooked/benthos/v3/lib/types"
)

func TestBatcherStandard(t *testing.T) {
	mock := &mockInput{
		ts: make(chan types.Transaction),
	}

	batchConf := batch.NewPolicyConfig()
	batchConf.Count = 3

	batchPol, err := batch.NewPolicy(batchConf, types.NoopMgr(), log.Noop(), metrics.Noop())
	if err != nil {
		t.Fatal(err)
	}

	batcher := NewBatcher(batchPol, mock, log.Noop(), metrics.Noop())

	testMsgs := []string{}
	testResChans := []chan types.Response{}
	for i := 0; i < 8; i++ {
		testMsgs = append(testMsgs, fmt.Sprintf("test%v", i))
		testResChans = append(testResChans, make(chan types.Response))
	}

	resErrs := []error{}
	doneWritesChan := make(chan struct{})
	doneReadsChan := make(chan struct{})
	go func() {
		for i, m := range testMsgs {
			mock.ts <- types.NewTransaction(message.New([][]byte{[]byte(m)}), testResChans[i])
		}
		close(doneWritesChan)
		for _, rChan := range testResChans {
			resErrs = append(resErrs, (<-rChan).Error())
		}
		close(doneReadsChan)
	}()

	resChans := []chan<- types.Response{}

	var tran types.Transaction
	select {
	case tran = <-batcher.TransactionChan():
	case <-time.After(time.Second):
		t.Fatal("timed out")
	}
	resChans = append(resChans, tran.ResponseChan)

	if exp, act := 3, tran.Payload.Len(); exp != act {
		t.Errorf("Wrong batch size: %v != %v", act, exp)
	}
	tran.Payload.Iter(func(i int, part types.Part) error {
		if exp, act := fmt.Sprintf("test%v", i), string(part.Get()); exp != act {
			t.Errorf("Unexpected message part: %v != %v", act, exp)
		}
		return nil
	})

	select {
	case tran = <-batcher.TransactionChan():
	case <-time.After(time.Second):
		t.Fatal("timed out")
	}
	resChans = append(resChans, tran.ResponseChan)

	if exp, act := 3, tran.Payload.Len(); exp != act {
		t.Errorf("Wrong batch size: %v != %v", act, exp)
	}
	tran.Payload.Iter(func(i int, part types.Part) error {
		if exp, act := fmt.Sprintf("test%v", i+3), string(part.Get()); exp != act {
			t.Errorf("Unexpected message part: %v != %v", act, exp)
		}
		return nil
	})

	select {
	case <-batcher.TransactionChan():
		t.Error("Unexpected batch received")
	default:
	}

	select {
	case <-doneWritesChan:
	case <-time.After(time.Second):
		t.Error("timed out")
	}
	batcher.CloseAsync()

	select {
	case tran = <-batcher.TransactionChan():
	case <-time.After(time.Second):
		t.Fatal("timed out")
	}
	resChans = append(resChans, tran.ResponseChan)

	if exp, act := 2, tran.Payload.Len(); exp != act {
		t.Errorf("Wrong batch size: %v != %v", act, exp)
	}
	tran.Payload.Iter(func(i int, part types.Part) error {
		if exp, act := fmt.Sprintf("test%v", i+6), string(part.Get()); exp != act {
			t.Errorf("Unexpected message part: %v != %v", act, exp)
		}
		return nil
	})

	for i, rChan := range resChans {
		select {
		case rChan <- response.NewError(fmt.Errorf("testerr%v", i)):
		case <-time.After(time.Second):
			t.Fatal("timed out")
		}
	}

	select {
	case <-doneReadsChan:
	case <-time.After(time.Second):
		t.Error("timed out")
	}

	for i, err := range resErrs {
		exp := "testerr0"
		if i >= 3 {
			exp = "testerr1"
		}
		if i >= 6 {
			exp = "testerr2"
		}
		if act := err.Error(); exp != act {
			t.Errorf("Unexpected error returned: %v != %v", act, exp)
		}
	}

	if err := batcher.WaitForClose(time.Second); err != nil {
		t.Error(err)
	}
}

func TestBatcherTiming(t *testing.T) {
	mock := &mockInput{
		ts: make(chan types.Transaction),
	}

	batchConf := batch.NewPolicyConfig()
	batchConf.Count = 0
	batchConf.Period = "1ms"

	batchPol, err := batch.NewPolicy(batchConf, types.NoopMgr(), log.Noop(), metrics.Noop())
	if err != nil {
		t.Fatal(err)
	}

	batcher := NewBatcher(batchPol, mock, log.Noop(), metrics.Noop())

	resChan := make(chan types.Response)
	select {
	case mock.ts <- types.NewTransaction(message.New([][]byte{[]byte("foo1")}), resChan):
	case <-time.After(time.Second):
		t.Fatal("timed out")
	}

	var tran types.Transaction
	select {
	case tran = <-batcher.TransactionChan():
	case <-time.After(time.Second):
		t.Fatal("timed out")
	}

	if exp, act := 1, tran.Payload.Len(); exp != act {
		t.Errorf("Wrong batch size: %v != %v", act, exp)
	}
	if exp, act := "foo1", string(tran.Payload.Get(0).Get()); exp != act {
		t.Errorf("Unexpected message part: %v != %v", act, exp)
	}

	errSend := errors.New("this is a test error")
	select {
	case tran.ResponseChan <- response.NewError(errSend):
	case <-time.After(time.Second):
		t.Fatal("timed out")
	}
	select {
	case err := <-resChan:
		if err.Error() != errSend {
			t.Errorf("Unexpected error: %v != %v", err.Error(), errSend)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out")
	}

	select {
	case mock.ts <- types.NewTransaction(message.New([][]byte{[]byte("foo2")}), resChan):
	case <-time.After(time.Second):
		t.Fatal("timed out")
	}

	select {
	case tran = <-batcher.TransactionChan():
	case <-time.After(time.Second):
		t.Fatal("timed out")
	}

	if exp, act := 1, tran.Payload.Len(); exp != act {
		t.Errorf("Wrong batch size: %v != %v", act, exp)
	}
	if exp, act := "foo2", string(tran.Payload.Get(0).Get()); exp != act {
		t.Errorf("Unexpected message part: %v != %v", act, exp)
	}

	batcher.CloseAsync()

	select {
	case tran.ResponseChan <- response.NewError(errSend):
	case <-time.After(time.Second):
		t.Fatal("timed out")
	}
	select {
	case err := <-resChan:
		if err.Error() != errSend {
			t.Errorf("Unexpected error: %v != %v", err.Error(), errSend)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out")
	}

	if err := batcher.WaitForClose(time.Second); err != nil {
		t.Error(err)
	}
}
