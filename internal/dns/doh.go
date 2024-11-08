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

// TODO: Work on DO, CD, & Trace. They don't do anything right now.
type Request struct {
	Name  string `json:"name" validate:"required,fqdn"`
	Type  QType  `json:"type"`
	DO    bool   `json:"do"`    // whether the client wants DNSSEC records
	CD    bool   `json:"cd"`    // disable DNSSEC validation
	Trace bool   `json:"trace"` // show what blocklists/allowlists were applied
}

type Response struct {
	Status     int        `json:"Status"`
	TC         bool       `json:"TC"`
	RD         bool       `json:"RD"`
	RA         bool       `json:"RA"`
	AD         bool       `json:"AD"`
	CD         bool       `json:"CD"`
	Question   []Question `json:"Question"`
	Answer     []Answer   `json:"Answer,omitempty"`
	Authority  []Answer   `json:"Authority,omitempty"`
	Additional []Answer   `json:"Additional,omitempty"`
	Comment    string     `json:"Comment,omitempty"`
}

func responseFromMsg(m *dnslib.Msg) *Response {
	qn := m.Question[0]
	answer := rrToAnswer(qn.Name, m.Answer)
	authority := rrToAnswer(qn.Name, m.Ns)
	additional := rrToAnswer(qn.Name, m.Extra)
	question := []Question{
		{
			Name: qn.Name,
			Type: QType(qn.Qtype),
		},
	}
	return &Response{
		Status:     m.Rcode,
		TC:         m.Truncated,
		RD:         m.RecursionDesired,
		RA:         m.RecursionAvailable,
		AD:         m.AuthenticatedData,
		CD:         m.CheckingDisabled,
		Question:   question,
		Answer:     answer,
		Authority:  authority,
		Additional: additional,
	}
}

func rrToAnswer(name string, rr []dnslib.RR) []Answer {
	answer := make([]Answer, len(rr))
	for i, a := range rr {
		answer[i] = Answer{
			Name: name,
			Type: a.Header().Rrtype,
			TTL:  a.Header().Ttl,
		}

		headerStr := a.Header().String()
		rrStr := a.String()
		answer[i].Data = strings.TrimLeft(rrStr, headerStr)
	}
	return answer
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
	blocked, leaves := isBlocked(rq.Name, filter)
	fmt.Println("leaves", leaves)
	if blocked {
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

		m, err := resolve(r)
		if err != nil {
			return nil, err
		}

		return responseFromMsg(m), nil
	}
}

func blockDomainOnDoH(rq *Request) *Response {
	var status int
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
		status = dnslib.RcodeSuccess
	case dnslib.TypeAAAA:
		a := Answer{
			Name: rq.Name,
			Type: dnslib.TypeAAAA,
			TTL:  defaultDNSTTL,
			Data: "::",
		}
		answer = append(answer, a)
		status = dnslib.RcodeSuccess
	default:
		status = dnslib.RcodeRefused
	}

	question := []Question{
		{
			Name: rq.Name,
			Type: rq.Type,
		},
	}
	return &Response{
		Status:   status,
		TC:       false,
		RD:       true,
		RA:       true,
		AD:       true,
		CD:       false,
		Question: question,
		Answer:   answer,
	}
}
