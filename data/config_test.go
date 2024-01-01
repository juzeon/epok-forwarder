package data

import (
	"github.com/juzeon/epok-forwarder/geo"
	"github.com/stretchr/testify/assert"
	"net"
	"testing"
)

func TestFirewall(t *testing.T) {
	geo.Setup()
	f := Firewall{
		Allow: "",
		Deny:  "",
	}
	ip := net.ParseIP("223.5.5.5")
	a, r := f.CheckAllow(ip)
	assert.Equal(t, true, a)
	assert.Equal(t, FirewallReasonDefault, r)
	f = Firewall{
		Allow: "",
		Deny:  "cn",
	}
	a, r = f.CheckAllow(ip)
	assert.Equal(t, false, a)
	assert.Equal(t, FirewallReasonGeo, r)
	f = Firewall{
		Allow: "223.0.0.0/8",
		Deny:  "cn",
	}
	a, r = f.CheckAllow(ip)
	assert.Equal(t, true, a)
	assert.Equal(t, FirewallReasonIPCIDR, r)
	f = Firewall{
		Allow: "us",
		Deny:  "223.5.5.5",
	}
	a, r = f.CheckAllow(ip)
	assert.Equal(t, false, a)
	assert.Equal(t, FirewallReasonIPAddress, r)
}
