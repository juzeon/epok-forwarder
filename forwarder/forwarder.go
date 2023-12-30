package forwarder

import (
	"context"
	"github.com/juzeon/epok-forwarder/data"
)

type Forwarder struct {
	config         data.Config
	hostForwarders []*HostForwarder
	ctx            context.Context
	cancelFunc     context.CancelFunc
	webForwarder   *WebForwarder
}

func New(config data.Config) error {
	ctx, cancel := context.WithCancel(context.Background())
	webForwarder, err := NewWebForwarder(ctx, config.BaseConfig)
	if err != nil {
		cancel()
		return err
	}
	return (&Forwarder{
		config:         config,
		hostForwarders: nil,
		ctx:            ctx,
		cancelFunc:     cancel,
		webForwarder:   webForwarder,
	}).start()
}
func (o *Forwarder) start() error {
	for _, host := range o.config.Hosts {
		hf, err := NewHostForwarder(o.ctx, o.config.BaseConfig, host, o.webForwarder)
		if err != nil {
			return err
		}
		o.hostForwarders = append(o.hostForwarders, hf)
	}
	err := o.webForwarder.StartAsync()
	if err != nil {
		return err
	}
	return nil
}
