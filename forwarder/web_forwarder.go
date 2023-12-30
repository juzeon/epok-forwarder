package forwarder

import (
	"bytes"
	"context"
	"crypto/tls"
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
		dest := net.JoinHostPort(target.DstIP, strconv.Itoa(target.DstHttpsPort))
		slog.Info("Serve https", "dest", dest, "hostname", clientHello.ServerName)
		backendConn, err := net.DialTimeout("tcp", dest,
			5*time.Second)
		if err != nil {
			slog.Warn("Cannot dial backend", "err", err)
			return
		}
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
	go func() {
		<-o.ctx.Done()
		l.Close()
	}()
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				slog.Error("Cannot accept https conn", "error", err)
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
		Handler: http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			target, ok := lo.Find(o.targets, func(item data.WebForwardTarget) bool {
				return wildcard.Match(item.Hostname, request.Host)
			})
			if !ok {
				handleErr(writer, "no hostname matches "+request.Host)
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

func peekClientHello(reader io.Reader) (*tls.ClientHelloInfo, io.Reader, error) {
	peekedBytes := new(bytes.Buffer)
	hello, err := readClientHello(io.TeeReader(reader, peekedBytes))
	if err != nil {
		return nil, nil, err
	}
	return hello, io.MultiReader(peekedBytes, reader), nil
}

type readOnlyConn struct {
	reader io.Reader
}

func (conn readOnlyConn) Read(p []byte) (int, error)         { return conn.reader.Read(p) }
func (conn readOnlyConn) Write(p []byte) (int, error)        { return 0, io.ErrClosedPipe }
func (conn readOnlyConn) Close() error                       { return nil }
func (conn readOnlyConn) LocalAddr() net.Addr                { return nil }
func (conn readOnlyConn) RemoteAddr() net.Addr               { return nil }
func (conn readOnlyConn) SetDeadline(t time.Time) error      { return nil }
func (conn readOnlyConn) SetReadDeadline(t time.Time) error  { return nil }
func (conn readOnlyConn) SetWriteDeadline(t time.Time) error { return nil }

func readClientHello(reader io.Reader) (*tls.ClientHelloInfo, error) {
	var hello *tls.ClientHelloInfo
	err := tls.Server(readOnlyConn{reader: reader}, &tls.Config{
		GetConfigForClient: func(argHello *tls.ClientHelloInfo) (*tls.Config, error) {
			hello = new(tls.ClientHelloInfo)
			*hello = *argHello
			return nil, nil
		},
	}).Handshake()
	if hello == nil {
		return nil, err
	}
	return hello, nil
}
