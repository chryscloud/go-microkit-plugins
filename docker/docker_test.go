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
	"fmt"
	"io/ioutil"
	"strings"
	"testing"

	mclog "github.com/chryscloud/go-microkit-plugins/log"
	"github.com/docker/docker/api/types"
)

var (
	zl, _ = mclog.NewZapLogger("info")

	host        = "tcp://127.0.0.1:2376"
	apiVersion  = "1.39"
	containerID = ""

	cacert, _  = ioutil.ReadFile("/media/igor/ubuntu/Nextcloud/Documents/Cocooncam/conffiles/development/docker-keys/docker-ca.pem")
	certKey, _ = ioutil.ReadFile("/media/igor/ubuntu/Nextcloud/Documents/Cocooncam/conffiles/development/docker-keys/docker-client-key.pem")
	cert, _    = ioutil.ReadFile("/media/igor/ubuntu/Nextcloud/Documents/Cocooncam/conffiles/development/docker-keys/docker-client-cert.pem")
)

func TestContainerReplace(t *testing.T) {
	cl := NewSocketClient(Log(zl), Host("unix:///var/run/docker.sock"))
	containers, err := cl.ContainersList()
	if err != nil {
		t.Fatal(err)
	}
	for _, cont := range containers {
		img := cont.Image
		if strings.Contains(img, "chryscloud/chrysedgeproxy:0.0.2") {
			err := cl.ContainerReplace(cont.ID, "chryscloud/chrysedgeproxy", "0.0.4")
			if err != nil {
				t.Fatal(err)
			}
		}
		break
	}
}

func TestSystemWideInfo(t *testing.T) {
	cl := NewSocketClient(Log(zl), Host("unix:///var/run/docker.sock"))
	systemInfo, diskUsage, err := cl.SystemWideInfo()
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("Container running: %v\n", systemInfo.ContainersRunning)
	fmt.Printf("Container paused: %v\n", systemInfo.ContainersPaused)
	fmt.Printf("Container stopped: %v\n", systemInfo.ContainersStopped)
	fmt.Printf("Containers total: %v\n", systemInfo.Containers)

	imgNum := len(diskUsage.Images)
	totalImgSize := int64(0)
	activeImages := int64(0)
	for _, im := range diskUsage.Images {
		activeImages += im.Containers
		totalImgSize += im.SharedSize
	}
	containerTotalSize := int64(0)
	for _, c := range diskUsage.Containers {
		containerTotalSize += c.SizeRw
	}
	totalVolumeSize := int64(0)
	activeVolumes := int64(0)
	for _, v := range diskUsage.Volumes {
		activeVolumes += v.UsageData.RefCount
		totalVolumeSize += v.UsageData.Size
	}
	fmt.Printf("disk size images: %v, size: %v, active images: %v\n", imgNum, totalImgSize, activeImages)
	fmt.Printf("Containers total disk size: %v, volume size: %v, Active volumes: %v\n", containerTotalSize, totalVolumeSize, activeVolumes)

	opts := types.ContainerListOptions{
		All:   true,
		Size:  true,
		Quiet: false,
	}
	containers, err := cl.ContainersListWithOptions(opts)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range containers {
		if len(c.Names) > 0 {
			skip := false
			for _, n := range c.Names {
				if strings.Contains(n, "chrysedgeportal") || strings.Contains(n, "chrysedgeserver") || strings.Contains(n, "redis") {
					skip = true
				}
			}
			if skip {
				continue
			}
			stats, err := cl.ContainerStats(c.ID)
			if err != nil {
				t.Fatal(err)
			}
			calculated := cl.CalculateStats(stats)
			calculated.Status = c.Status
			calculated.State = c.State
			fmt.Printf("Stats: %v\n", calculated)
		}
	}
}

func TestDockerPullImage(t *testing.T) {
	cl := NewSocketClient(Log(zl), Host("unix:///var/run/docker.sock"))
	cl.ImagePullDockerHub("chryscloud/chrysedgeserver", "0.0.8-arm64v8", "", "")
}

//TODO: tests need to be modified to run without actual docker config
// func TestSocketClient(t *testing.T) {
// 	cl := NewSocketClient(Log(zl), Host("unix:///var/run/docker.sock"))
// 	containers, err := cl.ContainersList()
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	fmt.Printf("containers: %v\n", containers)
// }

// func TestLocalDocker(t *testing.T) {
// 	// to enable TCP for dockerd on localhost
// 	// brew install socat
// 	// socat TCP-LISTEN:2376,reuseaddr,fork,bind=127.0.0.1 UNIX-CLIENT:/var/run/docker.sock
// 	cl := NewLocalClient(Log(zl), Host("tcp://127.0.0.1:2376"))
// 	containers, err := cl.ContainersList()
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	fmt.Printf("containers: %v\n", containers)
// 	for _, cont := range containers {
// 		stats, err := cl.ContainerStats(cont.ID)
// 		if err != nil {
// 			t.Fatal(err)
// 		}

// 		containerStats := cl.CalculateStats(stats)
// 		fmt.Printf("stats: %v\n", containerStats)
// 	}

// }

// func TestListContainersWithOptions(t *testing.T) {
// 	cl := NewTLSClient(Log(zl),
// 		Host(host),
// 		APIVersion(apiVersion),
// 		CACert(cacert),
// 		CertKey(certKey),
// 		Cert(cert))
// 	args := filters.NewArgs()
// 	args.Add("health", "unhealthy")
// 	containers, err := cl.ContainersListWithOptions(types.ContainerListOptions{
// 		Filters: args,
// 	})
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	for _, cont := range containers {
// 		fmt.Printf("cont: %v\n", cont.Names[0])
// 	}

// }

// func TestListContainers(t *testing.T) {

// 	cl := NewTLSClient(Log(zl),
// 		Host(host),
// 		APIVersion(apiVersion),
// 		CACert(cacert),
// 		CertKey(certKey),
// 		Cert(cert))
// 	containers, err := cl.ContainersListWithOptions(types.ContainerListOptions{All: true})
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	if len(containers) == 0 {
// 		errStart := cl.ContainerStart("nginx-proxy")
// 		if errStart != nil {
// 			t.Fatal(errStart)
// 		}
// 		errStart = cl.ContainerStart("nginx-proxy-letsencrypt")
// 		if errStart != nil {
// 			t.Fatal(errStart)
// 		}
// 	} else {
// 		for _, c := range containers {
// 			fmt.Printf("c: %v %v %v\n", c.Image, c.Names, c.ID)
// 		}
// 	}
// }

// func TestPullImage(t *testing.T) {
// 	cl := NewTLSClient(Log(zl),
// 		Host(host),
// 		APIVersion(apiVersion),
// 		CACert(cacert),
// 		CertKey(certKey),
// 		Cert(cert))

// 	output, err := cl.ImagePullDockerHub("dtable/test", "latest", "tesst", "tst")
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	fmt.Printf(output)
// }

// func TestListImages(t *testing.T) {
// 	cl := NewTLSClient(Log(zl),
// 		Host(host),
// 		APIVersion(apiVersion),
// 		CACert(cacert),
// 		CertKey(certKey),
// 		Cert(cert))
// 	list, err := cl.ImagesList()
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	for _, l := range list {
// 		fmt.Printf("image: %v, %v, %v, %v\n", l.RepoDigests, l.Labels, l.Created, l.RepoTags)
// 	}
// }

// func TestContainerLogs(t *testing.T) {
// 	cl := NewTLSClient(Log(zl),
// 		Host("tcp://127.0.0.1:2376"),
// 		APIVersion(apiVersion),
// 		CACert(cacert),
// 		CertKey(certKey),
// 		Cert(cert))

// 	logs, err := cl.ContainerLogs("abc", 50, time.Unix(0, 0))
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	test := string(logs.Stdout)
// 	fmt.Printf("%s", test)
// 	fmt.Printf("logs:  %v\n", string(logs.Stdout))

// }

// func TestContainerLogsStream(t *testing.T) {
// 	output := make(chan []byte)
// 	done := make(chan bool)
// 	stoppedStreaming := make(chan bool)
// 	defer close(output)
// 	defer close(done)
// 	defer close(stoppedStreaming)

// 	cl := NewTLSClient(Log(zl),
// 		Host("tcp://127.0.0.1:2376"),
// 		APIVersion(apiVersion),
// 		CACert(cacert),
// 		CertKey(certKey),
// 		Cert(cert))

// 	err := cl.ContainerLogsStream("abc", output, done)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	// stream for 5 seconds
// 	go func() {
// 		time.Sleep(time.Second * 5)
// 		done <- true
// 		stoppedStreaming <- true
// 	}()

// LOOP:
// 	for {
// 		select {
// 		case msg, ok := <-output:
// 			if !ok {
// 				fmt.Print("not ok reached")
// 				break
// 			}
// 			fmt.Printf("%s", string(msg))
// 		case <-stoppedStreaming:
// 			fmt.Printf("streaming stopped")
// 			break LOOP
// 		}
// 	}
// }
