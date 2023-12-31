package forwarder

import (
	"context"
	"github.com/IGLOU-EU/go-wildcard/v2"
	"github.com/juzeon/epok-forwarder/data"
	"github.com/samber/lo"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"sync"
	"time"
)

type WebForwarder struct {
	ctx            context.Context
	baseConfig     data.BaseConfig
	targets        []data.WebForwardTarget
	reverseProxies sync.Map
	waitGroup      *sync.WaitGroup
}

func NewWebForwarder(ctx context.Context, baseConfig data.BaseConfig, waitGroup *sync.WaitGroup) (*WebForwarder, error) {
	return &WebForwarder{
		ctx:            ctx,
		baseConfig:     baseConfig,
		targets:        nil,
		reverseProxies: sync.Map{},
		waitGroup:      waitGroup,
	}, nil
}
func (o *WebForwarder) RegisterTarget(hostname string, dstIP string, dstHttpPort int, dstHttpsPort int,
	firewallArray data.FirewallArray) {
	target := data.WebForwardTarget{
		Hostname:      hostname,
		DstIP:         dstIP,
		DstHttpPort:   dstHttpPort,
		DstHttpsPort:  dstHttpsPort,
		FirewallArray: firewallArray,
	}
	slog.Info("Register web forwarder", "target", target)
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
	l, err := net.Listen("tcp", ":"+strconv.Itoa(o.baseConfig.Https))
	if err != nil {
		return err
	}
	handleConnection := func(clientConn net.Conn) {
		streaming := false
		defer func() {
			if !streaming {
				clientConn.Close()
			}
		}()
		if err := clientConn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
			slog.Warn("Cannot set read deadline", "err", err)
			return
		}
		clientHello, clientReader, err := peekClientHello(clientConn)
		if err != nil {
			slog.Warn("Cannot peek client hello", "err", err)
			return
		}
		if err := clientConn.SetReadDeadline(time.Time{}); err != nil {
			slog.Warn("Cannot set read deadline", "err", err)
			return
		}
		target, ok := lo.Find(o.targets, func(item data.WebForwardTarget) bool {
			return wildcard.Match(item.Hostname, clientHello.ServerName)
		})
		if !ok {
			slog.Warn("No hostname matches", "hostname", clientHello.ServerName)
			return
		}
		allow, reason := target.FirewallArray.CheckAllowByAddr(clientConn.RemoteAddr().String())
		if !allow {
			slog.Warn("Deny https conn", "reason", reason)
			return
		}
		dest := net.JoinHostPort(target.DstIP, strconv.Itoa(target.DstHttpsPort))
		slog.Info("Serve https", "dest", dest, "hostname", clientHello.ServerName, "reason", reason)
		backendConn, err := net.DialTimeout("tcp", dest,
			5*time.Second)
		if err != nil {
			slog.Warn("Cannot dial backend", "err", err)
			return
		}
		streaming = true
		go func() {
			io.Copy(clientConn, backendConn)
			clientConn.Close()
			backendConn.Close()
		}()
		go func() {
			io.Copy(backendConn, clientReader)
			clientConn.Close()
			backendConn.Close()
		}()
	}
	o.waitGroup.Add(1)
	go func() {
		<-o.ctx.Done()
		l.Close()
		o.waitGroup.Done()
	}()
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				slog.Warn("Cannot accept https conn", "error", err)
				break
			}
			go handleConnection(conn)
		}
	}()
	return nil
}
func (o *WebForwarder) startHttpAsync() error {
	handleErr := func(writer http.ResponseWriter, msg string) {
		writer.Header().Set("Content-Type", "text/plain")
		writer.WriteHeader(400)
		writer.Write([]byte(msg))
	}
	server := http.Server{
		ReadTimeout:  20 * time.Second,
		WriteTimeout: 20 * time.Second,
		IdleTimeout:  120 * time.Second,
		Handler: http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			target, ok := lo.Find(o.targets, func(item data.WebForwardTarget) bool {
				return wildcard.Match(item.Hostname, request.Host)
			})
			if !ok {
				handleErr(writer, "no hostname matches "+request.Host)
				return
			}
			allow, reason := target.FirewallArray.CheckAllowByAddr(request.RemoteAddr)
			if !allow {
				slog.Warn("Deny http conn", "reason", reason)
				return
			}
			dest := "http://" + net.JoinHostPort(target.DstIP, strconv.Itoa(target.DstHttpPort))
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
			slog.Info("Serve http", "dest", dest, "hostname", target.Hostname, "reason", reason)
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
	o.waitGroup.Add(1)
	go func() {
		<-o.ctx.Done()
		l.Close()
		o.waitGroup.Done()
	}()
	go func() {
		err = server.Serve(l)
		if err != nil {
			slog.Warn("Cannot accept web", "error", err)
		}
	}()
	return nil
}
