package dns

import (
	"fmt"
	"log/slog"
	"net"
	"os"
	"strings"
	"sync"

	dnslib "github.com/miekg/dns"
	"github.com/st3v3nmw/beacon/internal/config"
	"github.com/st3v3nmw/beacon/internal/types"
)

var (
	clientMap sync.Map
)

func lookupHostname(ip net.IP) string {
	ipStr := ip.String()
	hostname, ok := clientMap.Load(ipStr)
	if ok {
		return hostname.(string)
	}

	if ip.IsLoopback() {
		hostname = lookupLocalHostname()
	} else {
		method := config.All.Hostnames.Method
		switch method {
		case types.HostnameLookupTailscale:
			hostname = lookupHostnameOnTailscale(ipStr)
		default:
			hostname = "unknown"
		}
	}

	clientMap.Store(ipStr, hostname)

	return hostname.(string)
}

func lookupLocalHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		slog.Warn("Error retrieving hostname", "error", err)
		return "unknown"
	}

	return hostname
}

func lookupHostnameOnTailscale(ip string) string {
	// tailscale results look like: <hostname>.<tailnet-name>.ts.net.
	hostname := reverseDNSLookup(ip)
	return strings.Split(hostname, ".")[0]
}

func reverseDNSLookup(ip string) string {
	addr, err := dnslib.ReverseAddr(ip)
	if err != nil {
		slog.Warn("Failed to parse address", "addr", addr)
		return "unknown"
	}

	msg := new(dnslib.Msg)
	msg.SetQuestion(addr, dnslib.TypePTR)
	serverAddr := fmt.Sprintf("%s:53", config.All.Hostnames.Upstream)

	c := new(dnslib.Client)
	m, _, err := c.Exchange(msg, serverAddr)

	if len(m.Answer) == 0 || err != nil {
		slog.Warn("Reverse DNS lookup failed", "error", err)
		return "unknown"
	}

	ans := m.Answer[0].(*dnslib.PTR)
	return ans.Ptr
}
