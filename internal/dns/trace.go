package dns

import (
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/go-playground/validator/v10"
	dnslib "github.com/miekg/dns"
)

func getQType(strVal string) (uint16, error) {
	intVal, err := strconv.Atoi(strVal)
	if err == nil {
		// confirm that the type exists
		qtype := uint16(intVal)
		_, ok := dnslib.TypeToString[qtype]
		if !ok {
			return qtype, fmt.Errorf("unknown type provided")
		}

		return qtype, nil
	}

	// the string representation was provided, e.g. A, AAAA
	qtype, ok := dnslib.StringToType[strVal]
	if !ok {
		return 0, fmt.Errorf("unknown type provided")
	}

	return qtype, nil
}

type Trace struct {
	Response       string  `json:"response"`
	Cached         bool    `json:"cached"`
	Blocked        bool    `json:"blocked"`
	Summary        string  `json:"summary"`
	Upstream       *string `json:"upstream"`
	ResponseTimeMs int     `json:"response_time_ms"`
}

func HandleTrace(fqdn, qTypeStr, ipStr string) (*Trace, error) {
	validate := validator.New(validator.WithRequiredStructEnabled())
	if err := validate.Var(fqdn, "fqdn"); err != nil {
		return nil, fmt.Errorf("name must be a valid fqdn")
	}

	qtype, err := getQType(qTypeStr)
	if err != nil {
		return nil, err
	}

	start := time.Now()
	r := &dnslib.Msg{
		MsgHdr: dnslib.MsgHdr{
			Opcode:           dnslib.OpcodeQuery,
			RecursionDesired: true,
		},
		Question: []dnslib.Question{
			{
				Name:   fqdn + ".",
				Qtype:  qtype,
				Qclass: dnslib.ClassINET,
			},
		},
	}

	hostname := lookupHostname(net.ParseIP(ipStr))
	m, cached, blocked, summary, _, upstream := process(r, hostname, true)

	end := time.Now()
	return &Trace{
		Response:       m.String(),
		Cached:         cached,
		Blocked:        blocked,
		Upstream:       upstream,
		Summary:        summary,
		ResponseTimeMs: int(end.UnixMilli() - start.UnixMilli()),
	}, nil
}
