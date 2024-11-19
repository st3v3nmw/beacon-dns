package dns

import (
	"fmt"
	"math"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/armon/go-radix"
	"github.com/maypok86/otter"
	dnslib "github.com/miekg/dns"
	"github.com/st3v3nmw/beacon/internal/config"
	"github.com/st3v3nmw/beacon/internal/types"
)

var (
	root   = make(map[types.Category]*radix.Tree)
	treeMu sync.RWMutex
	Cache  otter.CacheWithVariableTTL[string, *dnslib.Msg]
)

func NewCache() error {
	var err error
	Cache, err = otter.MustBuilder[string, *dnslib.Msg](config.All.Cache.Size).
		CollectStats().
		WithVariableTTL().
		Build()
	if err != nil {
		return err
	}

	return nil
}

type ListRule struct {
	List     string          `json:"list"`
	Category *types.Category `json:"category"`
	Action   *types.Action   `json:"action"`
}

func LoadListToMemory(name string, action types.Action, categories []types.Category, domains []string) {
	treeMu.Lock()
	defer treeMu.Unlock()

	for _, category := range categories {
		tree, ok := root[category]
		if !ok {
			tree = radix.New()
		}

		for _, domain := range domains {
			key := reverseFQDN(domain)

			raw, found := tree.Get(key)
			var rules []ListRule
			if found {
				rules = raw.([]ListRule)
			}

			rules = append(rules, ListRule{
				List:     name,
				Category: &category,
				Action:   &action,
			})

			tree.Insert(key, rules)
		}

		root[category] = tree
	}
}

func process(r *dnslib.Msg) (*dnslib.Msg, bool, bool, []ListRule, *string) {
	var m *dnslib.Msg
	qn := r.Question[0]
	fqdn := strings.TrimSuffix(qn.Name, ".")

	if qn.Qtype == dnslib.TypePTR {
		parts := strings.Split(qn.Name, ".")
		for i := 0; i < len(parts)/2; i++ {
			j := len(parts) - i - 1
			parts[i], parts[j] = parts[j], parts[i]
		}

		ipStr := strings.Join(parts[:4], ".")
		ip := net.ParseIP(ipStr)
		if ip.IsPrivate() {
			// Don't forward reverse DNS lookups for private IP ranges
			m = &dnslib.Msg{}
			m.SetReply(r)
			m.RecursionAvailable = true
			m.SetRcode(r, dnslib.RcodeNameError) // NXDomain
			return m, false, true, nil, nil
		}
	}

	blocked, rules := isBlocked(fqdn)
	cached := false
	var upstream *string
	if blocked {
		m = blockFQDN(r)
		blocked = true
	} else {
		var err error
		m, cached, upstream, err = resolve(r)

		if err != nil {
			m = &dnslib.Msg{}
			m.SetReply(r)
			m.RecursionAvailable = true
			m.SetRcode(r, dnslib.RcodeServerFailure)
		}
	}

	return m, cached, blocked, rules, upstream
}

func isBlocked(domain string) (bool, []ListRule) {
	treeMu.RLock()
	defer treeMu.RUnlock()

	key := reverseFQDN(domain)
	for _, category := range config.All.DNS.Block {
		blocked, rules := isBlockedByCategory(key, domain, category)
		if blocked {
			return blocked, rules
		}
	}
	return false, nil
}

func isBlockedByCategory(key string, domain string, category types.Category) (bool, []ListRule) {
	tree, ok := root[category]
	if !ok {
		return false, nil
	}

	prefix, raw, found := tree.LongestPrefix(key)
	if found {
		// check that it is indeed a match
		// in some cases like key = com.serverfault & blocked = com.server
		// this matches, even though it shouldn't
		// so we need to check that serverfault.com has suffix server.com
		base := reverseFQDN(string(prefix))
		if !strings.HasSuffix(domain, base) {
			return false, nil
		}

		rules := raw.([]ListRule)
		for _, rule := range rules {
			if *rule.Action == types.ActionAllow {
				return false, rules
			}
		}

		return len(rules) > 0, rules
	}

	return false, nil
}

// Reverse domain for better tree structure
// e.g., com.example -> example.com
func reverseFQDN(fqdn string) string {
	parts := strings.Split(fqdn, ".")
	for i := 0; i < len(parts)/2; i++ {
		j := len(parts) - i - 1
		parts[i], parts[j] = parts[j], parts[i]
	}
	return strings.Join(parts, ".")
}

func resolve(r *dnslib.Msg) (*dnslib.Msg, bool, *string, error) {
	qn := r.Question[0]
	key := fmt.Sprintf("%s-%d-%d", qn.Name, qn.Qtype, qn.Qclass)
	cached, ok := Cache.Get(key)
	if ok {
		m := cached.Copy()
		m.Id = r.Id
		return m, true, nil, nil
	}

	m, upstream, err := forwardToUpstream(r)
	if err != nil {
		return nil, false, upstream, err
	}

	maxUint32 := uint32(math.MaxUint32)
	cacheTtl := minAnswerTtl(maxUint32, m.Answer)
	cacheTtl = minAnswerTtl(cacheTtl, m.Ns)
	cacheTtl = minAnswerTtl(cacheTtl, m.Extra)
	if cacheTtl >= 15 && cacheTtl != maxUint32 {
		Cache.Set(key, m, time.Duration(cacheTtl)*time.Second)
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
	// TODO: Replace this
	upstream := config.All.DNS.Upstreams[0]
	serverAddr := fmt.Sprintf("%s:53", upstream)

	c := new(dnslib.Client)
	m, _, err := c.Exchange(r, serverAddr)
	return m, &upstream, err
}
