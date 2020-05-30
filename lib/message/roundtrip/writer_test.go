package roundtrip

import (
	"context"
	"testing"
	"time"

	"github.com/windhooked/benthos/v3/lib/message"
	"github.com/windhooked/benthos/v3/lib/types"
)

func TestWriter(t *testing.T) {
	impl := &resultStoreImpl{}
	w := Writer{}
	if err := w.Connect(); err != nil {
		t.Fatal(err)
	}

	ctx := context.WithValue(context.Background(), ResultStoreKey, impl)

	msg := message.New(nil)
	var p types.Part = message.NewPart([]byte("foo"))
	p = message.WithContext(ctx, p)
	msg.Append(p)
	msg.Append(message.NewPart([]byte("bar")))

	if err := w.Write(msg); err != nil {
		t.Fatal(err)
	}

	impl.Get()
	results := impl.Get()
	if len(results) != 1 {
		t.Fatalf("Wrong count of result batches: %v", len(results))
	}
	if results[0].Len() != 2 {
		t.Fatalf("Wrong count of messages: %v", results[0].Len())
	}
	if exp, act := "foo", string(results[0].Get(0).Get()); exp != act {
		t.Errorf("Wrong message contents: %v != %v", act, exp)
	}
	if exp, act := "bar", string(results[0].Get(1).Get()); exp != act {
		t.Errorf("Wrong message contents: %v != %v", act, exp)
	}
	if store := message.GetContext(results[0].Get(0)).Value(ResultStoreKey); store != nil {
		t.Error("Unexpected nested result store")
	}

	w.CloseAsync()
	if err := w.WaitForClose(time.Second); err != nil {
		t.Error(err)
	}
}
