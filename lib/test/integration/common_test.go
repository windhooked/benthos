package integration

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/windhooked/benthos/v3/lib/input/reader"
	"github.com/windhooked/benthos/v3/lib/message"
	"github.com/windhooked/benthos/v3/lib/output/writer"
	"github.com/windhooked/benthos/v3/lib/types"
)

func checkALOSynchronous(
	outputCtr func() (writer.Type, error),
	inputCtr func() (reader.Type, error),
	t *testing.T,
) {
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
		if actM, err = input.Read(); err != nil {
			t.Fatal(err)
		}
		var ackErr error
		if i%10 == 0 {
			ackErr = errors.New("this is getting rejected just cus")
		}
		if ackErr == nil {
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
		if err = input.Acknowledge(ackErr); err != nil {
			t.Error(err)
		}
	}

	lMsgs := len(testMsgs)
	if lMsgs == 0 {
		t.Error("Expected remaining messages")
	}

	for lMsgs > 0 {
		var actM types.Message
		if actM, err = input.Read(); err != nil {
			t.Fatal(err)
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
		if err = input.Acknowledge(nil); err != nil {
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

func checkALOSynchronousAndDie(
	outputCtr func() (writer.Type, error),
	inputCtr func() (reader.Type, error),
	t *testing.T,
) {
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
		if actM, err = input.Read(); err != nil {
			t.Fatal(err)
		}
		var ackErr error
		if i%10 == 0 {
			ackErr = errors.New("this is getting rejected just cus")
		}
		if ackErr == nil {
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
		if err = input.Acknowledge(ackErr); err != nil {
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
		if actM, err = input.Read(); err != nil {
			t.Fatal(err)
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
		if err = input.Acknowledge(nil); err != nil {
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
