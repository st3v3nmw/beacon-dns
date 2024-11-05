package dns

import (
	"net"
	"strings"

	"github.com/miekg/dns"
	dnslib "github.com/miekg/dns"
	"github.com/st3v3nmw/beacon/internal/models"
)

var (
	DNS *dnslib.Server
)

func New(addr string) {
	DNS = &dnslib.Server{
		Addr: addr,
		Net:  "udp",
	}

	DNS.Handler = dnslib.HandlerFunc(handleDNSRequest)
}

func Start() error {
	return DNS.ListenAndServe()
}

func handleDNSRequest(w dnslib.ResponseWriter, r *dnslib.Msg) {
	if len(r.Question) == 0 {
		// no question asked
		// TODO: Test how or when this occurs
		// dig +noqr example.com
		// dig +noqr +noquestion example.com
		return
	}

	// TODO: handle multiple DNS requests in one
	qn := r.Question[0]
	domain := strings.TrimSuffix(qn.Name, ".")
	if isBlocked(domain) {
		m := new(dnslib.Msg)
		m.SetReply(r)
		m.RecursionAvailable = true
		m.SetRcode(r, dnslib.RcodeSuccess)

		switch qn.Qtype {
		case dnslib.TypeA:
			a := &dnslib.A{
				Hdr: dns.RR_Header{
					Name:   qn.Name,
					Rrtype: dns.TypeA,
					Class:  dns.ClassINET,
					Ttl:    300,
				},
				A: net.ParseIP("0.0.0.0"),
			}
			m.Answer = append(m.Answer, a)
		case dnslib.TypeAAAA:
			aaaa := &dnslib.AAAA{
				Hdr: dns.RR_Header{
					Name:   qn.Name,
					Rrtype: dns.TypeAAAA,
					Class:  dns.ClassINET,
					Ttl:    300,
				},
				AAAA: net.ParseIP("::"),
			}
			m.Answer = append(m.Answer, aaaa)
		}

		w.WriteMsg(m)
		return
	}

	c := new(dnslib.Client)
	m, _, _ := c.Exchange(r, "1.1.1.1:53")
	w.WriteMsg(m)
}

func isBlocked(domain string) bool {
	var blocked bool
	err := models.DB.Model(&models.ListEntry{}).
		Select("count(*) > 0").
		Where("domain = ?", domain).
		Find(&blocked).
		Error

	if err != nil {
		// TODO: ERROR!
		return false
	}

	return blocked
}
