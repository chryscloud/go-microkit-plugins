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
