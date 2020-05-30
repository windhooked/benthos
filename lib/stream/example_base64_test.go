package stream

import (
	"bytes"
	"encoding/base64"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/windhooked/benthos/v3/lib/input"
	"github.com/windhooked/benthos/v3/lib/output"
	"github.com/windhooked/benthos/v3/lib/types"
)

// Base64Encoder is a types.Processor implementation that base64 encodes
// all messages travelling through a Benthos stream.
type Base64Encoder struct{}

// ProcessMessage base64 encodes all messages.
func (p Base64Encoder) ProcessMessage(m types.Message) ([]types.Message, types.Response) {
	// Create a copy of the original message
	result := m.Copy()

	// For each message part replace its contents with the base64 encoded
	// version.
	result.Iter(func(i int, part types.Part) error {
		var buf bytes.Buffer

		e := base64.NewEncoder(base64.StdEncoding, &buf)
		e.Write(part.Get())
		e.Close()

		part.Set(buf.Bytes())
		return nil
	})

	return []types.Message{result}, nil
}

// CloseAsync shuts down the processor and stops processing requests.
func (p Base64Encoder) CloseAsync() {
	// Do nothing as our processor doesn't require resource cleanup.
}

// WaitForClose blocks until the processor has closed down.
func (p Base64Encoder) WaitForClose(timeout time.Duration) error {
	// Do nothing as our processor doesn't require resource cleanup.
	return nil
}

// ExampleBase64Encoder demonstrates running a Kafka to Kafka stream where each
// incoming message is encoded with base64.
func Example_base64Encoder() {
	conf := NewConfig()

	conf.Input.Type = input.TypeKafka
	conf.Input.Kafka.Addresses = []string{
		"localhost:9092",
	}
	conf.Input.Kafka.Topic = "example_topic_one"

	conf.Output.Type = output.TypeKafka
	conf.Output.Kafka.Addresses = []string{
		"localhost:9092",
	}
	conf.Output.Kafka.Topic = "example_topic_two"

	s, err := New(conf, OptAddProcessors(func() (types.Processor, error) {
		return Base64Encoder{}, nil
	}))
	if err != nil {
		panic(err)
	}

	defer s.Stop(time.Second)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Wait for termination signal
	select {
	case <-sigChan:
		log.Println("Received SIGTERM, the service is closing.")
	}
}
