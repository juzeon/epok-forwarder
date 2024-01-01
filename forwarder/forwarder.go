package forwarder

import (
	"context"
	"errors"
	"github.com/juzeon/epok-forwarder/data"
	"log/slog"
	"sync"
)

type Forwarder struct {
	config         data.Config
	hostForwarders []*HostForwarder
	ctx            context.Context
	cancelFunc     context.CancelFunc
	webForwarder   *WebForwarder
	waitGroup      *sync.WaitGroup
}

func New(config data.Config) (*Forwarder, error) {
	ctx, cancel := context.WithCancel(context.Background())
	waitGroup := &sync.WaitGroup{}
	webForwarder, err := NewWebForwarder(ctx, config.BaseConfig, waitGroup)
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
		waitGroup:      waitGroup,
	}, nil
}
func (o *Forwarder) Stop() error {
	select {
	case <-o.ctx.Done():
		return errors.New("this instance has already stopped")
	default:
		slog.Info("Shutting down all listeners...")
		o.cancelFunc()
		o.waitGroup.Wait()
	}
	return nil
}
func (o *Forwarder) StartAsync() error {
	for _, host := range o.config.Hosts {
		hf, err := NewHostForwarder(o.ctx, o.config.BaseConfig, host, o.webForwarder, o.waitGroup)
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
