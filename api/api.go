package api

import (
	"fmt"
	"github.com/juzeon/epok-forwarder/data"
	"github.com/juzeon/epok-forwarder/forwarder"
	"github.com/juzeon/epok-forwarder/util"
	"log/slog"
	"net/http"
	"strings"
)

func StartServer(configFile string, config data.Config, forwarderIns *forwarder.Forwarder) error {
	mux := http.NewServeMux()
	mux.Handle("/api/reload", http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "text/plain")
		if request.Method != http.MethodPost {
			writer.WriteHeader(400)
			writer.Write([]byte("only POST requests are accepted"))
			return
		}
		if config.Secret != "" {
			auth := request.Header.Get("Authorization")
			auth = strings.TrimPrefix(auth, "Bearer ")
			if auth != config.Secret {
				writer.WriteHeader(403)
				writer.Write([]byte("unauthorized"))
				return
			}
		}
		newConfig, err := data.ReadConfig(configFile)
		if err != nil {
			writer.WriteHeader(500)
			writer.Write([]byte(err.Error()))
			return
		}
		slog.Info("Stopping previous forwarder instance...")
		if err := forwarderIns.Stop(); err != nil {
			writer.WriteHeader(500)
			writer.Write([]byte(err.Error()))
			return
		}
		slog.Info("Starting new forwarder instance...")
		if fwd, err := startForwarderWithConfig(newConfig); err != nil {
			// revert
			slog.Error("Could not start new forwarder, reverting...", "error", err)
			fwd0, err0 := startForwarderWithConfig(config)
			if err0 != nil {
				util.ErrExit(fmt.Errorf("failed to revert: %w", err0))
			}
			slog.Info("Reverted to previous forwarder")
			forwarderIns = fwd0
			writer.WriteHeader(500)
			writer.Write([]byte(err.Error()))
			return
		} else {
			slog.Info("Started new forwarder")
			forwarderIns = fwd
			config = newConfig
			writer.WriteHeader(200)
			writer.Write([]byte("ok"))
			return
		}
	}))
	slog.Info("Start API server on: " + config.API)
	return http.ListenAndServe(config.API, mux)
}
func startForwarderWithConfig(config data.Config) (*forwarder.Forwarder, error) {
	fwd, err := forwarder.New(config)
	if err != nil {
		return nil, err
	}
	err = fwd.StartAsync()
	if err != nil {
		return nil, err
	}
	return fwd, nil
}
