package single

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/windhooked/benthos/v3/lib/log"
	"github.com/windhooked/benthos/v3/lib/metrics"
)

func TestMmapCacheTracker(t *testing.T) {
	t.Skip("DEPRECATED")

	dir, err := ioutil.TempDir("", "benthos_test_")
	if err != nil {
		t.Error(err)
		return
	}

	defer cleanUpMmapDir(dir)

	conf := NewMmapCacheConfig()
	conf.FileSize = 1000
	conf.Path = dir

	cache, err := NewMmapCache(conf, log.New(os.Stdout, logConfig), metrics.DudType{})
	if err != nil {
		t.Error(err)
		return
	}
	cache.L.Lock()

	trackerBytes := cache.GetTracker()
	if trackerBytes == nil {
		t.Errorf("Tracker bytes were nil")
		return
	}

	if len(trackerBytes) != 16 {
		t.Errorf("Tracker was wrong length: %v != %v", len(trackerBytes), 16)
		return
	}

	if exp, act := trackerBytes[0], byte(0); exp != act {
		t.Errorf("Wrong byte from tracker: %v != %v", exp, act)
	}
	if exp, act := trackerBytes[1], byte(0); exp != act {
		t.Errorf("Wrong byte from tracker: %v != %v", exp, act)
	}
	if exp, act := trackerBytes[2], byte(0); exp != act {
		t.Errorf("Wrong byte from tracker: %v != %v", exp, act)
	}
	if exp, act := trackerBytes[3], byte(0); exp != act {
		t.Errorf("Wrong byte from tracker: %v != %v", exp, act)
	}

	trackerBytes[0] = byte(0)
	trackerBytes[1] = byte(1)
	trackerBytes[2] = byte(2)
	trackerBytes[3] = byte(3)

	if err = cache.RemoveAll(); err != nil {
		t.Error(err)
		return
	}

	cache.L.Unlock()
	cache, err = NewMmapCache(conf, log.New(os.Stdout, logConfig), metrics.DudType{})
	if err != nil {
		t.Error(err)
		return
	}
	cache.L.Lock()
	defer cache.L.Unlock()

	trackerBytes = cache.GetTracker()
	if trackerBytes == nil {
		t.Errorf("Tracker bytes were nil")
		return
	}

	if len(trackerBytes) != 16 {
		t.Errorf("Tracker was wrong length: %v != %v", len(trackerBytes), 16)
		return
	}

	if exp, act := trackerBytes[0], byte(0); exp != act {
		t.Errorf("Wrong byte from tracker: %v != %v", exp, act)
	}
	if exp, act := trackerBytes[1], byte(1); exp != act {
		t.Errorf("Wrong byte from tracker: %v != %v", exp, act)
	}
	if exp, act := trackerBytes[2], byte(2); exp != act {
		t.Errorf("Wrong byte from tracker: %v != %v", exp, act)
	}
	if exp, act := trackerBytes[3], byte(3); exp != act {
		t.Errorf("Wrong byte from tracker: %v != %v", exp, act)
	}
}

func TestMmapCacheIndexes(t *testing.T) {
	t.Skip("DEPRECATED")

	dir, err := ioutil.TempDir("", "benthos_test_")
	if err != nil {
		t.Error(err)
		return
	}

	defer cleanUpMmapDir(dir)

	conf := NewMmapCacheConfig()
	conf.FileSize = 1000
	conf.Path = dir

	cache, err := NewMmapCache(conf, log.New(os.Stdout, logConfig), metrics.DudType{})
	if err != nil {
		t.Error(err)
		return
	}
	cache.L.Lock()

	if err := cache.EnsureCached(20); err != nil {
		t.Error(err)
		return
	}

	bytes := cache.Get(20)
	if bytes == nil {
		t.Errorf("Index bytes were nil")
		return
	}

	if len(bytes) != conf.FileSize {
		t.Errorf("Index was wrong length: %v != %v", len(bytes), conf.FileSize)
		return
	}

	if exp, act := bytes[0], byte(0); exp != act {
		t.Errorf("Wrong byte from index: %v != %v", exp, act)
	}
	if exp, act := bytes[1], byte(0); exp != act {
		t.Errorf("Wrong byte from index: %v != %v", exp, act)
	}
	if exp, act := bytes[2], byte(0); exp != act {
		t.Errorf("Wrong byte from index: %v != %v", exp, act)
	}
	if exp, act := bytes[3], byte(0); exp != act {
		t.Errorf("Wrong byte from index: %v != %v", exp, act)
	}

	bytes[0] = byte(0)
	bytes[1] = byte(1)
	bytes[2] = byte(2)
	bytes[3] = byte(3)

	if err = cache.RemoveAll(); err != nil {
		t.Error(err)
		return
	}

	cache.L.Unlock()
	cache, err = NewMmapCache(conf, log.New(os.Stdout, logConfig), metrics.DudType{})
	if err != nil {
		t.Error(err)
		return
	}
	cache.L.Lock()
	defer cache.L.Unlock()

	if err := cache.EnsureCached(20); err != nil {
		t.Error(err)
		return
	}

	bytes = cache.Get(20)
	if bytes == nil {
		t.Errorf("Index bytes were nil")
		return
	}

	if len(bytes) != conf.FileSize {
		t.Errorf("Index was wrong length: %v != %v", len(bytes), conf.FileSize)
		return
	}

	if exp, act := bytes[0], byte(0); exp != act {
		t.Errorf("Wrong byte from index: %v != %v", exp, act)
	}
	if exp, act := bytes[1], byte(1); exp != act {
		t.Errorf("Wrong byte from index: %v != %v", exp, act)
	}
	if exp, act := bytes[2], byte(2); exp != act {
		t.Errorf("Wrong byte from index: %v != %v", exp, act)
	}
	if exp, act := bytes[3], byte(3); exp != act {
		t.Errorf("Wrong byte from index: %v != %v", exp, act)
	}
}

func TestMmapCacheRaces(t *testing.T) {
	t.Skip("DEPRECATED")

	dir, err := ioutil.TempDir("", "benthos_test_")
	if err != nil {
		t.Error(err)
		return
	}

	defer cleanUpMmapDir(dir)

	conf := NewMmapCacheConfig()
	conf.FileSize = 10
	conf.Path = dir

	cache, err := NewMmapCache(conf, log.New(os.Stdout, logConfig), metrics.DudType{})
	if err != nil {
		t.Error(err)
		return
	}
	cache.L.Lock()
	defer cache.L.Unlock()

	for i := 0; i < 100; i++ {
		if err := cache.EnsureCached(i); err != nil {
			t.Error(err)
			return
		}
	}

	for i := 0; i < 1000; i++ {
		bytes := cache.Get(20)
		if bytes == nil || len(bytes) == 0 {
			t.Errorf("Index %v bytes were nil or empty", i)
			return
		}

		bytes[0] = byte(1)
		bytes[1] = byte(2)
		bytes[2] = byte(3)
		bytes[3] = byte(4)
	}

}
