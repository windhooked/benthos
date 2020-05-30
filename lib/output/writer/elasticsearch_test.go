// +build integration

package writer

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/windhooked/benthos/v3/lib/log"
	"github.com/windhooked/benthos/v3/lib/message"
	"github.com/windhooked/benthos/v3/lib/metrics"
	"github.com/olivere/elastic"
	"github.com/ory/dockertest"
)

func TestElasticIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	pool, err := dockertest.NewPool("")
	if err != nil {
		t.Skipf("Could not connect to docker: %s", err)
	}
	pool.MaxWait = time.Second * 30

	resource, err := pool.Run("elasticsearch", "7.5.1", []string{
		"discovery.type=single-node",
	})
	if err != nil {
		t.Fatalf("Could not start resource: %s", err)
	}

	urls := []string{fmt.Sprintf("http://127.0.0.1:%v", resource.GetPort("9200/tcp"))}

	var client *elastic.Client

	if err = pool.Retry(func() error {
		opts := []elastic.ClientOptionFunc{
			elastic.SetURL(urls...),
			elastic.SetHttpClient(&http.Client{
				Timeout: time.Second,
			}),
			elastic.SetSniff(false),
		}

		var cerr error
		client, cerr = elastic.NewClient(opts...)

		if cerr == nil {
			index := `{
	"settings":{
		"number_of_shards": 1,
		"number_of_replicas": 0
	},
	"mappings":{
		"properties": {
			"user":{
				"type":"keyword"
			},
			"message":{
				"type":"text",
				"store": true,
				"fielddata": true
			}
		}
	}
}`
			_, cerr = client.
				CreateIndex("test_conn_index").
				Timeout("20s").
				Body(index).
				Do(context.Background())
		}
		return cerr
	}); err != nil {
		t.Fatalf("Could not connect to docker resource: %s", err)
	}

	defer func() {
		if err = pool.Purge(resource); err != nil {
			t.Logf("Failed to clean up docker resource: %v", err)
		}
	}()

	t.Run("TestElasticNoIndex", func(te *testing.T) {
		testElasticNoIndex(urls, client, te)
	})

	t.Run("TestElasticParallelWrites", func(te *testing.T) {
		testElasticParallelWrites(urls, client, te)
	})

	t.Run("TestElasticErrorHandling", func(te *testing.T) {
		testElasticErrorHandling(urls, client, te)
	})

	t.Run("TestElasticConnect", func(te *testing.T) {
		testElasticConnect(urls, client, te)
	})

	t.Run("TestElasticIndexInterpolation", func(te *testing.T) {
		testElasticIndexInterpolation(urls, client, te)
	})

	t.Run("TestElasticBatch", func(te *testing.T) {
		testElasticBatch(urls, client, te)
	})
}

func testElasticNoIndex(urls []string, client *elastic.Client, t *testing.T) {
	conf := NewElasticsearchConfig()
	conf.Index = "does_not_exist"
	conf.ID = "foo-${!count(\"noIndexTest\")}"
	conf.URLs = urls
	conf.MaxRetries = 1
	conf.Backoff.MaxElapsedTime = "1s"
	conf.Sniff = false

	m, err := NewElasticsearch(conf, log.Noop(), metrics.Noop())
	if err != nil {
		t.Fatal(err)
	}

	if err = m.Connect(); err != nil {
		t.Error(err)
	}

	defer func() {
		m.CloseAsync()
		if cErr := m.WaitForClose(time.Second); cErr != nil {
			t.Error(cErr)
		}
	}()

	if err = m.Write(message.New([][]byte{[]byte(`{"user":"1","message":"hello world"}`)})); err != nil {
		t.Error(err)
	}

	if err = m.Write(message.New([][]byte{
		[]byte(`{"user":"2","message":"hello world"}`),
		[]byte(`{"user":"3","message":"hello world"}`),
	})); err != nil {
		t.Error(err)
	}

	for i := 0; i < 3; i++ {
		id := fmt.Sprintf("foo-%v", i+1)
		get, err := client.Get().
			Index("does_not_exist").
			Type("_doc").
			Id(id).
			Do(context.Background())
		if err != nil {
			t.Fatalf("Failed to get doc '%v': %v", id, err)
		}
		if !get.Found {
			t.Errorf("document %v not found", i)
		}
	}
}

func testElasticParallelWrites(urls []string, client *elastic.Client, t *testing.T) {
	conf := NewElasticsearchConfig()
	conf.Index = "new_index_parallel_writes"
	conf.ID = "${!json(\"key\")}"
	conf.URLs = urls
	conf.MaxRetries = 1
	conf.Backoff.MaxElapsedTime = "1s"
	conf.Sniff = false

	m, err := NewElasticsearch(conf, log.Noop(), metrics.Noop())
	if err != nil {
		t.Fatal(err)
	}

	if err = m.Connect(); err != nil {
		t.Error(err)
	}

	defer func() {
		m.CloseAsync()
		if cErr := m.WaitForClose(time.Second); cErr != nil {
			t.Error(cErr)
		}
	}()

	N := 10

	startChan := make(chan struct{})
	wg := sync.WaitGroup{}
	wg.Add(N)

	docs := map[string]string{}

	for i := 0; i < N; i++ {
		str := fmt.Sprintf(`{"key":"doc-%v","message":"foobar"}`, i)
		docs[fmt.Sprintf("doc-%v", i)] = str
		go func(content string) {
			<-startChan
			if lerr := m.Write(message.New([][]byte{[]byte(content)})); lerr != nil {
				t.Error(lerr)
			}
			wg.Done()
		}(str)
	}

	close(startChan)
	wg.Wait()

	for id, exp := range docs {
		get, err := client.Get().
			Index("new_index_parallel_writes").
			Type("_doc").
			Id(id).
			Do(context.Background())
		if err != nil {
			t.Fatalf("Failed to get doc '%v': %v", id, err)
		}
		if !get.Found {
			t.Errorf("document %v not found", id)
		} else {
			rawBytes, err := get.Source.MarshalJSON()
			if err != nil {
				t.Error(err)
			} else {
				if act := string(rawBytes); act != exp {
					t.Errorf("Wrong result: %v != %v", act, exp)
				}
			}
		}
	}
}

func testElasticErrorHandling(urls []string, client *elastic.Client, t *testing.T) {
	conf := NewElasticsearchConfig()
	conf.Index = "test_conn_index?"
	conf.ID = "foo-static"
	conf.URLs = urls
	conf.Backoff.MaxInterval = "1s"
	conf.Sniff = false

	m, err := NewElasticsearch(conf, log.Noop(), metrics.Noop())
	if err != nil {
		t.Fatal(err)
	}

	if err = m.Connect(); err != nil {
		t.Fatal(err)
	}

	defer func() {
		m.CloseAsync()
		if cErr := m.WaitForClose(time.Second); cErr != nil {
			t.Error(cErr)
		}
	}()

	if err = m.Write(message.New([][]byte{[]byte(`{"message":true}`)})); err == nil {
		t.Error("Expected error")
	}

	if err = m.Write(message.New([][]byte{[]byte(`{"message":"foo"}`), []byte(`{"message":"bar"}`)})); err == nil {
		t.Error("Expected error")
	}
}

func testElasticConnect(urls []string, client *elastic.Client, t *testing.T) {
	conf := NewElasticsearchConfig()
	conf.Index = "test_conn_index"
	conf.ID = "foo-${!count(\"foo\")}"
	conf.URLs = urls
	conf.Type = "_doc"
	conf.Sniff = false

	m, err := NewElasticsearch(conf, log.Noop(), metrics.Noop())
	if err != nil {
		t.Fatal(err)
	}

	if err = m.Connect(); err != nil {
		t.Fatal(err)
	}

	defer func() {
		m.CloseAsync()
		if cErr := m.WaitForClose(time.Second); cErr != nil {
			t.Error(cErr)
		}
	}()

	N := 10

	testMsgs := [][][]byte{}
	for i := 0; i < N; i++ {
		testMsgs = append(testMsgs, [][]byte{
			[]byte(fmt.Sprintf(`{"user":"%v","message":"hello world"}`, i)),
		})
	}
	for i := 0; i < N; i++ {
		if err = m.Write(message.New(testMsgs[i])); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < N; i++ {
		id := fmt.Sprintf("foo-%v", i+1)
		get, err := client.Get().
			Index("test_conn_index").
			Type("_doc").
			Id(id).
			Do(context.Background())
		if err != nil {
			t.Fatalf("Failed to get doc '%v': %v", id, err)
		}
		if !get.Found {
			t.Errorf("document %v not found", i)
		}

		var sourceBytes []byte
		sourceBytes, err = get.Source.MarshalJSON()
		if err != nil {
			t.Error(err)
		} else {
			if exp, act := string(testMsgs[i][0]), string(sourceBytes); exp != act {
				t.Errorf("wrong user field returned: %v != %v", act, exp)
			}
		}
	}
}

func testElasticIndexInterpolation(urls []string, client *elastic.Client, t *testing.T) {
	conf := NewElasticsearchConfig()
	conf.Index = "${!meta(\"index\")}"
	conf.ID = "bar-${!count(\"bar\")}"
	conf.URLs = urls
	conf.Type = "_doc"
	conf.Sniff = false

	m, err := NewElasticsearch(conf, log.Noop(), metrics.Noop())
	if err != nil {
		t.Fatal(err)
	}

	if err = m.Connect(); err != nil {
		t.Fatal(err)
	}

	defer func() {
		m.CloseAsync()
		if cErr := m.WaitForClose(time.Second); cErr != nil {
			t.Error(cErr)
		}
	}()

	N := 10

	testMsgs := [][][]byte{}
	for i := 0; i < N; i++ {
		testMsgs = append(testMsgs, [][]byte{
			[]byte(fmt.Sprintf(`{"user":"%v","message":"hello world"}`, i)),
		})
	}
	for i := 0; i < N; i++ {
		msg := message.New(testMsgs[i])
		msg.Get(0).Metadata().Set("index", "test_conn_index")
		if err = m.Write(msg); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < N; i++ {
		id := fmt.Sprintf("bar-%v", i+1)
		get, err := client.Get().
			Index("test_conn_index").
			Type("_doc").
			Id(id).
			Do(context.Background())
		if err != nil {
			t.Fatalf("Failed to get doc '%v': %v", id, err)
		}
		if !get.Found {
			t.Errorf("document %v not found", i)
		}

		var sourceBytes []byte
		sourceBytes, err = get.Source.MarshalJSON()
		if err != nil {
			t.Error(err)
		} else {
			if exp, act := string(testMsgs[i][0]), string(sourceBytes); exp != act {
				t.Errorf("wrong user field returned: %v != %v", act, exp)
			}
		}
	}
}

func testElasticBatch(urls []string, client *elastic.Client, t *testing.T) {
	conf := NewElasticsearchConfig()
	conf.Index = "${!meta(\"index\")}"
	conf.ID = "bar-${!count(\"bar\")}"
	conf.URLs = urls
	conf.Sniff = false
	conf.Type = "_doc"

	m, err := NewElasticsearch(conf, log.Noop(), metrics.Noop())
	if err != nil {
		t.Fatal(err)
	}

	if err = m.Connect(); err != nil {
		t.Fatal(err)
	}

	defer func() {
		m.CloseAsync()
		if cErr := m.WaitForClose(time.Second); cErr != nil {
			t.Error(cErr)
		}
	}()

	N := 10

	testMsg := [][]byte{}
	for i := 0; i < N; i++ {
		testMsg = append(testMsg,
			[]byte(fmt.Sprintf(`{"user":"%v","message":"hello world"}`, i)),
		)
	}
	msg := message.New(testMsg)
	for i := 0; i < N; i++ {
		msg.Get(i).Metadata().Set("index", "test_conn_index")
	}
	if err = m.Write(msg); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < N; i++ {
		id := fmt.Sprintf("bar-%v", i+1)
		get, err := client.Get().
			Index("test_conn_index").
			Type("_doc").
			Id(id).
			Do(context.Background())
		if err != nil {
			t.Fatalf("Failed to get doc '%v': %v", id, err)
		}
		if !get.Found {
			t.Errorf("document %v not found", i)
		}

		var sourceBytes []byte
		sourceBytes, err = get.Source.MarshalJSON()
		if err != nil {
			t.Error(err)
		} else {
			if exp, act := string(testMsg[i]), string(sourceBytes); exp != act {
				t.Errorf("wrong user field returned: %v != %v", act, exp)
			}
		}
	}
}
