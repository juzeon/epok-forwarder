package forwarder

import (
	"context"
	"github.com/juzeon/epok-forwarder/data"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"sync"
)

type Forwarder struct {
	config         data.Config
	hostForwarders []*HostForwarder
	ctx            context.Context
	cancelFunc     context.CancelFunc

	webForwardTargets []data.WebForwardTarget
	webReverseProxies sync.Map
}

func New(config data.Config) error {
	ctx, cancel := context.WithCancel(context.Background())
	return (&Forwarder{
		config:            config,
		hostForwarders:    nil,
		ctx:               ctx,
		cancelFunc:        cancel,
		webForwardTargets: nil,
		webReverseProxies: sync.Map{},
	}).start()
}
func (o *Forwarder) start() error {
	registerWebForwarder := data.RegisterWebForwarderFunc(func(hostname string, dstIP string,
		dstHttpPort int, dstHttpsPort int) {
		o.webForwardTargets = append(o.webForwardTargets, data.WebForwardTarget{
			Hostname:     hostname,
			DstIP:        dstIP,
			DstHttpPort:  dstHttpPort,
			DstHttpsPort: dstHttpsPort,
		})
	})
	for _, host := range o.config.Hosts {
		hf, err := NewHostForwarder(o.ctx, o.config.BaseConfig, host, registerWebForwarder)
		if err != nil {
			return err
		}
		o.hostForwarders = append(o.hostForwarders, hf)
	}
	err := o.startWebForwarder()
	if err != nil {
		return err
	}
	return nil
}
func (o *Forwarder) startWebForwarder() error {
	server := http.Server{
		Handler: http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {

		}),
		BaseContext: func(listener net.Listener) context.Context {
			return o.ctx
		},
	}
	l, err := net.Listen("tcp", ":"+strconv.Itoa(o.config.Http))
	if err != nil {
		return err
	}
	go func() {
		<-o.ctx.Done()
		l.Close()
	}()
	go func() {
		err = server.Serve(l)
		if err != nil {
			slog.Error("Cannot accept web", "error", err)
		}
	}()
	return nil
}
