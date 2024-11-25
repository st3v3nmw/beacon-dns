package dns

import (
	"fmt"
	"log/slog"
	"math"
	"net"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/armon/go-radix"
	dnslib "github.com/miekg/dns"
	"github.com/st3v3nmw/beacon/internal/config"
	"github.com/st3v3nmw/beacon/internal/types"
)

var (
	Resolver *DNSResolver
	tree     *radix.Tree
	treeMu   sync.RWMutex
)

type Request struct {
	Msg        *dnslib.Msg
	Client     string
	IsPrefetch bool
	Result     chan *Response
}

type Response struct {
	Msg         *dnslib.Msg
	Cached      bool
	Blocked     bool
	BlockReason *string
	Upstream    *string
}

type DNSResolver struct {
	reqChan  chan *Request
	wg       sync.WaitGroup
	shutdown chan struct{}
}

func NewResolver() {
	tree = radix.New()

	Resolver = &DNSResolver{
		reqChan:  make(chan *Request, 1_000),
		shutdown: make(chan struct{}),
	}

	Resolver.wg.Add(2)
	for i := 0; i < 2; i++ {
		go Resolver.worker()
	}
}

func (r *DNSResolver) Await(msg *dnslib.Msg, client string) *Response {
	req := &Request{
		Msg:    msg,
		Client: client,
		Result: make(chan *Response, 1),
	}

	select {
	case r.reqChan <- req:
		return <-req.Result
	case <-time.After(5 * time.Second):
		slog.Error("request timed out")
		return &Response{
			Msg:     createErrorResponse(msg, dnslib.RcodeServerFailure),
			Blocked: true,
		}
	}
}

func (r *DNSResolver) Prefetch(msg *dnslib.Msg, client string) {
	req := &Request{
		Msg:        msg,
		Client:     client,
		IsPrefetch: true,
		Result:     make(chan *Response, 1),
	}

	select {
	case r.reqChan <- req:
	default:
		// Drop prefetch request if queue is full
	}
}

func (r *DNSResolver) worker() {
	defer r.wg.Done()

	for {
		select {
		case req := <-r.reqChan:
			req.Result <- r.process(req.Msg, req.Client, req.IsPrefetch)
		case <-r.shutdown:
			return
		}
	}
}

func (rasd *DNSResolver) process(q *dnslib.Msg, client string, isPrefetch bool) *Response {
	qn := q.Question[0]
	fqdn := strings.TrimSuffix(qn.Name, ".")

	if qn.Qtype == dnslib.TypePTR {
		if resp := handleReverseDNS(q, fqdn); resp != nil {
			return resp
		}
	}

	blocked, category, _ := isBlocked(fqdn, client)
	if blocked {
		return &Response{
			Msg:         blockFQDN(q),
			Blocked:     true,
			BlockReason: category,
		}
	}

	m, cached, upstream, err := resolve(q, client, isPrefetch)
	if err != nil {
		slog.Error("an error occurred:", "error", err)
		return &Response{
			Msg: createErrorResponse(q, dnslib.RcodeServerFailure),
		}
	}

	return &Response{
		Msg:      m,
		Cached:   cached,
		Upstream: upstream,
	}
}

func handleReverseDNS(q *dnslib.Msg, fqdn string) *Response {
	arpaStripped := strings.ReplaceAll(fqdn, ".in-addr.arpa", "")
	ipStr := reverseFQDN(arpaStripped)

	ip := net.ParseIP(ipStr)
	if ip.IsPrivate() {
		// Don't forward reverse DNS lookups for private IP ranges
		why := "rdns-private-ip"
		return &Response{
			Msg:         createErrorResponse(q, dnslib.RcodeNameError),
			Blocked:     true,
			BlockReason: &why,
		}
	}
	return nil
}

func createErrorResponse(q *dnslib.Msg, rcode int) *dnslib.Msg {
	m := new(dnslib.Msg)
	m.SetReply(q)
	m.RecursionAvailable = true
	m.SetRcode(q, rcode)
	return m
}

type Rule struct {
	List     string          `json:"list"`
	Category *types.Category `json:"category"`
	Action   *types.Action   `json:"action"`
}

func LoadListToMemory(name string, action *types.Action, category *types.Category, domains []string) {
	treeMu.Lock()
	defer treeMu.Unlock()

	for _, domain := range domains {
		key := reverseFQDN(domain)

		raw, found := tree.Get(key)
		var rules []Rule
		if found {
			rules = raw.([]Rule)
		}

		rules = append(rules, Rule{
			List:     name,
			Category: category,
			Action:   action,
		})

		tree.Insert(key, rules)
	}
}

func isBlocked(domain, client string) (bool, *string, []Rule) {
	treeMu.RLock()
	defer treeMu.RUnlock()

	key := reverseFQDN(domain)

	prefix, raw, found := tree.LongestPrefix(key)
	if found {
		// check that it is indeed a match
		// in some cases like key = com.serverfault & blocked = com.server
		// this matches, even though it shouldn't
		// so we need to check that serverfault.com has suffix server.com
		base := reverseFQDN(string(prefix))
		if !strings.HasSuffix(domain, base) {
			return false, nil, nil
		}

		rules := raw.([]Rule)
		for _, rule := range rules {
			if *rule.Action == types.ActionAllow {
				return false, nil, rules
			}

			blocked := config.All.IsClientBlocked(client, *rule.Category)
			if blocked {
				cat := string(*rule.Category)
				return true, &cat, rules
			}
		}

		return false, nil, rules
	}

	return false, nil, nil
}

// Reverse domain for better tree structure
// e.g., com.example -> example.com
func reverseFQDN(fqdn string) string {
	parts := strings.Split(fqdn, ".")
	slices.Reverse(parts)
	return strings.Join(parts, ".")
}

func resolve(r *dnslib.Msg, client string, isPrefetch bool) (*dnslib.Msg, bool, *string, error) {
	qn := r.Question[0]
	key := fmt.Sprintf("%s-%d-%d", qn.Name, qn.Qtype, qn.Qclass)

	cached, ok := Cache.Get(key)
	if ok && !(isPrefetch && cached.Stale) {
		if cached.touch() {
			Resolver.Prefetch(r, client)
		}

		m := cached.Msg.Copy()
		m.Id = r.Id
		return m, true, nil, nil
	}

	m, upstream, err := forwardToUpstream(r)
	if err != nil {
		return nil, false, upstream, err
	}

	cacheTtl := minAnswerTtl(math.MaxUint32, m.Answer)
	cacheTtl = minAnswerTtl(cacheTtl, m.Ns)
	cacheTtl = minAnswerTtl(cacheTtl, m.Extra)
	cacheTtl = max(cacheTtl, serve_stale_for)
	if len(m.Answer)+len(m.Ns)+len(m.Extra) > 0 {
		cached := Cached{
			Msg:     m,
			Touched: time.Now(),
			Stale:   false,
		}
		Cache.Set(key, &cached, time.Duration(cacheTtl)*time.Second)
	}

	return m, false, upstream, nil
}

func minAnswerTtl(cacheTtl uint32, rr []dnslib.RR) uint32 {
	for _, ans := range rr {
		ttl := ans.Header().Ttl
		if ttl < cacheTtl && ttl != 0 {
			cacheTtl = ttl
		}
	}
	return cacheTtl
}

func forwardToUpstream(r *dnslib.Msg) (*dnslib.Msg, *string, error) {
	// TODO: Implement proper upstream selection
	upstream := config.All.DNS.Upstreams[0]
	serverAddr := fmt.Sprintf("%s:53", upstream)

	c := new(dnslib.Client)
	m, _, err := c.Exchange(r, serverAddr)
	return m, &upstream, err
}
