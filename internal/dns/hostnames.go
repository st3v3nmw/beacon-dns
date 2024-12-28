package dns

import (
	"fmt"
	"log/slog"
	"net"
	"os"
	"strings"

	dnslib "github.com/miekg/dns"
	"github.com/st3v3nmw/beacon/internal/config"
	"github.com/st3v3nmw/beacon/internal/types"
	"github.com/st3v3nmw/beacon/pkg/threadsafe"
)

var (
	clientMap = threadsafe.NewExpiryMap[string, string]()
)

func lookupHostname(ip net.IP) string {
	ipStr := ip.String()
	hostname, ok := clientMap.Get(ipStr)
	if ok {
		return hostname
	}

	if h, ok := config.All.ClientLookup.Clients[ipStr]; ok {
		hostname = h
	} else if ip.IsLoopback() {
		hostname = lookupLocalHostname(ipStr)
	} else if ip.IsPrivate() {
		hostname = ipStr
	} else {
		method := config.All.ClientLookup.Method
		switch method {
		case types.ClientLookupRDNS:
			hostname = reverseDNSLookup(ipStr)
		default:
			hostname = ipStr
		}
	}

	clientMap.Set(ipStr, hostname, config.All.ClientLookup.RefreshEvery.Duration)
	return hostname
}

func lookupLocalHostname(ip string) string {
	hostname, err := os.Hostname()
	if err != nil {
		slog.Warn("Error retrieving hostname", "error", err)
		return ip
	}

	return hostname
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

	// Results look like <hostname>.<tailnet-name>.ts.net. or <hostname>.
	ans := m.Answer[0].(*dnslib.PTR)
	return strings.Split(ans.Ptr, ".")[0]
}
