package tests

import (
	"testing"
	"time"

	"github.com/windhooked/benthos/v3/lib/log"
	"github.com/windhooked/benthos/v3/lib/manager"
	"github.com/windhooked/benthos/v3/lib/metrics"
	"github.com/windhooked/benthos/v3/lib/output"
	"github.com/windhooked/benthos/v3/lib/types"
)

//------------------------------------------------------------------------------

func TestInproc(t *testing.T) {
	mgr, err := manager.New(manager.NewConfig(), nil, log.Noop(), metrics.Noop())
	if err != nil {
		t.Fatal(err)
	}

	conf := output.NewConfig()
	conf.Inproc = "foo"

	var ip output.Type
	if ip, err = output.NewInproc(conf, mgr, log.Noop(), metrics.Noop()); err != nil {
		t.Fatal(err)
	}

	if _, err = mgr.GetPipe("foo"); err != types.ErrPipeNotFound {
		t.Errorf("Wrong error returned: %v != %v", err, types.ErrPipeNotFound)
	}

	tinchan := make(chan types.Transaction)
	if err = ip.Consume(tinchan); err != nil {
		t.Fatal(err)
	}

	select {
	case tinchan <- types.NewTransaction(nil, nil):
	case <-time.After(time.Second):
		t.Error("Timed out")
	}

	var toutchan <-chan types.Transaction
	if toutchan, err = mgr.GetPipe("foo"); err != nil {
		t.Error(err)
	}

	select {
	case <-toutchan:
	case <-time.After(time.Second):
		t.Error("Timed out")
	}

	ip.CloseAsync()
	if err = ip.WaitForClose(time.Second); err != nil {
		t.Error(err)
	}

	select {
	case _, open := <-toutchan:
		if open {
			t.Error("transaction chan not closed")
		}
	case <-time.After(time.Second):
		t.Error("Timed out")
	}
	if _, err = mgr.GetPipe("foo"); err != types.ErrPipeNotFound {
		t.Errorf("Wrong error returned: %v != %v", err, types.ErrPipeNotFound)
	}
}

//------------------------------------------------------------------------------
