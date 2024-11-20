package dns

import (
	"fmt"
	"net"
	"slices"
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
	root        = make(map[types.Category]*radix.Tree)
	treeMu      sync.RWMutex
	Cache       otter.CacheWithVariableTTL[string, *dnslib.Msg]
	maxCacheTTL uint32
	minCacheTTL uint32
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

	maxCacheTTL = uint32(config.All.Cache.TTL.Max.Seconds())
	minCacheTTL = uint32(config.All.Cache.TTL.Min.Seconds())

	return nil
}

type Rule struct {
	List     string          `json:"list"`
	Category *types.Category `json:"category"`
	Action   *types.Action   `json:"action"`
}

func summarizeRules(rules []Rule) string {
	block, allow := []Rule{}, []Rule{}
	for _, rule := range rules {
		if *rule.Action == types.ActionBlock {
			block = append(block, rule)
		} else {
			allow = append(allow, rule)
		}
	}

	summary := ""
	if len(allow) > 0 {
		summary = fmt.Sprintf("Allowed by (%s): ", *allow[0].Category)
		for _, rule := range allow {
			summary += rule.List + ", "
		}
	} else {
		summary = fmt.Sprintf("Blocked by (%s): ", *block[0].Category)
		for _, rule := range block {
			summary += rule.List + ", "
		}
	}

	return strings.TrimSuffix(summary, ", ")
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
			var rules []Rule
			if found {
				rules = raw.([]Rule)
			}

			rules = append(rules, Rule{
				List:     name,
				Category: &category,
				Action:   &action,
			})

			tree.Insert(key, rules)
		}

		root[category] = tree
	}
}

func process(r *dnslib.Msg, client string, summarize bool) (*dnslib.Msg, bool, bool, string, *string, *string) {
	var m *dnslib.Msg
	qn := r.Question[0]
	fqdn := strings.TrimSuffix(qn.Name, ".")

	if qn.Qtype == dnslib.TypePTR {
		arpaStripped := strings.ReplaceAll(qn.Name, ".in-addr.arpa.", "")
		ipStr := reverseFQDN(arpaStripped)

		ip := net.ParseIP(ipStr)
		if ip.IsPrivate() {
			// Don't forward reverse DNS lookups for private IP ranges
			m = &dnslib.Msg{}
			m.SetReply(r)
			m.RecursionAvailable = true
			m.SetRcode(r, dnslib.RcodeNameError) // NXDomain

			summary := fmt.Sprintf("Reverse DNS lookup for private IP %s blocked", ipStr)
			category := "rdns-private-ip"
			return m, false, true, summary, &category, nil
		}
	}

	cached := false
	var upstream *string
	blocked, summary, category := isBlocked(fqdn, client, summarize)
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

		sum := ""
		if summarize {
			if cached {
				sum = "Served from local cache"
			} else {
				sum = "Resolved via " + *upstream
			}

			if !blocked && !strings.HasPrefix(summary, "Allow") {
				summary = ""
			}

			if summary != "" {
				summary += "; " + sum
			} else {
				summary = sum
			}
		}
	}

	return m, cached, blocked, summary, category, upstream
}

func isBlocked(domain, client string, summarize bool) (bool, string, *string) {
	treeMu.RLock()
	defer treeMu.RUnlock()

	var summary string
	key := reverseFQDN(domain)
	for category := range root {
		blocked, rules, group, schedule := isBlockedByCategory(key, domain, client, category)
		if summarize && len(rules) > 0 {
			summary = summarizeRules(rules)

			if group != "" {
				summary += fmt.Sprintf("; Group: %s", group)
			}

			if schedule != "" {
				summary += fmt.Sprintf("; Schedule: %s", schedule)
			}
		}

		if blocked {
			cat := string(category)
			return blocked, summary, &cat
		}
	}
	return false, summary, nil
}

func isBlockedByCategory(key, domain, client string, category types.Category) (bool, []Rule, string, string) {
	tree, ok := root[category]
	if !ok {
		return false, nil, "", ""
	}

	var group, schedule string
	prefix, raw, found := tree.LongestPrefix(key)
	if found {
		// check that it is indeed a match
		// in some cases like key = com.serverfault & blocked = com.server
		// this matches, even though it shouldn't
		// so we need to check that serverfault.com has suffix server.com
		base := reverseFQDN(string(prefix))
		if !strings.HasSuffix(domain, base) {
			return false, nil, "", ""
		}

		rules := raw.([]Rule)
		for _, rule := range rules {
			if *rule.Action == types.ActionAllow {
				return false, rules, "", ""
			}
		}

		blocked := len(rules) > 0
		if blocked {
			blocked, group, schedule = config.All.IsCategoryBlocked(client, category)
		}

		return blocked, rules, group, schedule
	}

	return false, nil, "", ""
}

// Reverse domain for better tree structure
// e.g., com.example -> example.com
func reverseFQDN(fqdn string) string {
	parts := strings.Split(fqdn, ".")
	slices.Reverse(parts)
	return strings.Join(parts, ".")
}

func resolve(r *dnslib.Msg) (*dnslib.Msg, bool, *string, error) {
	qn := r.Question[0]
	key := fmt.Sprintf("%s-%d-%d", qn.Name, qn.Qtype, qn.Qclass)
	cached, ok := Cache.Get(key)
	if ok {
		// TODO: This has the TTL of the original request
		// but time has passed so the TTL should be lower!
		m := cached.Copy()
		m.Id = r.Id
		return m, true, nil, nil
	}

	m, upstream, err := forwardToUpstream(r)
	if err != nil {
		return nil, false, upstream, err
	}

	cacheTtl := minAnswerTtl(maxCacheTTL, m.Answer)
	cacheTtl = minAnswerTtl(cacheTtl, m.Ns)
	cacheTtl = minAnswerTtl(cacheTtl, m.Extra)
	if cacheTtl >= minCacheTTL && len(m.Answer)+len(m.Ns)+len(m.Extra) > 0 {
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
