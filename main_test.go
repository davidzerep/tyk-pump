package main

import (
	"context"
	"fmt"

	"os"
	"os/signal"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/TykTechnologies/tyk-pump/analytics"
	"github.com/TykTechnologies/tyk-pump/pumps"
	"github.com/stretchr/testify/assert"
)

type MockedPump struct {
	CounterRequest int
	TurnedOff      bool
	pumps.CommonPumpConfig
}

func (p *MockedPump) GetName() string {
	return "Mocked Pump"
}

func (p *MockedPump) New() pumps.Pump {
	return &MockedPump{}
}
func (p *MockedPump) Init(config interface{}) error {
	return nil
}
func (p *MockedPump) WriteData(ctx context.Context, keys []interface{}) error {
	for range keys {
		p.CounterRequest++
	}
	return nil
}

func (p *MockedPump) Shutdown() error {
	p.TurnedOff = true
	return nil
}

func TestFilterData(t *testing.T) {

	mockedPump := &MockedPump{}

	mockedPump.SetFilters(
		analytics.AnalyticsFilters{
			APIIDs: []string{"api123"},
		},
	)

	keys := make([]interface{}, 3)
	keys[0] = analytics.AnalyticsRecord{APIID: "api111"}
	keys[1] = analytics.AnalyticsRecord{APIID: "api123"}
	keys[2] = analytics.AnalyticsRecord{APIID: "api321"}

	filteredKeys := filterData(mockedPump, keys)
	if len(keys) == len(filteredKeys) {
		t.Fatal("keys and filtered keys have the  same lenght")
	}

}

// TestTrimData check the correct functionality of max_record_size
func TestTrimData(t *testing.T) {
	mockedPump := &MockedPump{}

	loremIpsum := "Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua"

	// key = max_record_size, val = expected output
	testMatrix := map[int]int{
		0:                   len(loremIpsum), // if not set then we should not trim
		5:                   5,               // 5 should be the length of raw response and raw request
		len(loremIpsum) + 1: len(loremIpsum), // if the raw data is smaller than max_record_size, then nothing is trimmed
	}

	keys := make([]interface{}, 1)
	//test for global config max_record_size
	for maxRecordSize, expected := range testMatrix {
		SystemConfig.MaxRecordSize = maxRecordSize

		keys[0] = analytics.AnalyticsRecord{
			APIID:       "api1",
			RawResponse: loremIpsum,
			RawRequest:  loremIpsum,
		}

		filteredKeys := filterData(mockedPump, keys)
		decoded := filteredKeys[0].(analytics.AnalyticsRecord)

		assert.Equal(t, len(decoded.RawRequest), expected)
		assert.Equal(t, len(decoded.RawResponse), expected)
	}
	//test for individual pump config with max_record_size
	for maxRecordSize, expected := range testMatrix {
		mockedPump.SetMaxRecordSize(maxRecordSize)

		keys[0] = analytics.AnalyticsRecord{
			APIID:       "api1",
			RawResponse: loremIpsum,
			RawRequest:  loremIpsum,
		}

		filteredKeys := filterData(mockedPump, keys)
		decoded := filteredKeys[0].(analytics.AnalyticsRecord)

		assert.Equal(t, len(decoded.RawRequest), expected)
		assert.Equal(t, len(decoded.RawResponse), expected)
	}
}

func TestOmitDetailsFilterData(t *testing.T) {
	mockedPump := &MockedPump{}
	mockedPump.SetOmitDetailedRecording(true)

	keys := make([]interface{}, 1)
	keys[0] = analytics.AnalyticsRecord{APIID: "api111", RawResponse: "test", RawRequest: "test"}

	filteredKeys := filterData(mockedPump, keys)
	if len(filteredKeys) == 0 {
		t.Fatal("it shouldn't have filtered a key.")
	}
	record := filteredKeys[0].(analytics.AnalyticsRecord)
	if record.RawRequest != "" || record.RawResponse != "" {
		t.Fatal("raw_request  and raw_response should be empty")
	}
}

func TestWriteData(t *testing.T) {
	mockedPump := &MockedPump{}
	Pumps = []pumps.Pump{mockedPump}

	keys := make([]interface{}, 3)
	keys[0] = analytics.AnalyticsRecord{APIID: "api111"}
	keys[1] = analytics.AnalyticsRecord{APIID: "api123"}
	keys[2] = analytics.AnalyticsRecord{APIID: "api321"}

	job := instrument.NewJob("TestJob")

	writeToPumps(keys, job, time.Now(), 2)

	mockedPump = Pumps[0].(*MockedPump)

	if mockedPump.CounterRequest != 3 {
		t.Fatal("MockedPump should have 3 requests")
	}

}

func TestWriteDataWithFilters(t *testing.T) {
	mockedPump := &MockedPump{}
	mockedPump.SetFilters(
		analytics.AnalyticsFilters{
			APIIDs: []string{"api123"},
		},
	)

	Pumps = []pumps.Pump{mockedPump}

	keys := make([]interface{}, 3)
	keys[0] = analytics.AnalyticsRecord{APIID: "api111"}
	keys[1] = analytics.AnalyticsRecord{APIID: "api123"}
	keys[2] = analytics.AnalyticsRecord{APIID: "api321"}

	job := instrument.NewJob("TestJob")

	writeToPumps(keys, job, time.Now(), 2)

	mockedPump = Pumps[0].(*MockedPump)

	if mockedPump.CounterRequest != 1 {
		fmt.Println(mockedPump.CounterRequest)
		t.Fatal("MockedPump with filter should have 3 requests")
	}
}

func TestShutdown(t *testing.T) {
	mockedPump := &MockedPump{}
	mockedPump.SetFilters(
		analytics.AnalyticsFilters{
			APIIDs: []string{"api123"},
		},
	)

	Pumps = []pumps.Pump{mockedPump}

	wg := sync.WaitGroup{}
	wg.Add(1)
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		for {
			if checkShutdown(ctx, &wg) {
				return
			}
		}
	}()

	termChan := make(chan os.Signal, 1)
	signal.Notify(termChan, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	termChan <- os.Interrupt

	<-termChan
	cancel()
	wg.Wait()

	mockedPump = Pumps[0].(*MockedPump)

	if mockedPump.TurnedOff != true {
		t.Fatal("MockedPump should have turned off")
	}
}
