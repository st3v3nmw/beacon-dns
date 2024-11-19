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

	if h, ok := config.All.ClientLookup.Clients[ipStr]; ok {
		hostname = h
	} else if ip.IsLoopback() {
		hostname = lookupLocalHostname(ipStr)
	} else {
		method := config.All.ClientLookup.Method
		switch method {
		case types.ClientLookupTailscale:
			hostname = lookupHostnameOnTailscale(ipStr)
		default:
			hostname = ipStr
		}
	}

	clientMap.Store(ipStr, hostname)

	return hostname.(string)
}

func lookupLocalHostname(ip string) string {
	hostname, err := os.Hostname()
	if err != nil {
		slog.Warn("Error retrieving hostname", "error", err)
		return ip
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
		return ip
	}

	msg := new(dnslib.Msg)
	msg.SetQuestion(addr, dnslib.TypePTR)
	serverAddr := fmt.Sprintf("%s:53", config.All.ClientLookup.Upstream)

	c := new(dnslib.Client)
	m, _, err := c.Exchange(msg, serverAddr)

	if len(m.Answer) == 0 || err != nil {
		slog.Warn("Reverse DNS lookup failed", "error", err)
		return ip
	}

	ans := m.Answer[0].(*dnslib.PTR)
	return ans.Ptr
}
