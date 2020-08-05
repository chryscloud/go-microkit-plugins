// Copyright 2020 Wearless Tech Inc All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package backpressure

import (
	"fmt"
	"math/rand"
	"sync/atomic"
	"testing"
	"time"

	mclog "github.com/chryscloud/go-microkit-plugins/log"
)

type event struct {
	name string
}

var numberOfProcessed uint64

type batchWorker struct {
}

var (
	incomingDelay   = time.Duration(rand.Intn(1)) * time.Millisecond
	processingDelay = time.Duration(rand.Intn(1000)+1) * time.Millisecond
)

func (bw *batchWorker) PutMulti(events []interface{}) error {
	for i := 0; i < len(events); i++ {
		// event := events[i].(event)
		// if i == len(events)-1 {
		// fmt.Printf("last processed event: %v\n", event.name)
		// }
		atomic.AddUint64(&numberOfProcessed, 1)
	}
	time.Sleep(processingDelay) // processing delay per PutMulti method
	// fmt.Printf("Processed: %v events\n", len(events))
	return nil
}

func TestBackpressure(t *testing.T) {

	bw := &batchWorker{}
	zl, err := mclog.NewZapLogger("info")
	if err != nil {
		t.Fatal(err)
	}

	bckPress, err := NewBackpressureContext(bw, BatchMaxSize(300), BatchTimeMs(100), Workers(100), Log(zl))

	if err != nil {
		t.Fatal(err)
	}
	defer bckPress.Close()

	numEventsToSend := 15000 // 30K events

	numEventsSent := 0

	now := time.Now()

	for i := 0; i < numEventsToSend; i++ {
		e := event{
			name: fmt.Sprintf("event_i_%d", i+1),
		}
		numEventsSent++
		e1 := event{
			name: fmt.Sprintf("event_j_%d", i+1),
		}
		numEventsSent++

		time.Sleep(incomingDelay)

		err := bckPress.Add(e)
		err = bckPress.Add(e1)
		if err != nil {
			t.Fatal(err)
		}
	}

	// wait to complete the test
	delay := time.Duration(bckPress.batchTimeMs*2) * time.Millisecond
	time.Sleep(delay)

	if uint64(numEventsSent) != numberOfProcessed {
		t.Fatalf("expected to ingest %d events but only %d ingested", numEventsToSend, numberOfProcessed)
	}

	fmt.Printf("Processed all in %v\n", time.Since(now))

}
