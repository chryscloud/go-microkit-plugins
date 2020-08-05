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

import (
	"time"

	models "github.com/chryscloud/go-microkit-plugins/models/docker"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
)

// Docker API interfaces
type Docker interface {
	ContainersList() ([]types.Container, error)
	ContainersListWithOptions(opts types.ContainerListOptions) ([]types.Container, error)
	ContainerLogs(containerID string, tailNumberLines int, sinceTimestamp time.Time) (*models.DockerLogs, error)

	// ContainerLogsStream streams logs to output channel until done is received. User is responsible to close the passed in channel
	ContainerLogsStream(containerID string, output chan []byte, done chan bool) error

	// Container CRUD operations
	ContainerCreate(name string, config *container.Config, hostConfig *container.HostConfig, networkConfig *network.NetworkingConfig) (*container.ContainerCreateCreatedBody, error)
	ContainerStart(containerID string) error
	ContainerRestart(containerID string, waitForRestartLimit time.Duration) error
	ContainersPrune(pruneFilter filters.Args) (*types.ContainersPruneReport, error)
	ContainerStop(containerID string, killAfterTimeout *time.Duration) error
	ContainerGet(containerID string) (*types.ContainerJSON, error)
	ContainerStats(containerID string) (*types.StatsJSON, error)
	ImagesList() ([]types.ImageSummary, error)
	ImagePullDockerHub(image, tag string, username, password string) (string, error)
	ImageRemove(imageID string) ([]types.ImageDelete, error)
	VolumesPrune(pruneFilter filters.Args) (*types.VolumesPruneReport, error)
	GetDockerClient() *client.Client
	CalculateStats(jsonStats *types.StatsJSON) *models.Stats
}
