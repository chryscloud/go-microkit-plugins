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

package log

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"testing"

	"go.uber.org/zap"
)

// MemorySink implements zap.Sink by writing all messages to a buffer.
type MemorySink struct {
	*bytes.Buffer
}

// Implement Close and Sync as no-ops to satisfy the interface. The Write
// method is provided by the embedded buffer.

func (s *MemorySink) Close() error { return nil }
func (s *MemorySink) Sync() error  { return nil }

func TestZapLogging(t *testing.T) {
	// logger config
	zl, err := NewZapLogger("info")
	if err != nil {
		t.Fatal(err)
	}
	sink := &MemorySink{new(bytes.Buffer)}
	zap.RegisterSink("memory", func(*url.URL) (zap.Sink, error) {
		return sink, nil
	})
	zl.zapConfig.OutputPaths = []string{"memory://"}
	zapLogger, err := zl.zapConfig.Build()
	if err != nil {
		t.Fatal(err)
	}
	zapLogger.Sugar().Info("k1", "v1", "k2", "v2")

	output := sink.String()
	fmt.Printf("sink contents: %v\n", output)
	// check log kvs
	logMap := make(map[string]string)
	err = json.Unmarshal(sink.Bytes(), &logMap)
	if err != nil {
		t.Fatal(err)
	}
	if logMap["msg"] != "k1v1k2v2" {
		t.Fatalf("expected values: k1v1k2v2 but got: %s", logMap["msg"])
	}
}
