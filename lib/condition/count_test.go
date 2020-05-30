package condition

import (
	"os"
	"testing"

	"github.com/windhooked/benthos/v3/lib/log"
	"github.com/windhooked/benthos/v3/lib/message"
	"github.com/windhooked/benthos/v3/lib/metrics"
)

func TestCountCheck(t *testing.T) {
	testLog := log.New(os.Stdout, log.Config{LogLevel: "NONE"})
	testMet := metrics.DudType{}

	conf := NewConfig()
	conf.Type = "count"
	conf.Count.Arg = 10

	c, err := New(conf, nil, testLog, testMet)
	if err != nil {
		t.Fatal(err)
	}

	for j := 0; j < 10; j++ {
		for i := 0; i < conf.Count.Arg-1; i++ {
			if !c.Check(message.New(nil)) {
				t.Error("Expected true result during count")
			}
		}
		if c.Check(message.New(nil)) {
			t.Error("Expected false result at end of count")
		}
	}
}
