package forwarder

import (
	"context"
	udpForwarder "github.com/1lann/udp-forward"
	"github.com/juzeon/epok-forwarder/data"
	"github.com/juzeon/epok-forwarder/util"
	"io"
	"log/slog"
	"net"
	"strconv"
)

type HostForwarder struct {
	baseConfig data.BaseConfig
	hostConfig data.Host
	dstIP      string
	ctx        context.Context
}

func NewHostForwarder(ctx context.Context, baseConfig data.BaseConfig, hostConfig data.Host,
	webForwarder *WebForwarder) (*HostForwarder, error) {
	addr, err := net.LookupHost(hostConfig.Host)
	if err != nil {
		return nil, err
	}
	hf := &HostForwarder{
		baseConfig: baseConfig,
		hostConfig: hostConfig,
		dstIP:      addr[0],
		ctx:        ctx,
	}
	for _, forward := range hostConfig.Forwards {
		switch forward.Type {
		case data.ForwardTypePort:
			go hf.forwardTCP(forward.ForwardPort.Src, forward.ForwardPort.Dst)
			go hf.forwardUDP(forward.ForwardPort.Src, forward.ForwardPort.Dst)
		case data.ForwardTypePortRange:
			ports, err := forward.ForwardPortRange.GetPorts()
			if err != nil {
				return nil, err
			}
			for _, port := range ports {
				go hf.forwardTCP(port, port)
				go hf.forwardUDP(port, port)
			}
		case data.ForwardTypeWeb:
			for _, hostname := range forward.ForwardWeb.Hostnames {
				webForwarder.RegisterTarget(hostname, hf.dstIP, hostConfig.Http, hostConfig.Https)
			}
		}
	}
	return hf, nil
}
func (o *HostForwarder) forwardUDP(srcPort int, dstPort int) {
	slog.Info("Register udp forwarder", "src-port", srcPort, "dst-port", dstPort, "dst-ip", o.dstIP)
	f, err := udpForwarder.Forward(":"+strconv.Itoa(srcPort), o.dstIP+":"+strconv.Itoa(dstPort),
		udpForwarder.DefaultTimeout)
	if err != nil {
		util.ErrExit(err)
	}
	<-o.ctx.Done()
	f.Close()
	slog.Info("Close udp listener", "src-port", srcPort, "dst-port", dstPort, "dst-ip", o.dstIP)
}
func (o *HostForwarder) forwardTCP(srcPort int, dstPort int) {
	slog.Info("Register tcp forwarder", "src-port", srcPort, "dst-port", dstPort, "dst-ip", o.dstIP)
	l, err := net.Listen("tcp", ":"+strconv.Itoa(srcPort))
	if err != nil {
		util.ErrExit(err)
	}
	go func() {
		<-o.ctx.Done()
		l.Close()
	}()
	defer l.Close()
	defer slog.Info("Close tcp listener", "src-port", srcPort, "dst-port", dstPort, "dst-ip", o.dstIP)
	for {
		acceptedConn, err := l.Accept()
		if err != nil {
			slog.Warn("Cannot accept conn", "error", err)
			break
		}
		slog.Info("Accept connection", "addr", l.Addr().String())
		dialedConn, err := (&net.Dialer{}).DialContext(o.ctx, "tcp",
			net.JoinHostPort(o.dstIP, strconv.Itoa(dstPort)))
		if err != nil {
			slog.Warn("Cannot dial tcp", "error", err)
			acceptedConn.Close()
			continue
		}
		slog.Info("Dial connection", "addr", net.JoinHostPort(o.dstIP, strconv.Itoa(dstPort)))
		go func() {
			defer acceptedConn.Close()
			defer dialedConn.Close()
			io.Copy(acceptedConn, dialedConn)
		}()
		go func() {
			defer acceptedConn.Close()
			defer dialedConn.Close()
			io.Copy(dialedConn, acceptedConn)
		}()
	}
}
