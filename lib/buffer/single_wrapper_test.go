package buffer

import (
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/windhooked/benthos/v3/lib/buffer/single"
	"github.com/windhooked/benthos/v3/lib/log"
	"github.com/windhooked/benthos/v3/lib/message"
	"github.com/windhooked/benthos/v3/lib/metrics"
	"github.com/windhooked/benthos/v3/lib/response"
	"github.com/windhooked/benthos/v3/lib/types"
)

//------------------------------------------------------------------------------

func TestBasicMemoryBuffer(t *testing.T) {
	var incr, total uint8 = 100, 50

	tChan := make(chan types.Transaction)
	resChan := make(chan types.Response)

	conf := NewConfig()
	b := NewSingleWrapper(conf, single.NewMemory(single.MemoryConfig{
		Limit: int(incr+15) * int(total),
	}), log.Noop(), metrics.Noop())
	if err := b.Consume(tChan); err != nil {
		t.Fatal(err)
	}

	var i uint8

	// Check correct flow no blocking
	for ; i < total; i++ {
		msgBytes := make([][]byte, 1)
		msgBytes[0] = make([]byte, int(incr))
		msgBytes[0][0] = byte(i)

		select {
		// Send to buffer
		case tChan <- types.NewTransaction(message.New(msgBytes), resChan):
		case <-time.After(time.Second):
			t.Errorf("Timed out waiting for unbuffered message %v send", i)
			return
		}

		// Instant response from buffer
		select {
		case res := <-resChan:
			if res.Error() != nil {
				t.Error(res.Error())
			}
		case <-time.After(time.Second):
			t.Errorf("Timed out waiting for unbuffered message %v response", i)
			return
		}

		// Receive on output
		var outTr types.Transaction
		select {
		case outTr = <-b.TransactionChan():
			if actual := uint8(outTr.Payload.Get(0).Get()[0]); actual != i {
				t.Errorf("Wrong order receipt of unbuffered message receive: %v != %v", actual, i)
			}
		case <-time.After(time.Second):
			t.Errorf("Timed out waiting for unbuffered message %v read", i)
			return
		}

		// Response from output
		select {
		case outTr.ResponseChan <- response.NewAck():
		case <-time.After(time.Second):
			t.Errorf("Timed out waiting for unbuffered response send back %v", i)
			return
		}
	}

	for i = 0; i < total; i++ {
		msgBytes := make([][]byte, 1)
		msgBytes[0] = make([]byte, int(incr))
		msgBytes[0][0] = byte(i)

		select {
		case tChan <- types.NewTransaction(message.New(msgBytes), resChan):
		case <-time.After(time.Second):
			t.Errorf("Timed out waiting for buffered message %v send", i)
			return
		}
		select {
		case res := <-resChan:
			if res.Error() != nil {
				t.Error(res.Error())
			}
		case <-time.After(time.Second):
			t.Errorf("Timed out waiting for buffered message %v response", i)
			return
		}
	}

	// Should have reached limit here
	msgBytes := make([][]byte, 1)
	msgBytes[0] = make([]byte, int(incr))

	select {
	case tChan <- types.NewTransaction(message.New(msgBytes), resChan):
	case <-time.After(time.Second):
		t.Errorf("Timed out waiting for final buffered message send")
		return
	}

	// Response should block until buffer is relieved
	select {
	case res := <-resChan:
		if res.Error() != nil {
			t.Error(res.Error())
		} else {
			t.Errorf("Overflowed response returned before timeout")
		}
		return
	case <-time.After(100 * time.Millisecond):
	}

	var outTr types.Transaction

	// Extract last message
	select {
	case outTr = <-b.TransactionChan():
		if actual := uint8(outTr.Payload.Get(0).Get()[0]); actual != 0 {
			t.Errorf("Wrong order receipt of buffered message receive: %v != %v", actual, 0)
		}
		outTr.ResponseChan <- response.NewAck()
	case <-time.After(time.Second):
		t.Errorf("Timed out waiting for final buffered message read")
		return
	}

	// Response from the last attempt should no longer be blocking
	select {
	case res := <-resChan:
		if res.Error() != nil {
			t.Error(res.Error())
		}
	case <-time.After(100 * time.Millisecond):
		t.Errorf("Final buffered response blocked")
	}

	// Extract all other messages
	for i = 1; i < total; i++ {
		select {
		case outTr = <-b.TransactionChan():
			if actual := uint8(outTr.Payload.Get(0).Get()[0]); actual != i {
				t.Errorf("Wrong order receipt of buffered message: %v != %v", actual, i)
			}
		case <-time.After(time.Second):
			t.Errorf("Timed out waiting for buffered message %v read", i)
			return
		}

		select {
		case outTr.ResponseChan <- response.NewAck():
		case <-time.After(time.Second):
			t.Errorf("Timed out waiting for buffered response send back %v", i)
			return
		}
	}

	// Get final message
	select {
	case outTr = <-b.TransactionChan():
	case <-time.After(time.Second):
		t.Errorf("Timed out waiting for buffered message %v read", i)
		return
	}

	select {
	case outTr.ResponseChan <- response.NewAck():
	case <-time.After(time.Second):
		t.Errorf("Timed out waiting for buffered response send back %v", i)
		return
	}

	b.CloseAsync()
	b.WaitForClose(time.Second)

	close(resChan)
	close(tChan)
}

func TestBufferClosing(t *testing.T) {
	var incr, total uint8 = 100, 5

	tChan := make(chan types.Transaction)
	resChan := make(chan types.Response)

	conf := NewConfig()
	b := NewSingleWrapper(conf, single.NewMemory(single.MemoryConfig{
		Limit: int(incr+15) * int(total),
	}), log.Noop(), metrics.Noop())
	if err := b.Consume(tChan); err != nil {
		t.Fatal(err)
	}

	var i uint8

	// Populate buffer with some messages
	for i = 0; i < total; i++ {
		msgBytes := make([][]byte, 1)
		msgBytes[0] = make([]byte, int(incr))
		msgBytes[0][0] = byte(i)

		select {
		case tChan <- types.NewTransaction(message.New(msgBytes), resChan):
		case <-time.After(time.Second):
			t.Errorf("Timed out waiting for buffered message %v send", i)
			return
		}
		select {
		case res := <-resChan:
			if res.Error() != nil {
				t.Error(res.Error())
			}
		case <-time.After(time.Second):
			t.Errorf("Timed out waiting for buffered message %v response", i)
			return
		}
	}

	// Close input, this should prompt the stack buffer to CloseOnceEmpty().
	close(tChan)

	// Receive all of those messages from the buffer
	for i = 0; i < total; i++ {
		select {
		case val := <-b.TransactionChan():
			if actual := uint8(val.Payload.Get(0).Get()[0]); actual != i {
				t.Errorf("Wrong order receipt of buffered message receive: %v != %v", actual, i)
			}
			val.ResponseChan <- response.NewAck()
		case <-time.After(time.Second):
			t.Errorf("Timed out waiting for final buffered message read")
			return
		}
	}

	// The buffer should now be closed, therefore so should our read channel.
	select {
	case _, open := <-b.TransactionChan():
		if open {
			t.Error("Reader channel still open after clearing buffer")
		}
	case <-time.After(time.Second):
		t.Errorf("Timed out waiting for final buffered message read")
		return
	}

	// Should already be shut down.
	b.WaitForClose(time.Second)
}

func BenchmarkSingleMem(b *testing.B) {
	tChan := make(chan types.Transaction)
	resChan := make(chan types.Response)

	conf := NewConfig()
	buffer := NewSingleWrapper(conf, single.NewMemory(single.MemoryConfig{
		Limit: 50000000,
	}), log.Noop(), metrics.Noop())
	if err := buffer.Consume(tChan); err != nil {
		b.Fatal(err)
	}

	contents := [][]byte{
		make([]byte, 1024*1024*1),
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		select {
		case tChan <- types.NewTransaction(message.New(contents), resChan):
		case <-time.After(time.Second):
			b.Errorf("Timed out waiting for buffered message %v send", i)
			return
		}
		select {
		case res := <-resChan:
			if res.Error() != nil {
				b.Error(res.Error())
			}
		case <-time.After(time.Second):
			b.Errorf("Timed out waiting for buffered message %v response", i)
			return
		}

		select {
		case val := <-buffer.TransactionChan():
			val.ResponseChan <- response.NewAck()
		case <-time.After(time.Second):
			b.Errorf("Timed out waiting for final buffered message read")
			return
		}
	}
	b.StopTimer()

	buffer.CloseAsync()
	buffer.WaitForClose(time.Second)
}

func BenchmarkSingleMmap(b *testing.B) {
	dir, err := ioutil.TempDir("", "benthos_mmap_test")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(dir)

	tChan := make(chan types.Transaction)
	resChan := make(chan types.Response)

	mmapConf := single.NewMmapBufferConfig()
	mmapConf.CleanUp = true
	mmapConf.FileSize = 50000000
	mmapConf.Path = dir

	mmap, err := single.NewMmapBuffer(mmapConf, log.Noop(), metrics.Noop())
	if err != nil {
		b.Fatal(err)
	}

	conf := NewConfig()
	buffer := NewSingleWrapper(conf, mmap, log.Noop(), metrics.Noop())
	if err := buffer.Consume(tChan); err != nil {
		b.Error(err)
		return
	}

	contents := [][]byte{
		make([]byte, 1024*1024*1),
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		select {
		case tChan <- types.NewTransaction(message.New(contents), resChan):
		case <-time.After(time.Second):
			b.Errorf("Timed out waiting for buffered message %v send", i)
			return
		}
		select {
		case res := <-resChan:
			if res.Error() != nil {
				b.Error(res.Error())
			}
		case <-time.After(time.Second):
			b.Errorf("Timed out waiting for buffered message %v response", i)
			return
		}

		select {
		case val := <-buffer.TransactionChan():
			val.ResponseChan <- response.NewAck()
		case <-time.After(time.Second):
			b.Errorf("Timed out waiting for final buffered message read")
			return
		}
	}
	b.StopTimer()

	buffer.CloseAsync()
	buffer.WaitForClose(time.Second)
}

//------------------------------------------------------------------------------
