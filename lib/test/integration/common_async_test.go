package integration

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/windhooked/benthos/v3/lib/input/reader"
	"github.com/windhooked/benthos/v3/lib/message"
	"github.com/windhooked/benthos/v3/lib/output/writer"
	"github.com/windhooked/benthos/v3/lib/response"
	"github.com/windhooked/benthos/v3/lib/types"
)

func checkALOSynchronousAsync(
	outputCtr func() (writer.Type, error),
	inputCtr func() (reader.Async, error),
	t *testing.T,
) {
	ctx, done := context.WithTimeout(context.Background(), time.Second*60)
	defer done()

	output, err := outputCtr()
	if err != nil {
		t.Fatal(err)
	}
	input, err := inputCtr()
	if err != nil {
		t.Fatal(err)
	}

	N := 100

	testMsgs := map[string]struct{}{}
	for i := 0; i < N; i++ {
		str := fmt.Sprintf("hello world: %v", i)
		testMsgs[str] = struct{}{}
		msg := message.New([][]byte{
			[]byte(str),
		})
		if err = output.Write(msg); err != nil {
			t.Fatal(err)
		}
	}

	receivedMsgs := map[string]struct{}{}
	for i := 0; i < len(testMsgs); i++ {
		var actM types.Message
		var ackFn reader.AsyncAckFn
		if actM, ackFn, err = input.ReadWithContext(ctx); err != nil {
			if err == types.ErrNotConnected {
				if err = input.ConnectWithContext(ctx); err != nil {
					t.Fatal(err)
				}
				actM, ackFn, err = input.ReadWithContext(ctx)
			}
			if err != nil {
				t.Fatal(err)
			}
		}
		ackErr := false
		var res types.Response = response.NewAck()
		if i%10 == 0 {
			res = response.NewError(errors.New("nah"))
			ackErr = true
		}
		if !ackErr {
			actM.Iter(func(i int, part types.Part) error {
				act := string(part.Get())

				if _, exists := receivedMsgs[act]; exists {
					t.Errorf("Duplicate message: %v", act)
				} else {
					receivedMsgs[act] = struct{}{}
				}
				if _, exists := testMsgs[act]; !exists {
					t.Errorf("Unexpected message: %v", act)
				}
				delete(testMsgs, act)
				return nil
			})
		}
		if err = ackFn(ctx, res); err != nil {
			t.Error(err)
		}
	}

	lMsgs := len(testMsgs)
	if lMsgs == 0 {
		t.Error("Expected remaining messages")
	}

	for lMsgs > 0 {
		var actM types.Message
		var ackFn reader.AsyncAckFn
		if actM, ackFn, err = input.ReadWithContext(ctx); err != nil {
			if err == types.ErrNotConnected {
				if err = input.ConnectWithContext(ctx); err != nil {
					t.Fatal(err)
				}
				actM, ackFn, err = input.ReadWithContext(ctx)
			}
			if err != nil {
				t.Fatal(err)
			}
		}
		actM.Iter(func(i int, part types.Part) error {
			act := string(part.Get())
			if _, exists := receivedMsgs[act]; exists {
				t.Errorf("Duplicate message: %v", act)
			} else {
				receivedMsgs[act] = struct{}{}
			}
			if _, exists := testMsgs[act]; !exists {
				t.Errorf("Unexpected message: %v", act)
			}
			delete(testMsgs, act)
			return nil
		})
		if err = ackFn(ctx, response.NewAck()); err != nil {
			t.Error(err)
		}
		lMsgs = len(testMsgs)
	}

	input.CloseAsync()
	output.CloseAsync()
	if err = input.WaitForClose(time.Second); err != nil {
		t.Error(err)
	}
	if err = output.WaitForClose(time.Second); err != nil {
		t.Error(err)
	}
}

func checkALOAsyncParallelWrites(
	outputCtr func() (writer.Type, error),
	inputCtr func() (reader.Async, error),
	N int,
	t *testing.T,
) {
	ctx, done := context.WithTimeout(context.Background(), time.Second*60)
	defer done()

	output, err := outputCtr()
	if err != nil {
		t.Fatal(err)
	}
	input, err := inputCtr()
	if err != nil {
		t.Fatal(err)
	}

	testMsgs := map[string]struct{}{}
	testMsgsByIndex := [][]byte{}
	for i := 0; i < N; i++ {
		str := fmt.Sprintf("hello world: %v", i)
		testMsgs[str] = struct{}{}
		testMsgsByIndex = append(testMsgsByIndex, []byte(str))
	}

	startChan := make(chan struct{})
	wg := sync.WaitGroup{}
	wg.Add(N)
	for i := 0; i < N; i++ {
		go func(index int) {
			defer wg.Done()
			<-startChan
			msg := message.New([][]byte{
				testMsgsByIndex[index],
			})
			if werr := output.Write(msg); werr != nil {
				t.Fatal(werr)
			}
		}(i)
	}
	close(startChan)
	wg.Wait()

	receivedMsgs := map[string]struct{}{}
	for i := 0; i < len(testMsgs); i++ {
		var actM types.Message
		var ackFn reader.AsyncAckFn
		if actM, ackFn, err = input.ReadWithContext(ctx); err != nil {
			if err == types.ErrNotConnected {
				if err = input.ConnectWithContext(ctx); err != nil {
					t.Fatal(err)
				}
				actM, ackFn, err = input.ReadWithContext(ctx)
			}
			if err != nil {
				t.Fatal(err)
			}
		}
		ackErr := false
		var res types.Response = response.NewAck()
		if i%10 == 0 {
			res = response.NewError(errors.New("nah"))
			ackErr = true
		}
		if !ackErr {
			actM.Iter(func(i int, part types.Part) error {
				act := string(part.Get())

				if _, exists := receivedMsgs[act]; exists {
					t.Errorf("Duplicate message: %v", act)
				} else {
					receivedMsgs[act] = struct{}{}
				}
				if _, exists := testMsgs[act]; !exists {
					t.Errorf("Unexpected message: %v", act)
				}
				delete(testMsgs, act)
				return nil
			})
		}
		if err = ackFn(ctx, res); err != nil {
			t.Error(err)
		}
	}

	lMsgs := len(testMsgs)
	if lMsgs == 0 {
		t.Error("Expected remaining messages")
	}

	for lMsgs > 0 {
		var actM types.Message
		var ackFn reader.AsyncAckFn
		if actM, ackFn, err = input.ReadWithContext(ctx); err != nil {
			if err == types.ErrNotConnected {
				if err = input.ConnectWithContext(ctx); err != nil {
					t.Fatal(err)
				}
				actM, ackFn, err = input.ReadWithContext(ctx)
			}
			if err != nil {
				t.Fatal(err)
			}
		}
		actM.Iter(func(i int, part types.Part) error {
			act := string(part.Get())
			if _, exists := receivedMsgs[act]; exists {
				t.Errorf("Duplicate message: %v", act)
			} else {
				receivedMsgs[act] = struct{}{}
			}
			if _, exists := testMsgs[act]; !exists {
				t.Errorf("Unexpected message: %v", act)
			}
			delete(testMsgs, act)
			return nil
		})
		if err = ackFn(ctx, response.NewAck()); err != nil {
			t.Error(err)
		}
		lMsgs = len(testMsgs)
	}

	input.CloseAsync()
	output.CloseAsync()
	if err = input.WaitForClose(time.Second); err != nil {
		t.Error(err)
	}
	if err = output.WaitForClose(time.Second); err != nil {
		t.Error(err)
	}
}

func checkALOSynchronousAndDieAsync(
	outputCtr func() (writer.Type, error),
	inputCtr func() (reader.Async, error),
	t *testing.T,
) {
	ctx, done := context.WithTimeout(context.Background(), time.Second*60)
	defer done()

	output, err := outputCtr()
	if err != nil {
		t.Fatal(err)
	}
	input, err := inputCtr()
	if err != nil {
		t.Fatal(err)
	}

	N := 100

	testMsgs := map[string]struct{}{}
	for i := 0; i < N; i++ {
		str := fmt.Sprintf("hello world: %v", i)
		testMsgs[str] = struct{}{}
		msg := message.New([][]byte{
			[]byte(str),
		})
		if err = output.Write(msg); err != nil {
			t.Fatal(err)
		}
	}

	receivedMsgs := map[string]struct{}{}
	for i := 0; i < len(testMsgs); i++ {
		var actM types.Message
		var ackFn reader.AsyncAckFn
		if actM, ackFn, err = input.ReadWithContext(ctx); err != nil {
			if err == types.ErrNotConnected {
				if err = input.ConnectWithContext(ctx); err != nil {
					t.Fatal(err)
				}
				actM, ackFn, err = input.ReadWithContext(ctx)
			}
			if err != nil {
				t.Fatal(err)
			}
		}
		ackErr := false
		var res types.Response = response.NewAck()
		if i%10 == 0 {
			res = response.NewError(errors.New("nah"))
			ackErr = true
		}
		if !ackErr {
			actM.Iter(func(i int, part types.Part) error {
				act := string(part.Get())

				if _, exists := receivedMsgs[act]; exists {
					t.Errorf("Duplicate message: %v", act)
				} else {
					receivedMsgs[act] = struct{}{}
				}
				if _, exists := testMsgs[act]; !exists {
					t.Errorf("Unexpected message: %v", act)
				}
				delete(testMsgs, act)
				return nil
			})
		}
		if err = ackFn(ctx, res); err != nil {
			t.Error(err)
		}
	}

	lMsgs := len(testMsgs)
	if lMsgs == 0 {
		t.Error("Expected remaining messages")
	}

	input.CloseAsync()
	if err = input.WaitForClose(time.Second); err != nil {
		t.Error(err)
	}
	if input, err = inputCtr(); err != nil {
		t.Fatal(err)
	}

	for lMsgs > 0 {
		var actM types.Message
		var ackFn reader.AsyncAckFn
		if actM, ackFn, err = input.ReadWithContext(ctx); err != nil {
			if err == types.ErrNotConnected {
				if err = input.ConnectWithContext(ctx); err != nil {
					t.Fatal(err)
				}
				actM, ackFn, err = input.ReadWithContext(ctx)
			}
			if err != nil {
				t.Fatal(err)
			}
		}
		actM.Iter(func(i int, part types.Part) error {
			act := string(part.Get())
			if _, exists := receivedMsgs[act]; exists {
				t.Errorf("Duplicate message: %v", act)
			} else {
				receivedMsgs[act] = struct{}{}
			}
			if _, exists := testMsgs[act]; !exists {
				t.Errorf("Unexpected message: %v", act)
			}
			delete(testMsgs, act)
			return nil
		})
		if err = ackFn(ctx, response.NewAck()); err != nil {
			t.Error(err)
		}
		lMsgs = len(testMsgs)
	}

	input.CloseAsync()
	output.CloseAsync()
	if err = input.WaitForClose(time.Second); err != nil {
		t.Error(err)
	}
	if err = output.WaitForClose(time.Second); err != nil {
		t.Error(err)
	}
}

func checkALOParallelAsync(
	outputCtr func() (writer.Type, error),
	inputCtr func() (reader.Async, error),
	N int,
	t *testing.T,
) {
	ctx, done := context.WithTimeout(context.Background(), time.Second*60)
	defer done()

	output, err := outputCtr()
	if err != nil {
		t.Fatal(err)
	}
	input, err := inputCtr()
	if err != nil {
		t.Fatal(err)
	}

	testMsgs := map[string]struct{}{}
	for i := 0; i < N; i++ {
		str := fmt.Sprintf("hello world: %v", i)
		testMsgs[str] = struct{}{}
		msg := message.New([][]byte{
			[]byte(str),
		})
		if err = output.Write(msg); err != nil {
			t.Fatal(err)
		}
	}

	receivedMsgs := map[string]struct{}{}
	ackFns := []reader.AsyncAckFn{}
	for i := 0; i < len(testMsgs); i++ {
		var actM types.Message
		var ackFn reader.AsyncAckFn
		if actM, ackFn, err = input.ReadWithContext(ctx); err != nil {
			if err == types.ErrNotConnected {
				if err = input.ConnectWithContext(ctx); err != nil {
					t.Fatalf("Failed at '%v' read: %v", i, err)
				}
				actM, ackFn, err = input.ReadWithContext(ctx)
			}
			if err != nil {
				t.Fatalf("Failed at '%v' read: %v", i, err)
			}
		}
		ackFns = append(ackFns, ackFn)
		if i%10 != 0 {
			actM.Iter(func(i int, part types.Part) error {
				act := string(part.Get())

				if _, exists := receivedMsgs[act]; exists {
					t.Errorf("Duplicate message: %v", act)
				} else {
					receivedMsgs[act] = struct{}{}
				}
				if _, exists := testMsgs[act]; !exists {
					t.Errorf("Unexpected message: %v", act)
				}
				delete(testMsgs, act)
				return nil
			})
		}
	}

	for i, ackFn := range ackFns {
		var res types.Response = response.NewAck()
		if i%10 == 0 {
			res = response.NewError(errors.New("nah"))
		}
		if err = ackFn(ctx, res); err != nil {
			t.Error(err)
		}
	}

	lMsgs := len(testMsgs)
	if lMsgs == 0 {
		t.Error("Expected remaining messages")
	}

	for lMsgs > 0 {
		var actM types.Message
		var ackFn reader.AsyncAckFn
		if actM, ackFn, err = input.ReadWithContext(ctx); err != nil {
			if err == types.ErrNotConnected {
				if err = input.ConnectWithContext(ctx); err != nil {
					t.Fatal(err)
				}
				actM, ackFn, err = input.ReadWithContext(ctx)
			}
			if err != nil {
				t.Fatal(err)
			}
		}
		actM.Iter(func(i int, part types.Part) error {
			act := string(part.Get())
			if _, exists := receivedMsgs[act]; exists {
				t.Errorf("Duplicate message: %v", act)
			} else {
				receivedMsgs[act] = struct{}{}
			}
			if _, exists := testMsgs[act]; !exists {
				t.Errorf("Unexpected message: %v", act)
			}
			delete(testMsgs, act)
			return nil
		})
		if err = ackFn(ctx, response.NewAck()); err != nil {
			t.Error(err)
		}
		lMsgs = len(testMsgs)
	}

	input.CloseAsync()
	output.CloseAsync()
	if err = input.WaitForClose(time.Second); err != nil {
		t.Error(err)
	}
	if err = output.WaitForClose(time.Second); err != nil {
		t.Error(err)
	}
}
