// +build integration

package integration

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/windhooked/benthos/v3/lib/input/reader"
	"github.com/windhooked/benthos/v3/lib/log"
	"github.com/windhooked/benthos/v3/lib/message"
	"github.com/windhooked/benthos/v3/lib/metrics"
	"github.com/windhooked/benthos/v3/lib/output/writer"
	"github.com/windhooked/benthos/v3/lib/types"
	"github.com/ory/dockertest"
)

func TestRedisPubSubIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Parallel()

	pool, err := dockertest.NewPool("")
	if err != nil {
		t.Skipf("Could not connect to docker: %s", err)
	}
	pool.MaxWait = time.Second * 30

	resource, err := pool.Run("redis", "latest", nil)
	if err != nil {
		t.Fatalf("Could not start resource: %s", err)
	}
	defer func() {
		if err = pool.Purge(resource); err != nil {
			t.Logf("Failed to clean up docker resource: %v", err)
		}
	}()
	resource.Expire(900)

	url := fmt.Sprintf("tcp://localhost:%v", resource.GetPort("6379/tcp"))

	if err = pool.Retry(func() error {
		conf := reader.NewRedisPubSubConfig()
		conf.URL = url
		conf.Channels = []string{"foo_chan"}

		r, cErr := reader.NewRedisPubSub(conf, log.Noop(), metrics.Noop())
		if cErr != nil {
			return cErr
		}
		cErr = r.Connect()

		r.CloseAsync()
		return cErr
	}); err != nil {
		t.Fatalf("Could not connect to docker resource: %s", err)
	}

	t.Run("TestRedisPubSubSinglePart", func(te *testing.T) {
		testRedisPubSubSinglePart(url, te)
	})
	t.Run("TestRedisPubSubSinglePartGlobs", func(te *testing.T) {
		testRedisPubSubSinglePartGlobs(url, te)
	})
	t.Run("TestRedisPubSubMultiplePart", func(te *testing.T) {
		testRedisPubSubMultiplePart(url, te)
	})
	t.Run("TestRedisPubSubDisconnect", func(te *testing.T) {
		testRedisPubSubDisconnect(url, te)
	})
}

func createRedisPubSubInputOutput(
	inConf reader.RedisPubSubConfig, outConf writer.RedisPubSubConfig,
) (mInput reader.Type, mOutput writer.Type, err error) {
	if mInput, err = reader.NewRedisPubSub(inConf, log.Noop(), metrics.Noop()); err != nil {
		return
	}
	if err = mInput.Connect(); err != nil {
		return
	}
	if mOutput, err = writer.NewRedisPubSub(outConf, log.Noop(), metrics.Noop()); err != nil {
		return
	}
	if err = mOutput.Connect(); err != nil {
		return
	}
	return
}

func testRedisPubSubSinglePart(url string, t *testing.T) {
	inConf := reader.NewRedisPubSubConfig()
	inConf.URL = url
	inConf.Channels = []string{"benthos_test_pubsub_single_part"}

	outConf := writer.NewRedisPubSubConfig()
	outConf.URL = url
	outConf.Channel = "benthos_test_pubsub_single_part"

	mInput, mOutput, err := createRedisPubSubInputOutput(inConf, outConf)
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		mInput.CloseAsync()
		if cErr := mInput.WaitForClose(time.Second); cErr != nil {
			t.Error(cErr)
		}
		mOutput.CloseAsync()
		if cErr := mOutput.WaitForClose(time.Second); cErr != nil {
			t.Error(cErr)
		}
	}()

	N := 10

	wg := sync.WaitGroup{}
	wg.Add(N)

	testMsgs := map[string]struct{}{}
	for i := 0; i < N; i++ {
		str := fmt.Sprintf("hello world: %v", i)
		testMsgs[str] = struct{}{}
		go func(testStr string) {
			msg := message.New([][]byte{
				[]byte(testStr),
			})
			msg.Get(0).Metadata().Set("foo", "bar")
			msg.Get(0).Metadata().Set("root_foo", "bar2")
			if gerr := mOutput.Write(msg); gerr != nil {
				t.Fatal(gerr)
			}
			wg.Done()
		}(str)
	}

	lMsgs := len(testMsgs)
	for lMsgs > 0 {
		var actM types.Message
		actM, err = mInput.Read()
		if err != nil {
			t.Error(err)
		} else {
			act := string(actM.Get(0).Get())
			if _, exists := testMsgs[act]; !exists {
				t.Errorf("Unexpected message: %v", act)
			}
			delete(testMsgs, act)
		}
		if err = mInput.Acknowledge(nil); err != nil {
			t.Error(err)
		}
		lMsgs = len(testMsgs)
	}

	wg.Wait()
}

func testRedisPubSubSinglePartGlobs(url string, t *testing.T) {
	inConf := reader.NewRedisPubSubConfig()
	inConf.URL = url
	inConf.UsePatterns = true
	inConf.Channels = []string{"benthos_test_*_single_part"}

	outConf := writer.NewRedisPubSubConfig()
	outConf.URL = url
	outConf.Channel = "benthos_test_pubsub_single_part"

	mInput, mOutput, err := createRedisPubSubInputOutput(inConf, outConf)
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		mInput.CloseAsync()
		if cErr := mInput.WaitForClose(time.Second); cErr != nil {
			t.Error(cErr)
		}
		mOutput.CloseAsync()
		if cErr := mOutput.WaitForClose(time.Second); cErr != nil {
			t.Error(cErr)
		}
	}()

	N := 10

	wg := sync.WaitGroup{}
	wg.Add(N)

	testMsgs := map[string]struct{}{}
	for i := 0; i < N; i++ {
		str := fmt.Sprintf("hello world: %v", i)
		testMsgs[str] = struct{}{}
		go func(testStr string) {
			msg := message.New([][]byte{
				[]byte(testStr),
			})
			msg.Get(0).Metadata().Set("foo", "bar")
			msg.Get(0).Metadata().Set("root_foo", "bar2")
			if gerr := mOutput.Write(msg); gerr != nil {
				t.Fatal(gerr)
			}
			wg.Done()
		}(str)
	}

	lMsgs := len(testMsgs)
	for lMsgs > 0 {
		var actM types.Message
		actM, err = mInput.Read()
		if err != nil {
			t.Error(err)
		} else {
			act := string(actM.Get(0).Get())
			if _, exists := testMsgs[act]; !exists {
				t.Errorf("Unexpected message: %v", act)
			}
			delete(testMsgs, act)
		}
		if err = mInput.Acknowledge(nil); err != nil {
			t.Error(err)
		}
		lMsgs = len(testMsgs)
	}

	wg.Wait()
}

func testRedisPubSubMultiplePart(url string, t *testing.T) {
	inConf := reader.NewRedisPubSubConfig()
	inConf.URL = url
	inConf.Channels = []string{"benthos_test_pubsub_multi_part"}

	outConf := writer.NewRedisPubSubConfig()
	outConf.URL = url
	outConf.Channel = "benthos_test_pubsub_multi_part"

	mInput, mOutput, err := createRedisPubSubInputOutput(inConf, outConf)
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		mInput.CloseAsync()
		if cErr := mInput.WaitForClose(time.Second); cErr != nil {
			t.Error(cErr)
		}
		mOutput.CloseAsync()
		if cErr := mOutput.WaitForClose(time.Second); cErr != nil {
			t.Error(cErr)
		}
	}()

	N := 10

	wg := sync.WaitGroup{}
	wg.Add(N)

	testMsgs := map[string]struct{}{}
	for i := 0; i < N; i++ {
		str1 := fmt.Sprintf("hello world: %v part 1", i)
		str2 := fmt.Sprintf("hello world: %v part 2", i)
		str3 := fmt.Sprintf("hello world: %v part 3", i)
		testMsgs[str1] = struct{}{}
		testMsgs[str2] = struct{}{}
		testMsgs[str3] = struct{}{}
		go func(testStr1, testStr2, testStr3 string) {
			msg := message.New([][]byte{
				[]byte(testStr1),
				[]byte(testStr2),
				[]byte(testStr3),
			})
			msg.Get(0).Metadata().Set("foo", "bar")
			msg.Get(1).Metadata().Set("root_foo", "bar2")
			if gerr := mOutput.Write(msg); gerr != nil {
				t.Fatal(gerr)
			}
			wg.Done()
		}(str1, str2, str3)
	}

	lMsgs := len(testMsgs)
	for lMsgs > 0 {
		var actM types.Message
		actM, err = mInput.Read()
		if err != nil {
			t.Error(err)
		} else {
			act := string(actM.Get(0).Get())
			if _, exists := testMsgs[act]; !exists {
				t.Errorf("Unexpected message: %v", act)
			}
			delete(testMsgs, act)
		}
		if err = mInput.Acknowledge(nil); err != nil {
			t.Error(err)
		}
		lMsgs = len(testMsgs)
	}

	wg.Wait()
}

func testRedisPubSubDisconnect(url string, t *testing.T) {
	inConf := reader.NewRedisPubSubConfig()
	inConf.URL = url
	inConf.Channels = []string{"benthos_test_pubsub_disconnect"}

	outConf := writer.NewRedisPubSubConfig()
	outConf.URL = url
	outConf.Channel = "benthos_test_pubsub_disconnect"

	mInput, mOutput, err := createRedisPubSubInputOutput(inConf, outConf)
	if err != nil {
		t.Fatal(err)
	}

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		mInput.CloseAsync()
		if cErr := mInput.WaitForClose(time.Second); cErr != nil {
			t.Error(cErr)
		}
		mOutput.CloseAsync()
		if cErr := mOutput.WaitForClose(time.Second); cErr != nil {
			t.Error(cErr)
		}
		wg.Done()
	}()

	if _, err = mInput.Read(); err != types.ErrTypeClosed && err != types.ErrNotConnected {
		t.Errorf("Wrong error: %v != %v", err, types.ErrTypeClosed)
	}

	wg.Wait()
}
