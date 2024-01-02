package forwarder

import (
	"context"
	udpForwarder "github.com/1lann/udp-forward"
	"github.com/juzeon/epok-forwarder/data"
	"io"
	"log/slog"
	"net"
	"strconv"
	"sync"
)

type HostForwarder struct {
	baseConfig data.BaseConfig
	hostConfig data.Host
	dstIP      string
	ctx        context.Context
	waitGroup  *sync.WaitGroup
}

func NewHostForwarder(ctx context.Context, baseConfig data.BaseConfig, hostConfig data.Host,
	webForwarder *WebForwarder, waitGroup *sync.WaitGroup) (*HostForwarder, error) {
	addr, err := net.LookupHost(hostConfig.Host)
	if err != nil {
		return nil, err
	}
	hf := &HostForwarder{
		baseConfig: baseConfig,
		hostConfig: hostConfig,
		dstIP:      addr[0],
		ctx:        ctx,
		waitGroup:  waitGroup,
	}
	for _, forward := range hostConfig.Forwards {
		forward := forward
		firewallArray := data.FirewallArray{
			baseConfig.Firewall,
			hostConfig.Firewall,
			forward.Firewall,
		}
		switch forward.Type {
		case data.ForwardTypePort:
			if err = hf.forwardTCPAsync(forward.ForwardPort.Src, forward.ForwardPort.Dst, firewallArray); err != nil {
				return nil, err
			}
			if err = hf.forwardUDPAsync(forward.ForwardPort.Src, forward.ForwardPort.Dst); err != nil {
				return nil, err
			}
		case data.ForwardTypePortRange:
			ports, err := forward.ForwardPortRange.GetPorts()
			if err != nil {
				return nil, err
			}
			for _, port := range ports {
				if err = hf.forwardTCPAsync(port, port, firewallArray); err != nil {
					return nil, err
				}
				if err = hf.forwardUDPAsync(port, port); err != nil {
					return nil, err
				}
			}
		case data.ForwardTypeWeb:
			for _, hostname := range forward.ForwardWeb.Hostnames {
				webForwarder.RegisterTarget(hostname, hf.dstIP, forward.ForwardWeb.Http, forward.ForwardWeb.Https,
					firewallArray)
			}
		}
	}
	return hf, nil
}
func (o *HostForwarder) forwardUDPAsync(srcPort int, dstPort int) error {
	slog.Info("Register udp forwarder", "src-port", srcPort, "dst-port", dstPort, "dst-ip", o.dstIP)
	f, err := udpForwarder.Forward(":"+strconv.Itoa(srcPort), o.dstIP+":"+strconv.Itoa(dstPort),
		udpForwarder.DefaultTimeout)
	if err != nil {
		return err
	}
	o.waitGroup.Add(1)
	go func() {
		<-o.ctx.Done()
		f.Close()
		slog.Info("Close udp listener", "src-port", srcPort, "dst-port", dstPort, "dst-ip", o.dstIP)
		o.waitGroup.Done()
	}()
	return nil
}
func (o *HostForwarder) forwardTCPAsync(srcPort int, dstPort int, firewallArray data.FirewallArray) error {
	slog.Info("Register tcp forwarder", "src-port", srcPort, "dst-port", dstPort, "dst-ip", o.dstIP)
	l, err := net.Listen("tcp", ":"+strconv.Itoa(srcPort))
	if err != nil {
		return err
	}
	o.waitGroup.Add(1)
	go func() {
		<-o.ctx.Done()
		l.Close()
		slog.Info("Close tcp listener", "src-port", srcPort, "dst-port", dstPort, "dst-ip", o.dstIP)
		o.waitGroup.Done()
	}()
	go func() {
		defer l.Close()
		defer slog.Info("Close tcp listener", "src-port", srcPort, "dst-port", dstPort, "dst-ip", o.dstIP)
		for {
			acceptedConn, err := l.Accept()
			if err != nil {
				slog.Warn("Cannot accept conn", "error", err)
				break
			}
			allow, reason := firewallArray.CheckAllowByAddr(acceptedConn.RemoteAddr().String())
			if !allow {
				slog.Warn("Deny conn", "reason", reason)
				acceptedConn.Close()
				continue
			}
			slog.Info("Accept connection", "addr", l.Addr().String(), "reason", reason)
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
	}()
	return nil
}
