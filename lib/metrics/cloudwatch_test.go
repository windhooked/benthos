package metrics

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/windhooked/benthos/v3/lib/log"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/cloudwatch/cloudwatchiface"
	"github.com/stretchr/testify/assert"
)

type mockCloudWatchClient struct {
	cloudwatchiface.CloudWatchAPI
	errs []error

	inputs []cloudwatch.PutMetricDataInput
}

func (m *mockCloudWatchClient) PutMetricData(input *cloudwatch.PutMetricDataInput) (*cloudwatch.PutMetricDataOutput, error) {
	m.inputs = append(m.inputs, *input)
	if len(m.errs) > 0 {
		err := m.errs[0]
		m.errs = m.errs[1:]
		return nil, err
	}
	return nil, nil
}

type checkedDatum struct {
	unit       string
	dimensions map[string]string
	value      float64
	values     map[float64]float64
}

func checkInput(i cloudwatch.PutMetricDataInput) map[string]checkedDatum {
	m := map[string]checkedDatum{}
	for _, datum := range i.MetricData {
		if datum.Timestamp == nil {
			panic("Timestamp not set")
		} else {
			tSince := time.Since(*datum.Timestamp)
			if tSince < 0 {
				panic("Timestamp from the future")
			}
			if tSince > time.Minute {
				panic("Timestamp from ages ago")
			}
		}
		d := checkedDatum{
			unit: *datum.Unit,
		}
		if len(datum.Dimensions) > 0 {
			d.dimensions = map[string]string{}
			for _, v := range datum.Dimensions {
				d.dimensions[*v.Name] = *v.Value
			}
		}
		if datum.Value != nil {
			d.value = *datum.Value
		} else {
			d.values = map[float64]float64{}
			for i, val := range datum.Values {
				if len(datum.Counts) > i {
					d.values[*val] = *datum.Counts[i]
				} else {
					d.values[*val] = 1
				}
			}
		}
		id := *datum.MetricName
		if len(d.dimensions) > 0 {
			id = fmt.Sprintf("%v:%v", id, d.dimensions)
		}
		m[id] = d
	}
	return m
}

func TestCloudWatchBasic(t *testing.T) {
	mockSvc := &mockCloudWatchClient{}

	cw := &CloudWatch{
		config:    NewCloudWatchConfig(),
		datumses:  map[string]*cloudWatchDatum{},
		datumLock: &sync.Mutex{},
		log:       log.Noop(),
		client:    mockSvc,
	}
	cw.ctx, cw.cancel = context.WithCancel(context.Background())

	ctrFoo := cw.GetCounter("counter.foo")
	ctrFoo.Incr(7)
	ctrFoo.Incr(6)

	ctrBar := cw.GetCounter("counter.bar")
	ctrBar.Incr(1)
	ctrBar.Incr(1)
	ctrBar.Incr(1)

	ggeFoo := cw.GetGauge("gauge.foo")
	ggeFoo.Set(111)
	ggeFoo.Set(111)
	ggeFoo.Set(72)

	ggeBar := cw.GetGauge("gauge.bar")
	ggeBar.Set(12)
	ggeBar.Set(90)

	tmgFoo := cw.GetTimer("timer.foo")
	tmgFoo.Timing(23000)
	tmgFoo.Timing(87001)
	tmgFoo.Timing(23010)

	cw.flush()

	ctrFoo.Incr(2)

	ctrBar.Incr(1)
	ctrBar.Incr(1)

	ggeFoo.Set(72)

	ggeBar.Set(7)
	ggeBar.Set(9000)

	tmgFoo.Timing(87120)
	tmgFoo.Timing(23400)

	cw.flush()

	assert.Equal(t, 2, len(mockSvc.inputs))

	assert.Equal(t, "Benthos", *mockSvc.inputs[0].Namespace)
	assert.Equal(t, "Benthos", *mockSvc.inputs[1].Namespace)

	assert.Equal(t, map[string]checkedDatum{
		"counter.foo": {
			unit:  "Count",
			value: 13,
		},
		"counter.bar": {
			unit:  "Count",
			value: 3,
		},
		"gauge.foo": {
			unit: "None",
			values: map[float64]float64{
				111: 2,
				72:  1,
			},
		},
		"gauge.bar": {
			unit: "None",
			values: map[float64]float64{
				12: 1,
				90: 1,
			},
		},
		"timer.foo": {
			unit: "Microseconds",
			values: map[float64]float64{
				23: 2,
				87: 1,
			},
		},
	}, checkInput(mockSvc.inputs[0]))

	assert.Equal(t, map[string]checkedDatum{
		"counter.foo": {
			unit:  "Count",
			value: 2,
		},
		"counter.bar": {
			unit:  "Count",
			value: 2,
		},
		"gauge.foo": {
			unit: "None",
			values: map[float64]float64{
				72: 1,
			},
		},
		"gauge.bar": {
			unit: "None",
			values: map[float64]float64{
				7:    1,
				9000: 1,
			},
		},
		"timer.foo": {
			unit: "Microseconds",
			values: map[float64]float64{
				23: 1,
				87: 1,
			},
		},
	}, checkInput(mockSvc.inputs[1]))
}

func TestCloudWatchMoreThan20Items(t *testing.T) {
	mockSvc := &mockCloudWatchClient{}

	cw := &CloudWatch{
		config:    NewCloudWatchConfig(),
		datumses:  map[string]*cloudWatchDatum{},
		datumLock: &sync.Mutex{},
		log:       log.Noop(),
		client:    mockSvc,
	}
	cw.ctx, cw.cancel = context.WithCancel(context.Background())

	exp := map[string]checkedDatum{}
	for i := 0; i < 30; i++ {
		name := fmt.Sprintf("counter.%v", i)
		ctr := cw.GetCounter(name)
		ctr.Incr(23)
		exp[name] = checkedDatum{
			unit:  "Count",
			value: 23,
		}
	}

	cw.flush()

	assert.Equal(t, 2, len(mockSvc.inputs))
	assert.Equal(t, 20, len(mockSvc.inputs[0].MetricData))
	assert.Equal(t, 10, len(mockSvc.inputs[1].MetricData))

	assert.Equal(t, "Benthos", *mockSvc.inputs[0].Namespace)
	assert.Equal(t, "Benthos", *mockSvc.inputs[1].Namespace)

	act := checkInput(mockSvc.inputs[0])
	for k, v := range checkInput(mockSvc.inputs[1]) {
		act[k] = v
	}
	assert.Equal(t, exp, act)
}

func TestCloudWatchMoreThan150Values(t *testing.T) {
	mockSvc := &mockCloudWatchClient{}

	cw := &CloudWatch{
		config:    NewCloudWatchConfig(),
		datumses:  map[string]*cloudWatchDatum{},
		datumLock: &sync.Mutex{},
		log:       log.Noop(),
		client:    mockSvc,
	}
	cw.ctx, cw.cancel = context.WithCancel(context.Background())

	exp := checkedDatum{
		unit:   "None",
		values: map[float64]float64{},
	}

	gge := cw.GetGauge("foo")
	for i := int64(0); i < 300; i++ {
		v := i
		if i >= 150 {
			gge.Set(i)
			v = i - 150
		} else {
			exp.values[float64(v)] = 2
		}
		gge.Set(v)
	}

	cw.flush()

	assert.Equal(t, 1, len(mockSvc.inputs))
	assert.Equal(t, 1, len(mockSvc.inputs[0].MetricData))

	assert.Equal(t, "Benthos", *mockSvc.inputs[0].Namespace)

	assert.Equal(t, 150, len(mockSvc.inputs[0].MetricData[0].Values))
	assert.Equal(t, map[string]checkedDatum{
		"foo": exp,
	}, checkInput(mockSvc.inputs[0]))
}

func TestCloudWatchMoreThan150RandomReduce(t *testing.T) {
	mockSvc := &mockCloudWatchClient{}

	cw := &CloudWatch{
		config:    NewCloudWatchConfig(),
		datumses:  map[string]*cloudWatchDatum{},
		datumLock: &sync.Mutex{},
		log:       log.Noop(),
		client:    mockSvc,
	}
	cw.ctx, cw.cancel = context.WithCancel(context.Background())

	gge := cw.GetGauge("foo")
	for i := int64(0); i < 300; i++ {
		gge.Set(i)
	}

	cw.flush()

	assert.Equal(t, 1, len(mockSvc.inputs))
	assert.Equal(t, 1, len(mockSvc.inputs[0].MetricData))

	assert.Equal(t, "Benthos", *mockSvc.inputs[0].Namespace)

	assert.Equal(t, 150, len(mockSvc.inputs[0].MetricData[0].Values))
}

func TestCloudWatchMoreThan150LiveReduce(t *testing.T) {
	mockSvc := &mockCloudWatchClient{}

	cw := &CloudWatch{
		config:    NewCloudWatchConfig(),
		datumses:  map[string]*cloudWatchDatum{},
		datumLock: &sync.Mutex{},
		log:       log.Noop(),
		client:    mockSvc,
	}
	cw.ctx, cw.cancel = context.WithCancel(context.Background())

	gge := cw.GetGauge("foo")
	for i := int64(0); i < 5000; i++ {
		gge.Set(i)
	}

	cw.flush()

	assert.Equal(t, 1, len(mockSvc.inputs))
	assert.Equal(t, 1, len(mockSvc.inputs[0].MetricData))

	assert.Equal(t, "Benthos", *mockSvc.inputs[0].Namespace)

	assert.Equal(t, 150, len(mockSvc.inputs[0].MetricData[0].Values))
}

func TestCloudWatchTags(t *testing.T) {
	mockSvc := &mockCloudWatchClient{}

	cw := &CloudWatch{
		config:    NewCloudWatchConfig(),
		datumses:  map[string]*cloudWatchDatum{},
		datumLock: &sync.Mutex{},
		log:       log.Noop(),
		client:    mockSvc,
	}
	cw.ctx, cw.cancel = context.WithCancel(context.Background())

	ctrFoo := cw.GetCounterVec("counter.foo", []string{"foo"})
	ggeFoo := cw.GetGaugeVec("gauge.foo", []string{"bar"})

	ctrFoo.With("one").Incr(1)
	ctrFoo.With("two").Incr(2)
	ggeFoo.With("third").Set(3)

	cw.flush()

	assert.Equal(t, 1, len(mockSvc.inputs))
	assert.Equal(t, "Benthos", *mockSvc.inputs[0].Namespace)
	assert.Equal(t, map[string]checkedDatum{
		"counter.foo:map[foo:one]": {
			unit: "Count",
			dimensions: map[string]string{
				"foo": "one",
			},
			value: 1,
		},
		"counter.foo:map[foo:two]": {
			unit: "Count",
			dimensions: map[string]string{
				"foo": "two",
			},
			value: 2,
		},
		"gauge.foo:map[bar:third]": {
			unit: "None",
			dimensions: map[string]string{
				"bar": "third",
			},
			values: map[float64]float64{
				3: 1,
			},
		},
	}, checkInput(mockSvc.inputs[0]))
}

func TestCloudWatchTagsMoreThan20(t *testing.T) {
	mockSvc := &mockCloudWatchClient{}

	cw := &CloudWatch{
		config:    NewCloudWatchConfig(),
		datumses:  map[string]*cloudWatchDatum{},
		datumLock: &sync.Mutex{},
		log:       log.Noop(),
		client:    mockSvc,
	}
	cw.ctx, cw.cancel = context.WithCancel(context.Background())

	expTagMap := map[string]string{}
	tagNames := []string{}
	tagValues := []string{}
	for i := 0; i < 30; i++ {
		name := fmt.Sprintf("%v", i)
		value := fmt.Sprintf("foo%v", i)
		tagNames = append(tagNames, name)
		tagValues = append(tagValues, value)
		if i < 10 {
			expTagMap[name] = value
		}
	}

	ctrFoo := cw.GetCounterVec("counter.foo", tagNames)
	ctrFoo.With(tagValues...).Incr(3)

	cw.flush()

	expKey := fmt.Sprintf("counter.foo:%v", expTagMap)

	assert.Equal(t, 1, len(mockSvc.inputs))
	assert.Equal(t, "Benthos", *mockSvc.inputs[0].Namespace)
	assert.Equal(t, 1, len(mockSvc.inputs[0].MetricData))
	assert.Equal(t, 10, len(mockSvc.inputs[0].MetricData[0].Dimensions))
	assert.Equal(t, map[string]checkedDatum{
		expKey: {
			unit:       "Count",
			dimensions: expTagMap,
			value:      3,
		},
	}, checkInput(mockSvc.inputs[0]))
}
