package dns

import (
	"fmt"
	"strconv"
	"strings"

	dnslib "github.com/miekg/dns"
)

type QType uint16

func (f *QType) UnmarshalJSON(data []byte) error {
	strVal := strings.Trim(string(data), "\"")
	intVal, err := strconv.Atoi(strVal)
	if err == nil {
		// confirm that the type exists
		_, ok := dnslib.TypeToString[uint16(intVal)]
		if !ok {
			return fmt.Errorf("unknown type provided")
		}

		*f = QType(intVal)
		return nil
	}

	// the string representation was provided, e.g. A, AAAA
	uint16Val, ok := dnslib.StringToType[strVal]
	if !ok {
		return fmt.Errorf("unknown type provided")
	}

	*f = QType(uint16Val)
	return nil

}

// TODO: Work on DO & CD. They don't do anything right now.
type Request struct {
	Name string `json:"name" validate:"required,fqdn"`
	Type QType  `json:"type"`
	DO   bool   `json:"do"` // whether the client wants DNSSEC records
	CD   bool   `json:"cd"` // disable DNSSEC validation
}

type Response struct {
	Status   int        `json:"Status"`
	TC       bool       `json:"TC"`
	RD       bool       `json:"RD"`
	RA       bool       `json:"RA"`
	AD       bool       `json:"AD"`
	CD       bool       `json:"CD"`
	Question []Question `json:"Question"`
	Answer   []Answer   `json:"Answer"`
	Comment  string     `json:"Comment,omitempty"`
}

type Question struct {
	Name string `json:"name"`
	Type QType  `json:"type"`
}

type Answer struct {
	Name string `json:"name"`
	Type uint16 `json:"type"`
	TTL  uint32 `json:"ttl"`
	Data string `json:"data"`
}

func HandleDoHRequest(rq *Request, filter *Filter) (*Response, error) {
	if isBlocked(rq.Name, Filter{}) {
		return blockDomainOnDoH(rq), nil
	} else {
		r := &dnslib.Msg{
			MsgHdr: dnslib.MsgHdr{
				Opcode:           dnslib.OpcodeQuery,
				RecursionDesired: true,
			},
			Question: []dnslib.Question{
				{
					Name:   rq.Name + ".",
					Qtype:  uint16(rq.Type),
					Qclass: dnslib.ClassINET,
				},
			},
		}

		m, err := forwardToUpstream(r)
		if err != nil {
			return nil, err
		}

		answer := make([]Answer, len(m.Answer))
		for i, a := range m.Answer {
			answer[i] = Answer{
				Name: rq.Name,
				Type: a.Header().Rrtype,
				TTL:  a.Header().Ttl,
			}

			switch t := a.(type) {
			// TODO: Handle other message types
			case *dnslib.A:
				answer[i].Data = t.A.String()
			case *dnslib.AAAA:
				answer[i].Data = t.AAAA.String()
			}
		}

		question := []Question{
			{
				Name: rq.Name,
				Type: rq.Type,
			},
		}
		return &Response{
			Status:   m.Rcode,
			TC:       m.Truncated,
			RD:       m.RecursionDesired,
			RA:       m.RecursionAvailable,
			AD:       m.AuthenticatedData,
			CD:       m.CheckingDisabled,
			Question: question,
			Answer:   answer,
		}, nil
	}
}

func blockDomainOnDoH(rq *Request) *Response {
	answer := make([]Answer, 0, 1)

	switch uint16(rq.Type) {
	case dnslib.TypeA:
		a := Answer{
			Name: rq.Name,
			Type: dnslib.TypeA,
			TTL:  defaultDNSTTL,
			Data: "0.0.0.0",
		}
		answer = append(answer, a)
	case dnslib.TypeAAAA:
		a := Answer{
			Name: rq.Name,
			Type: dnslib.TypeAAAA,
			TTL:  defaultDNSTTL,
			Data: "::",
		}
		answer = append(answer, a)
	}

	question := []Question{
		{
			Name: rq.Name,
			Type: rq.Type,
		},
	}
	return &Response{
		Status:   dnslib.RcodeSuccess,
		TC:       true,
		RD:       true,
		RA:       true,
		AD:       true,
		CD:       true,
		Question: question,
		Answer:   answer,
	}
}
