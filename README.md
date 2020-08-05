# go-microkit-plugins

## Introduction

Microkit Plugins is a set of backend GO libraries unifying development across multiple distributed microservices. It contains set of best practices and simple reusable modules most commonly used with RESTful microservices.

Built on top of existing open source projects:
- [Gin Web Framework](https://github.com/gin-gonic/gin)
- [Zap Logging](https://github.com/uber-go/zap)
- [JWT Go](https://github.com/dgrijalva/jwt-go)
- [Go Swaggo](https://github.com/swaggo/swag)

## Features

The framework is built with `modularity`, `reusability`, `extensibility` and `deployability` in mind.

- **Authentication** - Configurable JWT and Token Middlewares
- **Configuration** - Simple and extendable `yaml` server configuration
- **Logging** - Zap logging plugin with custom output JSON format suitable for fluentd logging (e.g. Entry_L2, stack output on Google GKE clusters)
- **Crypto** - Best practices for encrypting and validating user passwords, HMAC256 for external physical device authorization
- **Server** - Simple RESTful server with graceful start and stop on top of `Gin Web Framework`
- **Backpressure** - Optimizes event data warehousing. Collects individual events into `listed` chunks to be stored in custom data warehouse through a simple `PutMulti` interface.

### Non-server plugins

- **Docker** - Most common interfaces to Docker containers (local sockets, over TCP and self signed certificates)

## Getting Started

* [Quick Start](#quick-start)

## Quick Start

Create a configuration file in the root folder of your project

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
```

Create main.go for your microservice:

```go
import (
    "net"
	"net/http"
	"os"
    "os/signal"
    
    msrv "github.com/myproject/go-microkit-plugins/server"
    cfg "github.com/chryscloud/go-microkit-plugins/config"
    mclog "github.com/chryscloud/go-microkit-plugins/log"
)

// Log global wide logging
var Log mclog.Logger
// Conf global config
var Conf Config

type Config struct {
	cfg.YamlConfig `yaml:",inline"`
}

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

	// var conf g.Config
	err := cfg.NewYamlConfig(configFile, &Conf)
	if err != nil {
		g.Log.Error(err, "conf.yaml failed to load")
		panic("Failed to load conf.yaml")
    }
    
    // server wait to shutdown monitoring channels
    done := make(chan bool, 1)
    quit := make(chan os.Signal, 1)

    signal.Notify(quit, os.Interrupt)

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
```

## Logging plugin

Logging implements tagged style logging with [Ubers zap logger](https://github.com/uber-go/zap)

The output of the logger is in JSON format, adapted to fluentd logging requirements. 

```
/logging
```

Tagged style logging methods
```go
func (z *ZapLogger) Error(keyvals ...interface{})
func (z *ZapLogger) Warn(keyvals ...interface{})
func (z *ZapLogger) Info(keyvals ...interface{})
```

Error function extracts golang style stacktrace.

Zap logging plugin with custom output JSON format suitable for fluentd logging