package manager

import (
	"os"
	"testing"

	"github.com/windhooked/benthos/v3/lib/cache"
	"github.com/windhooked/benthos/v3/lib/condition"
	"github.com/windhooked/benthos/v3/lib/log"
	"github.com/windhooked/benthos/v3/lib/metrics"
	"github.com/windhooked/benthos/v3/lib/processor"
	"github.com/windhooked/benthos/v3/lib/ratelimit"
	"github.com/windhooked/benthos/v3/lib/types"
)

//------------------------------------------------------------------------------

func TestManagerCache(t *testing.T) {
	testLog := log.New(os.Stdout, log.Config{LogLevel: "NONE"})

	conf := NewConfig()
	conf.Caches["foo"] = cache.NewConfig()
	conf.Caches["bar"] = cache.NewConfig()

	mgr, err := New(conf, nil, testLog, metrics.DudType{})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := mgr.GetCache("foo"); err != nil {
		t.Fatal(err)
	}
	if _, err := mgr.GetCache("bar"); err != nil {
		t.Fatal(err)
	}
	if _, err := mgr.GetCache("baz"); err != types.ErrCacheNotFound {
		t.Errorf("Wrong error returned: %v != %v", err, types.ErrCacheNotFound)
	}
}

func TestManagerBadCache(t *testing.T) {
	testLog := log.New(os.Stdout, log.Config{LogLevel: "NONE"})

	conf := NewConfig()
	badConf := cache.NewConfig()
	badConf.Type = "notexist"
	conf.Caches["bad"] = badConf

	if _, err := New(conf, nil, testLog, metrics.DudType{}); err == nil {
		t.Fatal("Expected error from bad cache")
	}
}

func TestManagerRateLimit(t *testing.T) {
	conf := NewConfig()
	conf.RateLimits["foo"] = ratelimit.NewConfig()
	conf.RateLimits["bar"] = ratelimit.NewConfig()

	mgr, err := New(conf, nil, log.Noop(), metrics.Noop())
	if err != nil {
		t.Fatal(err)
	}

	if _, err := mgr.GetRateLimit("foo"); err != nil {
		t.Fatal(err)
	}
	if _, err := mgr.GetRateLimit("bar"); err != nil {
		t.Fatal(err)
	}
	if _, err := mgr.GetRateLimit("baz"); err != types.ErrRateLimitNotFound {
		t.Errorf("Wrong error returned: %v != %v", err, types.ErrRateLimitNotFound)
	}
}

func TestManagerBadRateLimit(t *testing.T) {
	conf := NewConfig()
	badConf := ratelimit.NewConfig()
	badConf.Type = "notexist"
	conf.RateLimits["bad"] = badConf

	if _, err := New(conf, nil, log.Noop(), metrics.Noop()); err == nil {
		t.Fatal("Expected error from bad rate limit")
	}
}

func TestManagerCondition(t *testing.T) {
	testLog := log.New(os.Stdout, log.Config{LogLevel: "NONE"})

	conf := NewConfig()
	conf.Conditions["foo"] = condition.NewConfig()
	conf.Conditions["bar"] = condition.NewConfig()

	mgr, err := New(conf, nil, testLog, metrics.DudType{})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := mgr.GetCondition("foo"); err != nil {
		t.Fatal(err)
	}
	if _, err := mgr.GetCondition("bar"); err != nil {
		t.Fatal(err)
	}
	if _, err := mgr.GetCondition("baz"); err != types.ErrConditionNotFound {
		t.Errorf("Wrong error returned: %v != %v", err, types.ErrConditionNotFound)
	}
}

func TestManagerProcessor(t *testing.T) {
	conf := NewConfig()
	conf.Processors["foo"] = processor.NewConfig()
	conf.Processors["bar"] = processor.NewConfig()

	mgr, err := New(conf, nil, log.Noop(), metrics.Noop())
	if err != nil {
		t.Fatal(err)
	}

	if _, err := mgr.GetProcessor("foo"); err != nil {
		t.Fatal(err)
	}
	if _, err := mgr.GetProcessor("bar"); err != nil {
		t.Fatal(err)
	}
	if _, err := mgr.GetProcessor("baz"); err != types.ErrProcessorNotFound {
		t.Errorf("Wrong error returned: %v != %v", err, types.ErrProcessorNotFound)
	}
}

func TestManagerConditionRecursion(t *testing.T) {
	t.Skip("Not yet implemented")

	testLog := log.New(os.Stdout, log.Config{LogLevel: "NONE"})

	conf := NewConfig()

	fooConf := condition.NewConfig()
	fooConf.Type = "resource"
	fooConf.Resource = "bar"
	conf.Conditions["foo"] = fooConf

	barConf := condition.NewConfig()
	barConf.Type = "resource"
	barConf.Resource = "foo"
	conf.Conditions["bar"] = barConf

	if _, err := New(conf, nil, testLog, metrics.DudType{}); err == nil {
		t.Error("Expected error from recursive conditions")
	}
}

func TestManagerBadCondition(t *testing.T) {
	testLog := log.New(os.Stdout, log.Config{LogLevel: "NONE"})

	conf := NewConfig()
	badConf := condition.NewConfig()
	badConf.Type = "notexist"
	conf.Conditions["bad"] = badConf

	if _, err := New(conf, nil, testLog, metrics.DudType{}); err == nil {
		t.Fatal("Expected error from bad condition")
	}
}

func TestManagerPipeErrors(t *testing.T) {
	conf := NewConfig()
	mgr, err := New(conf, nil, log.Noop(), metrics.Noop())
	if err != nil {
		t.Fatal(err)
	}

	if _, err = mgr.GetPipe("does not exist"); err != types.ErrPipeNotFound {
		t.Errorf("Wrong error returned: %v != %v", err, types.ErrPipeNotFound)
	}
}

func TestManagerPipeGetSet(t *testing.T) {
	conf := NewConfig()
	mgr, err := New(conf, nil, log.Noop(), metrics.Noop())
	if err != nil {
		t.Fatal(err)
	}

	t1 := make(chan types.Transaction)
	t2 := make(chan types.Transaction)
	t3 := make(chan types.Transaction)

	mgr.SetPipe("foo", t1)
	mgr.SetPipe("bar", t3)

	var p <-chan types.Transaction
	if p, err = mgr.GetPipe("foo"); err != nil {
		t.Fatal(err)
	}
	if p != t1 {
		t.Error("Wrong transaction chan returned")
	}

	// Should be a noop
	mgr.UnsetPipe("foo", t2)
	if p, err = mgr.GetPipe("foo"); err != nil {
		t.Fatal(err)
	}
	if p != t1 {
		t.Error("Wrong transaction chan returned")
	}
	if p, err = mgr.GetPipe("bar"); err != nil {
		t.Fatal(err)
	}
	if p != t3 {
		t.Error("Wrong transaction chan returned")
	}

	mgr.UnsetPipe("foo", t1)
	if _, err = mgr.GetPipe("foo"); err != types.ErrPipeNotFound {
		t.Errorf("Wrong error returned: %v != %v", err, types.ErrPipeNotFound)
	}

	// Back to before
	mgr.SetPipe("foo", t1)
	if p, err = mgr.GetPipe("foo"); err != nil {
		t.Fatal(err)
	}
	if p != t1 {
		t.Error("Wrong transaction chan returned")
	}

	// Now replace pipe
	mgr.SetPipe("foo", t2)
	if p, err = mgr.GetPipe("foo"); err != nil {
		t.Fatal(err)
	}
	if p != t2 {
		t.Error("Wrong transaction chan returned")
	}
	if p, err = mgr.GetPipe("bar"); err != nil {
		t.Fatal(err)
	}
	if p != t3 {
		t.Error("Wrong transaction chan returned")
	}
}

//------------------------------------------------------------------------------
