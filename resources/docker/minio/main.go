package main

import (
	"time"

	"github.com/windhooked/benthos/v3/lib/input"
	"github.com/windhooked/benthos/v3/lib/types"
	"github.com/windhooked/benthos/v3/lib/stream"
)

func main() {
	conf := stream.NewConfig()
	conf.Input.Type = input.TypeFile

	s, err := stream.New(conf, stream.OptAddProcessors(func() (types.Processor, error) {
		return CustomProcessor{}, nil
	}))
	if err != nil {
		panic(err)
	}
	defer s.Stop(time.Second)

}
