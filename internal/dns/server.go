package dns

import (
	"net"
	"strings"
	"time"

	dnslib "github.com/miekg/dns"
	"github.com/st3v3nmw/beacon/internal/config"
	"github.com/st3v3nmw/beacon/internal/querylog"
)

var (
	UDP *dnslib.Server
)

func NewUDPServer(addr string) {
	UDP = &dnslib.Server{
		Addr: addr,
		Net:  "udp",
	}

	UDP.Handler = dnslib.HandlerFunc(handleUDPRequest)
}

func handleUDPRequest(w dnslib.ResponseWriter, r *dnslib.Msg) {
	start := time.Now()
	if len(r.Question) == 0 {
		return
	}

	var m *dnslib.Msg
	qn := r.Question[0]
	domain := strings.TrimSuffix(qn.Name, ".")

	blocked, rules := isBlocked(domain)
	cached := false
	var reason, upstream *string
	if blocked {
		m = blockDomainOnUDP(r)
		blocked = true
		reason = (*string)(rules[0].Category)
	} else {
		var err error
		m, cached, upstream, err = resolve(r)

		if err != nil {
			m = &dnslib.Msg{}
			m.SetReply(r)
			m.RecursionAvailable = true
			m.SetRcode(r, dnslib.RcodeServerFailure)
		}
	}

	w.WriteMsg(m)

	if config.All.QueryLog.Enabled {
		var hostname, ip string
		if config.All.QueryLog.LogClients {
			addr := w.RemoteAddr().(*net.UDPAddr)
			ip = addr.IP.String()
			hostname = lookupHostname(addr.IP)
		} else {
			hostname = "redacted"
		}

		queryType, ok := dnslib.TypeToString[qn.Qtype]
		if !ok {
			queryType = "UNKNOWN"
		}

		end := time.Now()
		querylog.QL.Log(
			querylog.QueryLog{
				Hostname:       hostname,
				IP:             &ip,
				Domain:         domain,
				QueryType:      queryType,
				Cached:         cached,
				Blocked:        blocked,
				BlockReason:    reason,
				Upstream:       upstream,
				ResponseCode:   dnslib.RcodeToString[m.Rcode],
				ResponseTimeMs: int(end.UnixMilli() - start.UnixMilli()),
				Timestamp:      end,
			},
		)
	}
}

func blockDomainOnUDP(r *dnslib.Msg) *dnslib.Msg {
	m := new(dnslib.Msg)
	m.SetReply(r)
	m.RecursionAvailable = true

	qn := r.Question[0]
	switch qn.Qtype {
	case dnslib.TypeA:
		a := &dnslib.A{
			Hdr: dnslib.RR_Header{
				Name:   qn.Name,
				Rrtype: dnslib.TypeA,
				Class:  dnslib.ClassINET,
				Ttl:    uint32(config.All.DNS.BlockingTTL),
			},
			A: net.ParseIP("0.0.0.0"),
		}
		m.Answer = append(m.Answer, a)
		m.SetRcode(r, dnslib.RcodeSuccess)
	case dnslib.TypeAAAA:
		aaaa := &dnslib.AAAA{
			Hdr: dnslib.RR_Header{
				Name:   qn.Name,
				Rrtype: dnslib.TypeAAAA,
				Class:  dnslib.ClassINET,
				Ttl:    uint32(config.All.DNS.BlockingTTL),
			},
			AAAA: net.ParseIP("::"),
		}
		m.Answer = append(m.Answer, aaaa)
		m.SetRcode(r, dnslib.RcodeSuccess)
	default:
		m.SetRcode(r, dnslib.RcodeRefused)
	}

	return m
}
