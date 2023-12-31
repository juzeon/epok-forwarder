package main

import (
	"flag"
	"github.com/juzeon/epok-forwarder/api"
	"github.com/juzeon/epok-forwarder/cli"
	"github.com/juzeon/epok-forwarder/data"
	"github.com/juzeon/epok-forwarder/forwarder"
	"github.com/juzeon/epok-forwarder/util"
	"log/slog"
	"os"
)

type Flags struct {
	configFile string
	reload     bool
	help       bool
	generate   bool
}

func main() { // TODO block based on geo
	var flg Flags
	flag.StringVar(&flg.configFile, "c", "config.yml", "specify a config file")
	flag.BoolVar(&flg.reload, "r", false, "perform hot reload")
	flag.BoolVar(&flg.help, "h", false, "get help")
	flag.BoolVar(&flg.generate, "g", false, "generate cli env based on the config file")
	flag.Parse()
	cli.InitConfig()
	if flg.reload {
		cli.Reload()
	}
	if flg.help {
		flag.PrintDefaults()
		os.Exit(0)
	}
	config, err := data.ReadConfig(flg.configFile)
	if err != nil {
		util.ErrExit(err)
	}
	if flg.generate {
		cli.Generate(config.BaseConfig)
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
	err = api.StartServer(flg.configFile, config, fwd)
	if err != nil {
		util.ErrExit(err)
	}
}
