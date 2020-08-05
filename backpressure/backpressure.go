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
	"errors"
	"fmt"
	"math"
	"time"

	mclog "github.com/chryscloud/go-microkit-plugins/log"
)

var (
	// ErrBackPressureInit in case initialization fails
	ErrBackPressureInit = errors.New("backpressure run context failed to initialize")
)

const (
	monitorWarningStart float64 = 0.8 // 80% upper limit before monitor starts warning
)

// Options - settings options insted of defaults
type Options struct {
	BatchTimeMs       float64
	BatchMaxSize      int
	MaxWorkers        int
	MaxBatchesInQueue int
	Log               mclog.Logger
}

// Option a single option
type Option func(*Options)

// BatchTimeMs how long a wait before sending list of tasks to our worker
func BatchTimeMs(batchTimeMs float64) Option {
	return func(args *Options) {
		args.BatchTimeMs = batchTimeMs
	}
}

// BatchMaxSize maximum number of
func BatchMaxSize(batchMaxSize int) Option {
	return func(args *Options) {
		args.BatchMaxSize = batchMaxSize
	}
}

// Workers - number of workers processing jobs
func Workers(maxWorkers int) Option {
	return func(args *Options) {
		args.MaxWorkers = maxWorkers
	}
}

// MaxBatchesInQueue - a queue with batches. Defines maximum number of batches help in the batch queue
func MaxBatchesInQueue(maxBatchesInQueue int) Option {
	return func(args *Options) {
		args.MaxBatchesInQueue = maxBatchesInQueue
	}
}

// Log - if log present it means debug enabled
func Log(log mclog.Logger) Option {
	return func(args *Options) {
		args.Log = log
	}
}

// PressureContext which combines all the channels
type PressureContext struct {
	inputChan          chan interface{}
	batchChan          chan []interface{}
	doneChan           chan bool
	batchTimeMs        float64 // waiting for 1 second to collect before processing
	batchMaxSize       int     // maximum number of events in batch
	maxBatchesInQueue  int     // maximum number of batches that can wait to be processed
	maxWorkers         int     // maximum number of worker routines
	workerCount        uint64  // current worker count
	log                mclog.Logger
	backpressureMethod Backpressure
}

// NewBackpressureContext creates a backpressure run context and kicks off 2 go routinges (consumer and collector)
func NewBackpressureContext(backpressurePutMulti Backpressure, opts ...Option) (*PressureContext, error) {
	args := &Options{
		MaxWorkers:        100,
		MaxBatchesInQueue: 100,
		BatchTimeMs:       100,
		BatchMaxSize:      50,
		Log:               nil,
	}
	for _, op := range opts {
		op(args)
	}

	runCtx := &PressureContext{
		inputChan:          make(chan interface{}),
		batchChan:          make(chan []interface{}, args.MaxBatchesInQueue),
		doneChan:           make(chan bool),
		batchTimeMs:        float64(args.BatchTimeMs),
		batchMaxSize:       args.BatchMaxSize,
		maxBatchesInQueue:  args.MaxBatchesInQueue,
		log:                args.Log,
		backpressureMethod: backpressurePutMulti,
		maxWorkers:         args.MaxWorkers,
	}
	if runCtx.log != nil {
		runCtx.log.Info("Running context with ", args.MaxWorkers, "workers, ", args.BatchTimeMs, "ms batch time, ", args.BatchMaxSize, " max batch size", args.MaxBatchesInQueue, " max batches in queue")
	}
	go runCtx.collectBatch()

	for i := 0; i < args.MaxWorkers; i++ {
		go runCtx.consumeBatch()
	}

	return runCtx, nil
}

// Add event to be handled by backpressure mechanism
func (rc *PressureContext) Add(value interface{}) error {
	if rc != nil {
		rc.inputChan <- value
	} else {
		if rc.log != nil {
			rc.log.Error(ErrBackPressureInit)
		}
		return ErrBackPressureInit
	}
	return nil
}

func (rc *PressureContext) collectBatch() {
	eventbatch := make([]interface{}, 0)

	ticker := time.Tick(time.Duration(rc.batchTimeMs) * time.Millisecond)

	for {
		// if max size reached before ticker ticks
		if len(eventbatch) >= rc.batchMaxSize {

			rc.batchChan <- eventbatch
			eventbatch = make([]interface{}, 0)
		}

		select {
		case ev, ok := <-rc.inputChan:
			if !ok {
				// dispatch last batch
				if len(eventbatch) > 0 {
					if rc.log != nil {
						rc.log.Info("dispatching last batch before shutdown")
					}
					rc.batchChan <- eventbatch
				}
				return // exit consumer
			}
			eventbatch = append(eventbatch, ev)

		case <-ticker:
			if len(eventbatch) > 0 {
				rc.batchChan <- eventbatch
				// reset event batch
				eventbatch = make([]interface{}, 0)
			}
		case <-rc.doneChan:
			return
		}
	}
}

func (rc *PressureContext) consumeBatch() {
	for {

		select {
		case eb, ok := <-rc.batchChan:
			if !ok {
				if rc.log != nil {
					rc.log.Info("batch writer complete", ok)
				}
				return
			}

			eventQueueLength := len(rc.inputChan)
			batchQueueLength := len(rc.batchChan)

			if eventQueueLength > int(math.Round(monitorWarningStart*float64(rc.batchMaxSize))) ||
				batchQueueLength > int(math.Round(monitorWarningStart*float64(rc.maxBatchesInQueue))) {

				if rc.log != nil {
					rc.log.Warn("WARNING:", "Batch queues almost full", "event queue size: ", eventQueueLength, "batch queue size: ", batchQueueLength)
				}
			} else {
				if rc.log != nil {
					rc.log.Info("Current event channel size", eventQueueLength, "Current batch queue size", batchQueueLength)
				}
			}

			if rc.log != nil {
				rc.log.Info(fmt.Sprintf("batch of size %v delivered to processing (PutMulti) %v\n", len(eb), time.Now()))
			}

			err := rc.backpressureMethod.PutMulti(eb)
			if err != nil {
				if rc.log != nil {
					rc.log.Error("failed to consumer events", err)
				}
			}

		case <-rc.doneChan:
			if rc.log != nil {
				rc.log.Info("Shutting down backpressure")
			}
			return
		}
	}
}

// Close channels
func (rc *PressureContext) Close() {
	if rc != nil {
		close(rc.doneChan)
	}
}
