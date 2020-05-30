package tests

import (
	"testing"
	"time"

	"github.com/windhooked/benthos/v3/lib/input"
	"github.com/windhooked/benthos/v3/lib/log"
	"github.com/windhooked/benthos/v3/lib/manager"
	"github.com/windhooked/benthos/v3/lib/metrics"
	"github.com/windhooked/benthos/v3/lib/types"
)

//------------------------------------------------------------------------------

func TestInprocDryRun(t *testing.T) {
	t.Parallel()

	mgr, err := manager.New(manager.NewConfig(), nil, log.Noop(), metrics.Noop())
	if err != nil {
		t.Fatal(err)
	}

	mgr.SetPipe("foo", make(chan types.Transaction))

	conf := input.NewConfig()
	conf.Inproc = "foo"

	var ip input.Type
	if ip, err = input.NewInproc(conf, mgr, log.Noop(), metrics.Noop()); err != nil {
		t.Fatal(err)
	}

	<-time.After(time.Millisecond * 100)

	ip.CloseAsync()
	if err = ip.WaitForClose(time.Second); err != nil {
		t.Error(err)
	}
}

func TestInprocDryRunNoConn(t *testing.T) {
	t.Parallel()

	mgr, err := manager.New(manager.NewConfig(), nil, log.Noop(), metrics.Noop())
	if err != nil {
		t.Fatal(err)
	}

	conf := input.NewConfig()
	conf.Inproc = "foo"

	var ip input.Type
	if ip, err = input.NewInproc(conf, mgr, log.Noop(), metrics.Noop()); err != nil {
		t.Fatal(err)
	}

	<-time.After(time.Millisecond * 100)

	ip.CloseAsync()
	if err = ip.WaitForClose(time.Second); err != nil {
		t.Error(err)
	}
}

//------------------------------------------------------------------------------
