# go-microkit-plugins

[![](https://godoc.org/github.com/chryscloud/go-microkit-plugins?status.svg)](https://godoc.org/github.com/chryscloud/go-microkit-plugins)

## Introduction

Microkit Plugins is a set of backend GO libraries unifying development across multiple distributed microservices. It contains set of best practices and simple reusable modules most commonly used with RESTful microservices.

Built on top of existing open source projects:
- [Gin Web Framework](https://github.com/gin-gonic/gin)
- [Zap Logging](https://github.com/uber-go/zap)
- [JWT Go](https://github.com/dgrijalva/jwt-go)
- [Go Swaggo](https://github.com/swaggo/swag) OpenAPI2.0 specification

## Features

The framework is built with `modularity`, `reusability`, `extensibility` and `deployability` in mind.

- **Authentication** - Configurable JWT and Token Middlewares
- **Configuration** - Simple and extendable `yaml` server configuration
- **Logging** - Zap logging plugin with custom output JSON format suitable for fluentd logging (e.g. Entry_L2, stack output on Google GKE clusters)
- **Crypto** - Best practices for encrypting and validating user passwords, HMAC256 for external physical device authorization
- **Server** - Simple RESTful server with graceful start and stop on top of `Gin Web Framework`
- **Backpressure** - Optimizes event data warehousing. Collects individual events into `listed` chunks to be stored in custom data warehouse through a simple `PutMulti` interface.

### Integration plugins

- **Docker** - Most common interfaces to Docker containers (local sockets, over TCP and self signed certificates)

## Contents

* [Quick Start](#quick-start)
* [Authenticaton](#authentication)
	* [JWT Authentication](#jwt-authentication)
	* [Token Identity](#token-identity)
* [Configuration](#configuration)
	* [Extending Configuration](extending-configuration)
* [Logging](#logging)
* [Crypto](#crypto)
	* [Passwords](#passwords)
	* [Secret Key Generator](#secret-key-generator)
	* [HMAC-X](#hmac-x)
* [Backpressure](#backpressure)
* [Docker](#docker)
* [Contributing](#contributing)
* [Versioning](#versioning)
* [Authors](#authors)
* [License](#license)
* [Acknowledgments](#acknowledgments)

## Quick Start

Create a configuration file in the root folder of your project

```yaml
version: 1.0
port: 8080
title: Go Micro kit service framework
description: Go Micro kit service framework
swagger: false
mode: debug # "debug": or "release"
auth_token:
  enabled: false
  header: "authkey"
  token: "abc"

jwt_token:
  enabled: false
  secret_key: "abcedf"
  cookie_name: "mycookie"
```

Create main.go for your microservice:

```go
package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"

	cfg "github.com/chryscloud/go-microkit-plugins/config"
	"github.com/chryscloud/go-microkit-plugins/endpoints"
	mclog "github.com/chryscloud/go-microkit-plugins/log"
	msrv "github.com/chryscloud/go-microkit-plugins/server"
)

// Log global wide logging
var Log mclog.Logger

// Conf global config
var Conf Config

// useful in case we extend the configuration
type Config struct {
	cfg.YamlConfig `yaml:",inline"`
}

// init logging
func init() {
	l, err := mclog.NewEntry2ZapLogger("myservice")
	if err != nil {
		panic("failed to initialize logging")
	}
	Log = l
}

func main() {
	var (
		configFile string
	)
	// configuration file optional path. Default:  current dir with  filename conf.yaml
	flag.StringVar(&configFile, "c", "conf.yaml", "Configuration file path.")
	flag.StringVar(&configFile, "config", "conf.yaml", "Configuration file path.")
	flag.Usage = usage
	flag.Parse()

    // init configuration from conf.yaml
	err := cfg.NewYamlConfig(configFile, &Conf)
	if err != nil {
		Log.Error(err, "conf.yaml failed to load")
		panic("Failed to load conf.yaml")
	}

	// server wait to shutdown monitoring channels
	done := make(chan bool, 1)
	quit := make(chan os.Signal, 1)

	signal.Notify(quit, os.Interrupt)

	// init routing (for endpoints)
	router := msrv.NewAPIRouter(&Conf.YamlConfig)

	root := router.Group("/")
	{
		root.GET("/", endpoints.PingEndpoint)
	}

	// start server
	srv := msrv.Start(&Conf.YamlConfig, router, Log)
	// wait for server shutdown
	go msrv.Shutdown(srv, Log, quit, done)

	Log.Info("Server is ready to handle requests at", Conf.Port)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		Log.Error("Could not listen on %s: %v\n", Conf.Port, err)
	}

	<-done
}

// usage will print out the flag options for the server.
func usage() {
	usageStr := `Usage: operator [options]
	Server Options:
	-c, --config <file>              Configuration file path
`
	fmt.Printf("%s\n", usageStr)
	os.Exit(0)
}

```

Run:
```bash
go run main.go
```

Visit: `http://localhost:8080`

Expected output:
```json

{"message":"pong at 20200804203903"}
```

## Authentication

### JWT Authentication

Enable JWT token authentication in `conf.yaml` and define your secret key:

```yaml
jwt_token:
  enabled: true
  secret_key: "abcedf"
```

Optionally define `cookie_name: "mycookie"` which JWT middleware checks if no `Authorization` header found. Make sure, if storing JWT tokens as cookies on the client side, to use `httpOnly` to avoid XSS.

Usage:

```go
import (
	microKitAuth "github.com/chryscloud/go-microkit-plugins/auth"
)
// init JWT Authentication for private endpoints
keys := func(token *jwt.Token) (interface{}, error) {
	return []byte(Conf.JWTToken.SecretKey), nil
}
auhtMiddleware := microKitAuth.JwtMiddleware(&Conf.YamlConfig, &models.UserClaim{}, jwt.SigningMethodHS256, keys)

root := router.Group("/", auhtMiddleware)
{
	root.GET("/", endpoints.PingEndpoint)
}
```

Generating JWT Tokens on authentication request:
```go
userClaim := models.UserClaim{
	ID:    "userID",
	Roles:   []string{"role1","role2"},
	Enabled: true,
}
token, err := microKitAuth.NewJWTToken([]byte(Conf.JWTToken.SecretKey), jwt.SigningMethodHS256, userClaim)
```

Do not store passwords or any sensitive information into the `userClaim`!

### Token Identity

Token identity is named `identity` since it's not a complete solution for authorization. Nonetheless, it's included in this plugins package as it may be handy for `read-only` operations in some cases.


Enable token identity in the `conf.yaml`:
```yaml
auth_token:
  enabled: true
  header: "mycustomauthkey"
  token: "secret_token"
  path: "/*"
```

Usage: 
```go
import (
	microKitAuth "github.com/chryscloud/go-microkit-plugins/auth"
)

tokenMiddleware := microKitAuth.TokenMiddleware(&Cong.YamlConfig)

root := router.Group("/", tokenMiddleware)
{
	root.GET("/", endpoints.PingEndpoint)
}
```

## Configuration

Example configuration:

```yaml
version: 1.0
port: 8080
title: Go Micro kit service framework
description: Go Micro kit service framework
swagger: true
mode: debug # "debug": or "release"
auth_token:
  enabled: false
  header: "authkey"
  token: "abc"

jwt_token:
  enabled: false
  secret_key: "abcedf"
  cookie_name: "mycookie"

# extended custom config
test_endpoint: "this is test"
```

Loading default configuration:
```go
var conf cfg.YamlConfig
err := cfg.NewYamlConfig("/path/to/conf.yaml", &conf)
```

### Extending configuration

To extend default configuration define your own structure:
```go
var conf Config

type Config struct {
	cfg.YamlConfig 	`yaml:",inline"`
	TestEndpoint 	`yaml:"test_endpoint"`
}

// in your main.go file load extended configuration
 
err := cfg.NewYamlConfig("/path/to/conf.yaml", &Conf)
if err != nil {
	Log.Error(err, "conf.yaml failed to load")
	panic("Failed to load conf.yaml")
}
```

## Logging

Logging implements tagged style logging with [Ubers zap logger](https://github.com/uber-go/zap)

The output of the logger is in JSON format, adapted to fluentd logging requirements (stackdriver logging on Google or customizable to work on AWS EKS)


Tagged style logging methods
```go
func (z *ZapLogger) Error(keyvals ...interface{})
func (z *ZapLogger) Warn(keyvals ...interface{})
func (z *ZapLogger) Info(keyvals ...interface{})
```

Error function extracts golang style stacktrace.

Example to init logging into `stdout`:
```go
import (
	mclog "github.com/chryscloud/go-microkit-plugins/log"
)

...

log, err := mclog.NewZapLogger("info")
```

Compatible example with Google GKE logging:
```go
import (
	mclog "github.com/chryscloud/go-microkit-plugins/log"
)

...

log, err := mclog.NewEntry2ZapLogger("nameofmyservice")
```

## Crypto

Crypto plugin has prepared functions for generating and validating strong cryptographic passwords. 

### Passwwords

Password generator uses `bcrypt` algorithm with `cost` of `14` currently. The cost factor defines the time it takes to generate password. Higher the cost, longer the time needed to generate it. 

Usage:
```go
import (
	c "github.com/chryscloud/go-microkit-plugins/crypto"
)
...
pass, err := c.HashPassword("this is my password")
ok := c.CheckPasswordHash("this is my password", pass)
```

### Secret Key Generator

Simple method for generating random secret key from Gos Crypto package. Input parameter indicates the length of the secret in bytes. Method returns hex encoded string.

```go
import (
	c "github.com/chryscloud/go-microkit-plugins/crypto"
)

secret := c.GenerateSecretKey(16)
```

### HMAC-X

Hash-based message authentication for verifying message integrity and authenticity (e.g. for external devices), with custom Hash algorithm.

```go
import (
	c "github.com/chryscloud/go-microkit-plugins/crypto"
)
...
payload := "this is payload"
mac := ComputeHmac(sha256.New, payload, "mysecret")

isValid := c.ValidateHmacSignature(sha256.New, payload, "mysecret", mac)
```

`sha256.New` can be exhanged for example with `sha512.New`. 

Example generating HMAC-256 (sha256) using curl:
```bash
$secret="mysecret"
nonce=$(date +%s)
apisig=`echo -n "$nonce$key" | openssl dgst -sha256 -hmac $secret -binary | xxd -p -c 256`

curl -s --connect-timeout 15  "https://example.com/api/v1/myapi?timestamp=$nonce&signature=$apisig
```

## Backpressure

Handling of backpressure in case of event spikes when storing to Data Warehouse (such as BigQuery, Redshift,...) by batching them together. 

- Producer side pressure handling
- **Parallel processing** - custom number of workers
- **Configurable** - custom batch buffer and behaviour controls
- **Dropping** - dropping event on buffer full

Implementing `PutMulti` method:
```go
type batchWorker struct {
}
func (bw *batchWorker) PutMulti(events []interface{}) error {
	// custom implementation (e.g. storing to database in batches, email sending, file writing,...)
	return nil
}
```

Use:
```go
type event struct {
	name string
}

bw := &batchWorker{}

bckPress, err := backpressure.NewBackpressureContext(bw, BatchMaxSize(300), BatchTimeMs(100), Workers(100))
defer bckPress.Close()

e := event{
	name: "example event",
}
err := bckPress.Add(e)
```

- **BatchMaxSize** is the maximum number of items in a single batch
- **BatchTimeMs** is the time to wait to collect the items in a single batch
- **Workers** is number of workers (go routines)
- **Log** is logging (compatible only with /log/log.go interface)

Backpressure is mainly intended for high load batching and streaming to BigData such as BigQuery. It can also be used for loads that come occasionally in bursts, such as email (e.g. Mailgun supports batch sending) or any other scenario that involves batch processing or a large amount of small tasks.

A Worker is a single blocking (synchronous) worker. It enqueues items and processes them in a blocking manner.

With defining e.g. `Workers(N>1)` it employs multiple background workers.

Your custom PutMulti implementation is activated according to item amount `BatchMaxSize(N)` and time threshold for each background worker `BatchTimeMs(MS)`.

## Docker

Convenient methods programmatic manipulation of Docker containers.

Prepared 3 types of Docker access:
- Local Socket Connection
- Local TCP IP Connection
- Remote TCP IP Connection using TLS self-signed certificates

### TLS Client with self-signed certificates

```go
cert, _ := ioutil.ReadFile("/pathto/certificate/file")
caCert, _ := ioutil.ReadFile("/pathto/certificateCA/file")
certKey, _ := ioutil.ReadFile("/pathto/certificateKey/file")
cl := NewTLSClient(Host("tcp://xxx.xxx.xxxx.xxxx:2376"), APIVersion("1.40"), CACert(cacert), CertKey(certKey), Cert(cert))
```

Bash script for creating client/server self-signed certificates:
```bash
!/bin/bash
set -ex
mkdir certs && cd certs
echo "Creating server keys..."
echo 01 > ca.srl
openssl genrsa -des3 -out ca-key.pem
openssl req -new -x509 -days 3650 -key ca-key.pem -out ca.pem
openssl genrsa -des3 -out server-key.pem
openssl req -new -key server-key.pem -out server.csr
openssl x509 -req -days 365 -in server.csr -CA ca.pem -CAkey ca-key.pem \
    -out server-cert.pem

echo "Creating client keys..."
openssl genrsa -des3 -out client-key.pem
openssl req -new -key client-key.pem -out client.csr
echo extendedKeyUsage = clientAuth > extfile.cnf
openssl x509 -req -days 365 -in client.csr -CA ca.pem -CAkey ca-key.pem \
    -out client-cert.pem -extfile extfile.cnf

echo "Stripping passwords from keys..."
openssl rsa -in server-key.pem -out server-key.pem
openssl rsa -in client-key.pem -out client-key.pem
```

### Enable remote access to docker

Create `daemon.json` file in folder: `/etc/docker/daemon.json`

```json
{
    "hosts": ["fd://", "tcp://0.0.0.0:2376"],
    "log-driver": "json-file",
    "log-opts": {"max-size": "10m", "max-file": "3"},
    "tlscacert": "/root/.docker/ca.pem",
    "tlscert": "/root/.docker/server-cert.pem",
    "tlskey": "/root/.docker/server-key.pem",
    "tlsverify": true
}
```

Modify `docker.service` config. Open `/lib/systemd/system/docker.service` and comment out ExecStart and replace:

```
#ExecStart=/usr/bin/dockerd -H fd:// --containerd=/run/containerd/containerd.sock
ExecStart=
ExecStart=/usr/bin/dockerd
```

Reload daemon:
```
systemctl daemon-reload
```

Restart docker service:
```
systemctl restart docker.service
```

## Enable access to local Docker Socket

Create `daemon.json` file in `/etc/docker` folder and add in:
```json
{
  "hosts": [
    "fd://",
    "unix:///var/run/docker.sock"
  ]
}
```

Create a new file `/etc/systemd/system/docker.service.d/docker.conf` with the following contents:
```
[Service]
ExecStart=
ExecStart=/usr/bin/dockerd
```

Reload daemon:
```
sudo systemctl daemon-reload
```

Restart docker:
```
sudo service docker restart
```

Connect to docker socket:
```go
import (
	"github.com/chryscloud/go-microkit-plugins/docker"
)
cl := docker.NewSocketClient(docker.Log(g.Log), docker.Host("unix:///var/run/docker.sock"))
```
Available docker methods:
```go
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
```

# Contributing

Please read `CONTRIBUTING.md` for details on our code of conduct, and the process of submitting pull requests to us. 

# Versioning



# Authors

- **Igor Rendulic** - Initial work - [Chrysalis Cloud](https://chryscloud.com)

# License

This project is licensed under Apache 2.0 License - see the `LICENSE` for details.

# Acknowledgments





