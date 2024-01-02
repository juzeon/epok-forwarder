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
func TestFirewallArray(t *testing.T) {
	geo.Setup()
	ip := net.ParseIP("223.5.5.5")
	fa := FirewallArray{
		Firewall{
			Allow: "",
			Deny:  "cn",
		},
		Firewall{
			Allow: "223.0.0.0/8",
			Deny:  "",
		},
	}
	a, r := fa.CheckAllow(ip)
	assert.Equal(t, true, a)
	assert.Equal(t, FirewallReasonIPCIDR, r)
	fa = FirewallArray{
		Firewall{
			Allow: "0.0.0.0/0",
			Deny:  "us",
		},
		Firewall{
			Allow: "",
			Deny:  "223.5.5.5",
		},
	}
	a, r = fa.CheckAllow(ip)
	assert.Equal(t, false, a)
	assert.Equal(t, FirewallReasonIPAddress, r)
	fa = FirewallArray{
		Firewall{
			Allow: "",
			Deny:  "us",
		},
		Firewall{
			Allow: "",
			Deny:  "223.6.6.6",
		},
	}
	a, r = fa.CheckAllow(ip)
	assert.Equal(t, true, a)
	assert.Equal(t, FirewallReasonDefault, r)
	fa = FirewallArray{
		Firewall{
			Allow: "223.5.5.5",
			Deny:  "",
		},
		Firewall{
			Allow: "",
			Deny:  "cn",
		},
	}
	a, r = fa.CheckAllow(ip)
	assert.Equal(t, false, a)
	assert.Equal(t, FirewallReasonGeo, r)
	fa = FirewallArray{
		Firewall{
			Allow: "",
			Deny:  "",
		},
		Firewall{
			Allow: "",
			Deny:  "",
		},
	}
	a, r = fa.CheckAllow(ip)
	assert.Equal(t, true, a)
	assert.Equal(t, FirewallReasonDefault, r)
}
