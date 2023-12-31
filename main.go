package main

import (
	"flag"
	"github.com/juzeon/epok-forwarder/api"
	"github.com/juzeon/epok-forwarder/cli"
	"github.com/juzeon/epok-forwarder/data"
	"github.com/juzeon/epok-forwarder/forwarder"
	"github.com/juzeon/epok-forwarder/util"
	"gopkg.in/yaml.v3"
	"log/slog"
	"os"
)

var configFile string
var reload bool

func main() { // TODO block based on geo
	flag.StringVar(&configFile, "c", "config.yml", "config file")
	flag.BoolVar(&reload, "r", false, "perform hot reload")
	flag.Parse()
	cli.InitConfig()
	if reload {
		cli.Reload()
	}
	configData, err := os.ReadFile(configFile)
	if err != nil {
		util.ErrExit(err)
	}
	var config data.Config
	err = yaml.Unmarshal(configData, &config)
	if err != nil {
		util.ErrExit(err)
	}
	slog.Info("Starting forwarder...")
	fwd, err := forwarder.New(config)
	if err != nil {
		util.ErrExit(err)
	}
	err = fwd.StartAsync()
	if err != nil {
		util.ErrExit(err)
	}
	slog.Info("All listeners are on.")
	err = api.StartServer(configFile, config, fwd)
	if err != nil {
		util.ErrExit(err)
	}
}
