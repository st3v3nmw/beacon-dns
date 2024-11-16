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
	filter := &Filter{
		Ads:     true,
		Malware: true,
	}
	m := processMsg(r, filter)

	if m != nil {
		w.WriteMsg(m)
	}
}

func processMsg(r *dnslib.Msg, filter *Filter) *dnslib.Msg {
	if len(r.Question) == 0 {
		return nil
	}

	var m *dnslib.Msg
	qn := r.Question[0]
	domain := strings.TrimSuffix(qn.Name, ".")

	blocked, _ := isBlocked(domain, filter)
	if blocked {
		m = blockDomainOnUDP(r)
	} else {
		var err error
		m, err = resolve(r)

		if err != nil {
			m = &dnslib.Msg{}
			m.SetReply(r)
			m.RecursionAvailable = true
			m.SetRcode(r, dnslib.RcodeServerFailure)
		}
	}

	return m
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
				Ttl:    defaultDNSTTL,
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
				Ttl:    defaultDNSTTL,
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
