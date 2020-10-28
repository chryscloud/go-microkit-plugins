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
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"time"

	mclog "github.com/chryscloud/go-microkit-plugins/log"
	models "github.com/chryscloud/go-microkit-plugins/models/docker"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

// Options for docker client
type Options struct {
	Log        mclog.Logger
	Host       string
	APIVersion string
	CACert     []byte
	KeyCert    []byte
	Cert       []byte
}

// Option a single option
type Option func(*Options)

// Log - recommended to be enabled at all times
func Log(log mclog.Logger) Option {
	return func(args *Options) {
		args.Log = log
	}
}

// APIVersion - the remote docker API version
func APIVersion(version string) Option {
	return func(args *Options) {
		args.APIVersion = version
	}
}

// Host - remote host
func Host(remoteHost string) Option {
	return func(args *Options) {
		args.Host = remoteHost
	}
}

// CACert - CA Client Certificate
func CACert(cacert []byte) Option {
	return func(args *Options) {
		args.CACert = cacert
	}
}

// CertKey - Client certificate key
func CertKey(key []byte) Option {
	return func(args *Options) {
		args.KeyCert = key
	}
}

// Cert - Client certificate
func Cert(cert []byte) Option {
	return func(args *Options) {
		args.Cert = cert
	}
}

// Client - digitalocean abstraction
type Client struct {
	client     *client.Client
	httpClient *http.Client
	host       string
	version    string
	log        mclog.Logger
}

func NewSocketClient(opts ...Option) Docker {
	args := &Options{}
	for _, op := range opts {
		op(args)
	}
	if args.APIVersion == "" {
		args.APIVersion = "1.40"
	}
	if args.Host == "" {
		args.Host = "tcp://127.0.0.1:2375"
	}
	cl, err := client.NewClient(args.Host, args.APIVersion, nil, nil)
	if err != nil {
		if args.Log != nil {
			args.Log.Error("failed to init docker client", err)
		}
		panic("failed to init docker client")
	}
	return &Client{
		client:  cl,
		host:    args.Host,
		version: args.APIVersion,
		log:     args.Log,
	}
}

// NewLocalClient creates a client without TLS connection (recommended only if running on localhost)
func NewLocalClient(opts ...Option) Docker {
	args := &Options{}
	for _, op := range opts {
		op(args)
	}
	if args.APIVersion == "" {
		args.APIVersion = "1.40"
	}
	if args.Host == "" {
		args.Host = "tcp://127.0.0.1:2375"
	}
	// if CACert available define httpClient transport

	tr := defaultTransport()
	httpClient := http.Client{Transport: tr}
	if &httpClient == nil {
		if args.Log != nil {
			args.Log.Error("failed to init httpClient")
		}
		panic("failed to create httpClient for docker")
	}

	cl, err := client.NewClient(args.Host, args.APIVersion, &httpClient, nil)
	if err != nil {
		if args.Log != nil {
			args.Log.Error("failed to init docker client", err)
		}
		panic("failed to init docker client")
	}
	return &Client{
		client:     cl,
		httpClient: &httpClient,
		host:       args.Host,
		version:    args.APIVersion,
		log:        args.Log,
	}
}

// NewTLSClient init digital ocean client
func NewTLSClient(opts ...Option) Docker {
	args := &Options{}
	for _, op := range opts {
		op(args)
	}

	if args.APIVersion == "" {
		args.APIVersion = "1.40"
	}
	if args.Host == "" {
		args.Host = "tcp://127.0.0.1:2376"
	}

	// if CACert available define httpClient transport
	var httpClient http.Client
	tlsConfig := &tls.Config{}
	if args.Cert != nil && args.KeyCert != nil {
		tlsCert, err := tls.X509KeyPair(args.Cert, args.KeyCert)
		if err != nil {
			panic(fmt.Sprintf("failed to create tlsCert for http client: %v\n", err.Error()))
		}
		tlsConfig.Certificates = []tls.Certificate{tlsCert}
	}
	tlsConfig.InsecureSkipVerify = true
	if args.CACert != nil {
		caPool := x509.NewCertPool()
		if !caPool.AppendCertsFromPEM(args.CACert) {
			panic("could not add RootCA pem")
		}
		tlsConfig.RootCAs = caPool
	}
	tr := defaultTransport()
	tr.TLSClientConfig = tlsConfig

	httpClient = http.Client{Transport: tr}

	if &httpClient == nil {
		if args.Log != nil {
			args.Log.Error("failed to init httpClient")
		}
		panic("failed to create httpClient for docker")
	}

	cl, err := client.NewClient(args.Host, args.APIVersion, &httpClient, nil)
	if err != nil {
		if args.Log != nil {
			args.Log.Error("failed to init docker client", err)
		}
		panic("failed to init docker client")
	}
	return &Client{
		client:     cl,
		httpClient: &httpClient,
		host:       args.Host,
		version:    args.APIVersion,
		log:        args.Log,
	}
}

// GetDockerClient - return configured docker client
func (cl *Client) GetDockerClient() *client.Client {
	return cl.client
}

// ContainersListWithOptions  list containers with various filters
func (cl *Client) ContainersListWithOptions(opts types.ContainerListOptions) ([]types.Container, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	containers, err := cl.client.ContainerList(ctx, opts)
	if err != nil {
		return nil, err
	}
	return containers, nil
}

// ContainersList for all docker containers
func (cl *Client) ContainersList() ([]types.Container, error) {
	containers, err := cl.client.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		return nil, err
	}
	return containers, nil
}

// ContainerLogs - retrurn last defined number of lines since timestamp (if larger than zero)
func (cl *Client) ContainerLogs(containerID string, tailNumberLines int, sinceTimestamp time.Time) (*models.DockerLogs, error) {
	// default is 100 lines
	if tailNumberLines == 0 {
		tailNumberLines = 100
	}
	opts := types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true, Follow: false, Tail: strconv.Itoa(tailNumberLines)}

	singleCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	reader, err := cl.client.ContainerLogs(singleCtx, containerID, opts)
	if reader != nil {
		defer reader.Close()
	} else {
		return &models.DockerLogs{
			Stderr: []byte{},
			Stdout: []byte{},
		}, nil
	}
	if err != nil {
		if cl.log != nil {
			cl.log.Error("failed to read logs from container", err)
			return nil, err
		}
	}

	// demux output and error logs
	stdoutput := new(bytes.Buffer)
	stderror := new(bytes.Buffer)
	_, err = stdcopy.StdCopy(stdoutput, stderror, reader)
	if err != nil {
		cl.log.Error("failed to demux log stream", err)
		return nil, err
	}
	logs := &models.DockerLogs{
		Stderr: stderror.Bytes(),
		Stdout: stdoutput.Bytes(),
	}
	// p := make([]byte, 8)
	// reader.Read(p)
	// content, err := ioutil.ReadAll(reader)
	// if err != nil {
	// 	cl.log.Error("failed to copy logs to string", err)
	// 	return "", err
	// }
	return logs, nil
}

// ContainerLogsStream streams logs from server until done channel received true
func (cl *Client) ContainerLogsStream(containerID string, output chan []byte, done chan bool) error {

	// this part is for streaming
	go func(containerID string, done chan bool) error {
		opts := types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true, Follow: true}
		ctx := context.Background()
		reader, err := cl.client.ContainerLogs(ctx, containerID, opts)
		defer reader.Close()
		if err != nil {
			if cl.log != nil {
				cl.log.Error("failed to read logs from container", err)
				return err
			}
		}
		// nBytes, nChunks := int64(0), int64(0)
		for {
			buf := make([]byte, 0, 1024)
			n, err := reader.Read(buf[:cap(buf)])
			buf = buf[:n]
			if err != nil {
				if err == io.EOF {
					return nil
				}
				cl.log.Error("failed to read log stream", err)
				return err
			}
			// nChunks++
			// nBytes += int64(len(buf))
			output <- buf
			select {
			case <-done:
				return nil
			default:
				break
			}
		}
	}(containerID, done)

	return nil
}

// ContainerCreate - Creates a new container
func (cl *Client) ContainerCreate(name string, config *container.Config, hostConfig *container.HostConfig, networkConfig *network.NetworkingConfig) (*container.ContainerCreateCreatedBody, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	body, err := cl.client.ContainerCreate(ctx, config, hostConfig, networkConfig, name)
	if err != nil {
		return nil, err
	}
	return &body, nil
}

// ContainerStart - start a created container
func (cl *Client) ContainerStart(containerID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return cl.client.ContainerStart(ctx, containerID, types.ContainerStartOptions{})
}

// ContainerRestart - restarting a running container
func (cl *Client) ContainerRestart(containerID string, waitForRestartLimit time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*30)
	defer cancel()
	return cl.client.ContainerRestart(ctx, containerID, &waitForRestartLimit)
}

// ContainerStop - stops the container
func (cl *Client) ContainerStop(containerID string, forceKillAfter *time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*600)
	defer cancel()
	if err := cl.client.ContainerStop(ctx, containerID, forceKillAfter); err != nil {
		return err
	}
	return nil
}

// ContainerStats - returns a snapshot of container stats in a moment it's called
func (cl *Client) ContainerStats(containerID string) (*types.StatsJSON, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*600)
	defer cancel()
	stats, err := cl.client.ContainerStats(ctx, containerID, false)
	if err != nil {
		return nil, err
	}
	defer stats.Body.Close()
	buf := new(bytes.Buffer)
	buf.ReadFrom(stats.Body)
	var jsonStats types.StatsJSON
	jsonErr := json.Unmarshal(buf.Bytes(), &jsonStats)
	if jsonErr != nil {
		return nil, err
	}

	return &jsonStats, nil
}

// ContainerGet insepcts a container
func (cl *Client) ContainerGet(containerID string) (*types.ContainerJSON, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
	defer cancel()
	j, err := cl.client.ContainerInspect(ctx, containerID)
	if err != nil {
		return nil, err
	}

	return &j, nil
}

// ContainersPrune - cleaning up non-running containers
func (cl *Client) ContainersPrune(pruneFilter filters.Args) (*types.ContainersPruneReport, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*5)
	defer cancel()
	prune, err := cl.client.ContainersPrune(ctx, pruneFilter)
	if err != nil {
		return nil, err
	}
	return &prune, nil
}

// VolumesPrune - cleaning up non-attached volumes
func (cl *Client) VolumesPrune(pruneFilter filters.Args) (*types.VolumesPruneReport, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*5)
	defer cancel()
	prune, err := cl.client.VolumesPrune(ctx, pruneFilter)
	if err != nil {
		return nil, err
	}
	return &prune, nil
}

// ImagePullDockerHub - pull private image from docker hub (it waits for pull to finish)
func (cl *Client) ImagePullDockerHub(image, tag string, username, password string) (string, error) {
	authConfig := types.AuthConfig{
		Username: username,
		Password: password,
	}
	encodedJSON, err := json.Marshal(authConfig)
	if err != nil {
		if cl.log != nil {
			cl.log.Error("failed to unmarshall auth config", err)
			return "", err
		}
	}
	authStr := base64.URLEncoding.EncodeToString(encodedJSON)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	out, err := cl.client.ImagePull(ctx, "docker.io/"+image+":"+tag, types.ImagePullOptions{RegistryAuth: authStr})
	if err != nil {
		if cl.log != nil {
			cl.log.Error("failed to pull docker image", "docker.io/ "+image+":"+tag)
			return "", err
		}
	}
	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, out)
	if err != nil && err != io.EOF {
		if cl.log != nil {
			cl.log.Error("failed to copy logs to string", err)
			return "", err
		}
	}
	logStr := buf.String()
	return logStr, nil
}

// ImageRemove - removes an image by force and prunes its children
func (cl *Client) ImageRemove(imageID string) ([]types.ImageDelete, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	deleted, err := cl.client.ImageRemove(ctx, imageID, types.ImageRemoveOptions{
		Force:         true,
		PruneChildren: true,
	})
	if err != nil {
		return nil, err
	}
	return deleted, nil
}

// ImagesList - returns list of pulled images
func (cl *Client) ImagesList() ([]types.ImageSummary, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	images, err := cl.client.ImageList(ctx, types.ImageListOptions{})
	if err != nil {
		return nil, err
	}
	return images, nil
}

// CalculateStats - converting container stats into more easy readable stats
func (cl *Client) CalculateStats(jsonStats *types.StatsJSON) *models.Stats {
	memPercent := float64(0)
	if jsonStats.MemoryStats.Limit != 0 {
		memPercent = float64(jsonStats.MemoryStats.Usage) / float64(jsonStats.MemoryStats.Limit) * 100.0
	}
	previousCPU := jsonStats.PreCPUStats.CPUUsage.TotalUsage
	previousSystem := jsonStats.PreCPUStats.SystemUsage
	cpuPercent := calculateCPUPercentUnix(previousCPU, previousSystem, jsonStats)
	blkRead, blkWrite := calculateBlockIO(jsonStats.BlkioStats)
	mem := float64(jsonStats.MemoryStats.Usage)
	memLimit := float64(jsonStats.MemoryStats.Limit)
	pidsStatsCurrent := jsonStats.PidsStats.Current

	netRx, netTx := calculateNetwork(jsonStats.Networks)

	stats := &models.Stats{
		Name:          jsonStats.Name,
		ID:            jsonStats.ID,
		CPUPercent:    cpuPercent,
		Memory:        mem,
		MemoryPercent: memPercent,
		MemoryLimit:   memLimit,
		NetworkRx:     netRx,
		NetworkTx:     netTx,
		BlockRead:     float64(blkRead),
		BlockWrite:    float64(blkWrite),
		PidsCurrent:   pidsStatsCurrent,
	}

	return stats
}

func (cl *Client) ContainerReplace(containerID string, image string, tag string) error {

	originalContainer, err := cl.ContainerGet(containerID)
	if err != nil {
		if cl.log != nil {
			cl.log.Error("failed to get container with id", containerID, err)
		}
		return err
	}

	// stopping old container
	killAfter := time.Second * 5
	stopErr := cl.ContainerStop(containerID, &killAfter)
	if stopErr != nil {
		if cl.log != nil {
			cl.log.Error("failed to stop old container", containerID, stopErr)
		}
		return stopErr
	}

	originalContainerName := originalContainer.Name
	tempContainerName := originalContainerName + "_temp"
	rErr := cl.ContainerRename(originalContainer.ID, tempContainerName)
	if rErr != nil {
		sErr := cl.ContainerStart(originalContainer.ID)
		if sErr != nil {
			if cl.log != nil {
				cl.log.Error("failed to start an old container back up after failing to rename", sErr)
			}
		}
		return rErr
	}

	originalConf := originalContainer.Config
	// replace image with the new image
	originalConf.Image = image + ":" + tag

	newlyCreatedContainer, ccErr := cl.ContainerCreate(originalContainerName, originalConf, originalContainer.HostConfig, nil)
	if ccErr != nil {
		// revert renaming back the old container
		rbErr := cl.ContainerRename(containerID, originalContainerName)
		rbErr = cl.ContainerStart(containerID)
		if rbErr != nil {
			return rbErr
		}
		if cl.log != nil {
			cl.log.Error("failed to create a new container with original name", originalContainerName, ccErr)
		}
		return ccErr
	}

	sErr := cl.ContainerStart(newlyCreatedContainer.ID)
	if sErr != nil {
		if cl.log != nil {
			cl.log.Error("failed to start newly created container", originalContainerName, newlyCreatedContainer.ID, sErr)
		}
		// undo previous changes to origial container and remove newly created container
		cerr := cl.ContainerRename(containerID, originalContainerName)
		cerr = cl.ContainerStart(containerID)
		cerr = cl.ContainerRemove(newlyCreatedContainer.ID)
		if cerr != nil {
			return cerr
		}

		return sErr
	}

	_, remErr := cl.ContainersPrune(filters.NewArgs())
	if remErr != nil {
		if cl.log != nil {
			cl.log.Error("failed to remove old container", containerID, remErr)
		}
		return remErr
	}

	return nil
}

// ContainerRemove - removing the container. timeout in 10 seconds, force removing all
func (cl *Client) ContainerRemove(containerID string) error {

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	remErr := cl.client.ContainerRemove(ctx, containerID, types.ContainerRemoveOptions{Force: true, RemoveVolumes: true, RemoveLinks: true})
	if remErr != nil {
		if cl.log != nil {
			cl.log.Error("failed to remove old container", containerID, remErr)
		}
		return remErr
	}
	return nil
}

// replaceRevert reverts the phases done by ContainerReplace function in case of errors
func (cl *Client) ContainerRename(containerID string, newContainerName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	// revert renaming back
	rbErr := cl.client.ContainerRename(ctx, containerID, newContainerName)
	if rbErr != nil {
		if cl.log != nil {
			cl.log.Error("failed to rename old container back", newContainerName, rbErr)
		}
		return rbErr
	}

	return nil
}

func calculateBlockIO(blkio types.BlkioStats) (blkRead uint64, blkWrite uint64) {
	for _, bioEntry := range blkio.IoServiceBytesRecursive {
		switch strings.ToLower(bioEntry.Op) {
		case "read":
			blkRead = blkRead + bioEntry.Value
		case "write":
			blkWrite = blkWrite + bioEntry.Value
		}
	}
	return
}

func calculateNetwork(network map[string]types.NetworkStats) (float64, float64) {
	var rx, tx float64

	for _, v := range network {
		rx += float64(v.RxBytes)
		tx += float64(v.TxBytes)
	}
	return rx, tx
}

func calculateCPUPercentUnix(previousCPU, previousSystem uint64, v *types.StatsJSON) float64 {
	var (
		cpuPercent = 0.0
		// calculate the change for the cpu usage of the container in between readings
		cpuDelta = float64(v.CPUStats.CPUUsage.TotalUsage) - float64(previousCPU)
		// calculate the change for the entire system between readings
		systemDelta = float64(v.CPUStats.SystemUsage) - float64(previousSystem)
	)

	if systemDelta > 0.0 && cpuDelta > 0.0 {
		cpuPercent = (cpuDelta / systemDelta) * float64(len(v.CPUStats.CPUUsage.PercpuUsage)) * 100.0
	}
	return cpuPercent
}

// defaultTransport returns a new http.Transport with similar default values to
// http.DefaultTransport, but with idle connections and keepalives disabled.
func defaultTransport() *http.Transport {
	transport := defaultPooledTransport()
	transport.DisableKeepAlives = true
	transport.MaxIdleConnsPerHost = -1
	return transport
}

// defaultPooledTransport returns a new http.Transport with similar default
// values to http.DefaultTransport. Do not use this for transient transports as
// it can leak file descriptors over time. Only use this for transports that
// will be re-used for the same host(s).
func defaultPooledTransport() *http.Transport {
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		MaxIdleConnsPerHost:   runtime.GOMAXPROCS(0) + 1,
	}
	return transport
}
