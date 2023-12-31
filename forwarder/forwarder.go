package forwarder

import (
	"context"
	"errors"
	"github.com/juzeon/epok-forwarder/data"
	"log/slog"
)

type Forwarder struct {
	config         data.Config
	hostForwarders []*HostForwarder
	ctx            context.Context
	cancelFunc     context.CancelFunc
	webForwarder   *WebForwarder
}

func New(config data.Config) (*Forwarder, error) {
	slog.Info("Validating config...")
	err := config.Validate()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	webForwarder, err := NewWebForwarder(ctx, config.BaseConfig)
	if err != nil {
		cancel()
		return nil, err
	}
	return &Forwarder{
		config:         config,
		hostForwarders: nil,
		ctx:            ctx,
		cancelFunc:     cancel,
		webForwarder:   webForwarder,
	}, nil
}
func (o *Forwarder) Stop() error {
	select {
	case <-o.ctx.Done():
		return errors.New("this instance has already stopped")
	default:
		slog.Info("Shutting down all listeners...")
		o.cancelFunc()
	}
	return nil
}
func (o *Forwarder) StartAsync() error {
	for _, host := range o.config.Hosts {
		hf, err := NewHostForwarder(o.ctx, o.config.BaseConfig, host, o.webForwarder)
		if err != nil {
			o.cancelFunc()
			return err
		}
		o.hostForwarders = append(o.hostForwarders, hf)
	}
	err := o.webForwarder.StartAsync()
	if err != nil {
		o.cancelFunc()
		return err
	}
	return nil
}
