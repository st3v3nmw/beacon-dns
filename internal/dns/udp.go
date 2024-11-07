package dns

import (
	"net"
	"strings"

	dnslib "github.com/miekg/dns"
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

func StartUDPServer() error {
	return UDP.ListenAndServe()
}

func handleUDPRequest(w dnslib.ResponseWriter, r *dnslib.Msg) {
	if len(r.Question) == 0 {
		// no question asked
		// TODO: Test how or when this occurs
		// dig +noqr example.com
		// dig +noqr +noquestion example.com
		return
	}

	var m *dnslib.Msg
	qn := r.Question[0]
	domain := strings.TrimSuffix(qn.Name, ".")
	filter := Filter{
		Ads:     true,
		Malware: true,
	}
	if isBlocked(domain, filter) {
		m = blockDomainOnUDP(r)
	} else {
		var err error
		m, err = forwardToUpstream(r)

		if err != nil {
			// TODO: Handle this
		}
	}

	w.WriteMsg(m)
}

func blockDomainOnUDP(r *dnslib.Msg) *dnslib.Msg {
	m := new(dnslib.Msg)
	m.SetReply(r)
	m.RecursionAvailable = true
	m.SetRcode(r, dnslib.RcodeSuccess)

	qn := r.Question[0]
	switch qn.Qtype {
	case dnslib.TypeA:
		a := &dnslib.A{
			Hdr: dnslib.RR_Header{
				Name:   qn.Name,
				Rrtype: dnslib.TypeA,
				Class:  dnslib.ClassINET,
				Ttl:    defaultDNSTTL,
			},
			A: net.ParseIP("0.0.0.0"),
		}
		m.Answer = append(m.Answer, a)
	case dnslib.TypeAAAA:
		aaaa := &dnslib.AAAA{
			Hdr: dnslib.RR_Header{
				Name:   qn.Name,
				Rrtype: dnslib.TypeAAAA,
				Class:  dnslib.ClassINET,
				Ttl:    defaultDNSTTL,
			},
			AAAA: net.ParseIP("::"),
		}
		m.Answer = append(m.Answer, aaaa)
	}

	return m
}
