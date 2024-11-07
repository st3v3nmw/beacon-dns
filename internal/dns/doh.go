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
	if err != nil {
		// the string representation was provided, e.g. A, AAAA
		fmt.Println("err", err)
		intVal, ok := dnslib.StringToType[strVal]
		if !ok {
			return fmt.Errorf("unknown type provided")
		}

		*f = QType(intVal)
		return nil
	}

	// confirm that the type exists
	_, ok := dnslib.TypeToString[uint16(intVal)]
	if !ok {
		return fmt.Errorf("unknown type provided")
	}

	*f = QType(intVal)
	return nil
}

type Question struct {
	Name string `json:"name" validate:"required,fqdn"`
	Type QType  `json:"type"`
}

type Answer struct {
	Name string `json:"name"`
	Type uint16 `json:"type"`
	TTL  uint32 `json:"ttl"`
	Data string `json:"data"`
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

func HandleDoHRequest(qn *Question, filter *Filter) (*Response, error) {
	if isBlocked(qn.Name, Filter{}) {
		return blockDomainOnDoH(qn), nil
	} else {
		question := []dnslib.Question{
			{
				Name:   qn.Name + ".",
				Qtype:  uint16(qn.Type),
				Qclass: dnslib.ClassINET,
			},
		}

		r := &dnslib.Msg{
			MsgHdr: dnslib.MsgHdr{
				Opcode:           dnslib.OpcodeQuery,
				RecursionDesired: true,
			},
			Question: question,
		}

		m, err := forwardToUpstream(r)
		if err != nil {
			return nil, err
		}

		answer := make([]Answer, len(m.Answer))
		for i, a := range m.Answer {
			answer[i] = Answer{
				Name: qn.Name,
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

		return &Response{
			Status:   m.Rcode,
			TC:       m.Truncated,
			RD:       m.RecursionDesired,
			RA:       m.RecursionAvailable,
			AD:       m.AuthenticatedData,
			CD:       m.CheckingDisabled,
			Question: []Question{*qn},
			Answer:   answer,
		}, nil
	}
}

func blockDomainOnDoH(qn *Question) *Response {
	answer := make([]Answer, 0, 1)

	switch uint16(qn.Type) {
	case dnslib.TypeA:
		a := Answer{
			Name: qn.Name,
			Type: dnslib.TypeA,
			TTL:  defaultDNSTTL,
			Data: "0.0.0.0",
		}
		answer = append(answer, a)
	case dnslib.TypeAAAA:
		a := Answer{
			Name: qn.Name,
			Type: dnslib.TypeAAAA,
			TTL:  defaultDNSTTL,
			Data: "::",
		}
		answer = append(answer, a)
	}

	return &Response{
		Status:   dnslib.RcodeSuccess,
		TC:       true,
		RD:       true,
		RA:       true,
		AD:       true,
		CD:       true,
		Question: []Question{*qn},
		Answer:   answer,
	}
}
