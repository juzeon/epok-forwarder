package main

import (
	"flag"
	"github.com/juzeon/epok-forwarder/data"
	"github.com/juzeon/epok-forwarder/forwarder"
	"github.com/juzeon/epok-forwarder/util"
	"gopkg.in/yaml.v3"
	"log/slog"
	"os"
)

func main() {
	var configFile string
	flag.StringVar(&configFile, "c", "config.yml", "config file")
	flag.Parse()
	configData, err := os.ReadFile(configFile)
	if err != nil {
		util.ErrExit(err)
	}
	var config data.Config
	err = yaml.Unmarshal(configData, &config)
	if err != nil {
		util.ErrExit(err)
	}
	slog.Info("Validating config...")
	err = config.Validate()
	if err != nil {
		util.ErrExit(err)
	}
	slog.Info("Starting forwarder...")
	err = forwarder.New(config)
	if err != nil {
		util.ErrExit(err)
	}
	select {}
}
