package processor

import (
	"testing"
	"time"

	"github.com/windhooked/benthos/v3/lib/log"
	"github.com/windhooked/benthos/v3/lib/message"
	"github.com/windhooked/benthos/v3/lib/metrics"
)

func TestThrottle(t *testing.T) {
	conf := NewConfig()
	conf.Type = TypeThrottle
	conf.Throttle.Period = "1ns"

	throt, err := New(conf, nil, log.Noop(), metrics.Noop())
	if err != nil {
		t.Fatal(err)
	}

	msgIn := message.New(nil)
	msgsOut, res := throt.ProcessMessage(msgIn)
	if res != nil {
		t.Fatal(res.Error())
	}

	if exp, act := msgIn, msgsOut[0]; exp != act {
		t.Errorf("Wrong message returned: %v != %v", act, exp)
	}
}

func TestThrottle200Millisecond(t *testing.T) {
	conf := NewConfig()
	conf.Type = TypeThrottle
	conf.Throttle.Period = "200ms"

	throt, err := New(conf, nil, log.Noop(), metrics.Noop())
	if err != nil {
		t.Fatal(err)
	}

	tBefore := time.Now()
	throt.ProcessMessage(message.New(nil))
	tBetween := time.Now()
	throt.ProcessMessage(message.New(nil))
	tAfter := time.Now()

	if dur := tBetween.Sub(tBefore); dur > (time.Millisecond * 50) {
		t.Errorf("First message took too long")
	}
	if dur := tAfter.Sub(tBetween); dur < (time.Millisecond * 200) {
		t.Errorf("First message didn't take long enough")
	}
}

func TestThrottleBadPeriod(t *testing.T) {
	conf := NewConfig()
	conf.Type = TypeThrottle
	conf.Throttle.Period = "1gfdfgfdns"

	_, err := New(conf, nil, log.Noop(), metrics.Noop())
	if err == nil {
		t.Error("Expected error from bad duration")
	}
}
