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

	UDP.Handler = dnslib.HandlerFunc(handleRequest)
}

func handleRequest(w dnslib.ResponseWriter, q *dnslib.Msg) {
	start := time.Now()
	if len(q.Question) == 0 {
		return
	}

	var ip, hostname string
	if config.All.QueryLog.LogClients {
		addr := w.RemoteAddr().(*net.UDPAddr)
		ip = addr.IP.String()
		hostname = lookupHostname(addr.IP)
	} else {
		ip = "-"
		hostname = "-"
	}

	response := process(q, hostname, true)
	w.WriteMsg(response.Msg)

	if config.All.QueryLog.Enabled {
		qn := q.Question[0]
		queryType, ok := dnslib.TypeToString[qn.Qtype]
		if !ok {
			queryType = "UNKNOWN"
		}

		end := time.Now()
		querylog.QL.Log(
			&querylog.QueryLog{
				Hostname:     hostname,
				IP:           ip,
				Domain:       strings.TrimSuffix(qn.Name, "."),
				QueryType:    queryType,
				Cached:       response.Cached,
				Blocked:      response.Blocked,
				BlockReason:  response.BlockReason,
				Upstream:     response.Upstream,
				ResponseCode: dnslib.RcodeToString[response.Msg.Rcode],
				ResponseTime: int(end.UnixMilli() - start.UnixMilli()),
				Prefetched:   response.Prefetched,
				Timestamp:    start.UTC(),
			},
		)
	}
}

func blockFQDN(q *dnslib.Msg) *dnslib.Msg {
	m := new(dnslib.Msg)
	m.SetReply(q)
	m.RecursionAvailable = true

	qn := q.Question[0]
	switch qn.Qtype {
	case dnslib.TypeA:
		a := &dnslib.A{
			Hdr: dnslib.RR_Header{
				Name:   qn.Name,
				Rrtype: dnslib.TypeA,
				Class:  dnslib.ClassINET,
				Ttl:    300,
			},
			A: net.ParseIP("0.0.0.0"),
		}
		m.Answer = append(m.Answer, a)
		m.SetRcode(q, dnslib.RcodeSuccess)
	case dnslib.TypeAAAA:
		aaaa := &dnslib.AAAA{
			Hdr: dnslib.RR_Header{
				Name:   qn.Name,
				Rrtype: dnslib.TypeAAAA,
				Class:  dnslib.ClassINET,
				Ttl:    300,
			},
			AAAA: net.ParseIP("::"),
		}
		m.Answer = append(m.Answer, aaaa)
		m.SetRcode(q, dnslib.RcodeSuccess)
	default:
		m.SetRcode(q, dnslib.RcodeRefused)
	}

	return m
}
