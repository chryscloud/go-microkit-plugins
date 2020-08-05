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

package docker

// Stats holding docker stats from a container
type Stats struct {
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryPercent float64 `json:"memory_percent"`
	MemoryLimit   float64 `json:"memory_limit"`
	Memory        float64 `json:"memory"`
	Name          string  `json:"name"`
	ID            string  `json:"id"`
	NetworkRx     float64 `json:"network_rx"`
	NetworkTx     float64 `json:"network_tx"`
	BlockRead     float64 `json:"block_read"`
	BlockWrite    float64 `json:"block_write"`
	PidsCurrent   uint64  `json:"pids_current"`
	Status        string  `json:"status"`
	State         string  `json:"state"`
}

type DockerLogs struct {
	Stdout []byte `json:"stdout"`
	Stderr []byte `json:"stderr"`
}
