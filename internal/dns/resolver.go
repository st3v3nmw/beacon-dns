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
	tree   = radix.New()
	treeMu sync.RWMutex
)

type Response struct {
	Msg         *dnslib.Msg
	Cached      bool
	Blocked     bool
	BlockReason *string
	Prefetched  bool
	Upstream    *string
}

func process(q *dnslib.Msg, client string, prefetch bool) *Response {
	qn := q.Question[0]
	fqdn := strings.ToLower(strings.TrimSuffix(qn.Name, "."))

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

	m, cached, upstream, prefetched, err := resolve(q, fqdn, client, prefetch)
	if err != nil {
		slog.Error("an error occurred:", "error", err)
		return &Response{
			Msg:      createErrorResponse(q, dnslib.RcodeServerFailure),
			Upstream: upstream,
		}
	}

	return &Response{
		Msg:        m,
		Cached:     cached,
		Upstream:   upstream,
		Prefetched: prefetched,
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
		// Check that it is indeed a match
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
		}

		for _, rule := range rules {
			if *rule.Action != types.ActionAllow {
				blocked := config.All.IsClientBlocked(client, *rule.Category)
				if blocked {
					cat := string(*rule.Category)
					return true, &cat, rules
				}
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

func resolve(q *dnslib.Msg, fqdn, client string, prefetch bool) (*dnslib.Msg, bool, *string, bool, error) {
	qn := q.Question[0]
	key := fmt.Sprintf("%s-%d-%d", qn.Name, qn.Qtype, qn.Qclass)

	cached, ok := Cache.Get(key)
	if ok && (prefetch || !cached.Stale) {
		if cached.touch() {
			go process(q, client, false)
		}

		if prefetch && config.All.Cache.QueryPatterns.Follow {
			go prefetchRelated(fqdn, client)
		}

		m := cached.Msg.Copy()
		m.Id = q.Id
		return m, true, nil, cached.Prefetched, nil
	}

	m, upstream, err := forwardToUpstream(q)
	if err != nil {
		return nil, false, upstream, false, err
	}

	cacheTtl := minAnswerTtl(math.MaxUint32, m.Answer)
	cacheTtl = minAnswerTtl(cacheTtl, m.Ns)
	cacheTtl = minAnswerTtl(cacheTtl, m.Extra)
	cacheTtl = max(cacheTtl, serveStaleFor)
	if len(m.Answer)+len(m.Ns)+len(m.Extra) > 0 {
		cached := Cached{
			Msg:     m,
			Touched: time.Now(),
			Stale:   false,
			// The initial request will have prefetch = true,
			// the follow ups will have prefetch = false
			Prefetched: !prefetch,
		}
		Cache.Set(key, &cached, time.Duration(cacheTtl)*time.Second)
	}

	if prefetch {
		go prefetchRelated(fqdn, client)
	}

	return m, false, upstream, false, nil
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

func forwardToUpstream(q *dnslib.Msg) (*dnslib.Msg, *string, error) {
	// TODO: Implement proper upstream selection
	upstream := config.All.DNS.Upstreams[0]
	serverAddr := fmt.Sprintf("%s:53", upstream)

	c := &dnslib.Client{ReadTimeout: 15 * time.Second}
	m, _, err := c.Exchange(q, serverAddr)
	return m, &upstream, err
}

func prefetchRelated(fqdn, client string) {
	if prefetched, ok := Prefetch[fqdn]; ok {
		for domain, recordTypes := range prefetched {
			for _, recordType := range recordTypes {
				key := fmt.Sprintf("%s-%s", domain, recordType)
				if _, exists := ongoingPrefetches.LoadOrStore(key, struct{}{}); exists {
					// Prefetch already in progress, skip
					continue
				}

				defer ongoingPrefetches.Delete(key)

				q := new(dnslib.Msg)
				q.SetQuestion(domain+".", dnslib.StringToType[recordType])
				process(q, client, false)
			}
		}
	}
}
