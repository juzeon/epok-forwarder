package forwarder

import (
	"context"
	"github.com/IGLOU-EU/go-wildcard/v2"
	"github.com/juzeon/epok-forwarder/data"
	"github.com/samber/lo"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"sync"
)

type WebForwarder struct {
	ctx            context.Context
	baseConfig     data.BaseConfig
	targets        []data.WebForwardTarget
	reverseProxies sync.Map
}

func NewWebForwarder(ctx context.Context, baseConfig data.BaseConfig) (*WebForwarder, error) {
	return &WebForwarder{
		ctx:            ctx,
		baseConfig:     baseConfig,
		targets:        nil,
		reverseProxies: sync.Map{},
	}, nil
}
func (o *WebForwarder) RegisterTarget(hostname string, dstIP string, dstHttpPort int, dstHttpsPort int) {
	target := data.WebForwardTarget{
		Hostname:     hostname,
		DstIP:        dstIP,
		DstHttpPort:  dstHttpPort,
		DstHttpsPort: dstHttpsPort,
	}
	slog.Info("Register http forwarder", "target", target)
	o.targets = append(o.targets, target)
}
func (o *WebForwarder) StartAsync() error {
	if err := o.startHttpAsync(); err != nil {
		return err
	}
	if err := o.startHttpsAsync(); err != nil {
		return err
	}
	return nil
}
func (o *WebForwarder) startHttpsAsync() error {
	return nil
}
func (o *WebForwarder) startHttpAsync() error {
	handleErr := func(writer http.ResponseWriter, msg string) {
		writer.Header().Set("Content-Type", "text/plain")
		writer.WriteHeader(400)
		writer.Write([]byte(msg))
	}
	server := http.Server{
		Handler: http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			target, ok := lo.Find(o.targets, func(item data.WebForwardTarget) bool {
				return wildcard.Match(item.Hostname, request.Host)
			})
			if !ok {
				handleErr(writer, "no hostname matches "+request.Host)
				return
			}
			dest := "http://" + target.DstIP + ":" + strconv.Itoa(target.DstHttpPort)
			u, err := url.Parse(dest)
			if err != nil {
				handleErr(writer, err.Error())
				return
			}
			r := httputil.NewSingleHostReverseProxy(u)
			originalDirector := r.Director
			r.Director = func(request *http.Request) {
				originalDirector(request)
				request.Host = target.Hostname
			}
			actualR, _ := o.reverseProxies.LoadOrStore(dest, r)
			r = actualR.(*httputil.ReverseProxy)
			slog.Info("Serve http", "dest", dest, "hostname", target.Hostname)
			r.ServeHTTP(writer, request)
		}),
		BaseContext: func(listener net.Listener) context.Context {
			return o.ctx
		},
	}
	l, err := net.Listen("tcp", ":"+strconv.Itoa(o.baseConfig.Http))
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
