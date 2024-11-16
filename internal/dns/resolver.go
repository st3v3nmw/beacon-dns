package dns

import (
	"fmt"
	"math"
	"strconv"
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
	root          = make(map[types.Category]*radix.Tree)
	treeMu        sync.RWMutex
	Cache         otter.CacheWithVariableTTL[string, *dnslib.Msg]
	defaultDNSTTL uint32 = 300
)

func NewCache() error {
	var err error
	Cache, err = otter.MustBuilder[string, *dnslib.Msg](1_048_576).
		CollectStats().
		WithVariableTTL().
		Build()
	if err != nil {
		return err
	}

	return nil
}

type Rule struct {
	List     string          `json:"list"`
	Category *types.Category `json:"category"`
	Action   *types.Action   `json:"action"`
}

// By default, we filter out ads & malware.
type Filter struct {
	Ads            bool
	Malware        bool
	Adult          bool
	Dating         bool
	SocialMedia    bool
	VideoStreaming bool
	Gambling       bool
	Gaming         bool
	Piracy         bool
	Drugs          bool
}

func NewFilterFromStr(filterStr string) (*Filter, error) {
	mask, err := strconv.Atoi(filterStr)
	if err != nil {
		return nil, err
	}

	if mask >= 1024 {
		return nil, fmt.Errorf("filter must be less than 1024")
	}

	return &Filter{
		Ads:            mask&(1<<0) != 0,
		Malware:        mask&(1<<1) != 0,
		Adult:          mask&(1<<2) != 0,
		Dating:         mask&(1<<3) != 0,
		SocialMedia:    mask&(1<<4) != 0,
		VideoStreaming: mask&(1<<5) != 0,
		Gambling:       mask&(1<<6) != 0,
		Gaming:         mask&(1<<7) != 0,
		Piracy:         mask&(1<<8) != 0,
		Drugs:          mask&(1<<9) != 0,
	}, nil
}

func (f *Filter) Categories() []types.Category {
	categoryMap := map[types.Category]bool{
		types.CategoryAds:            f.Ads,
		types.CategoryMalware:        f.Malware,
		types.CategoryAdult:          f.Adult,
		types.CategoryDating:         f.Dating,
		types.CategorySocialMedia:    f.SocialMedia,
		types.CategoryVideoStreaming: f.VideoStreaming,
		types.CategoryGambling:       f.Gambling,
		types.CategoryGaming:         f.Gaming,
		types.CategoryPiracy:         f.Piracy,
		types.CategoryDrugs:          f.Drugs,
	}

	categories := make([]types.Category, 0, 10)
	for category, enabled := range categoryMap {
		if enabled {
			categories = append(categories, category)
		}
	}

	return categories
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
			key := reverseDomain(domain)

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

func isBlocked(domain string, filter *Filter) (bool, []Rule) {
	treeMu.RLock()
	defer treeMu.RUnlock()

	key := reverseDomain(domain)
	for _, category := range filter.Categories() {
		blocked, rules := isBlockedByCategory(key, domain, category)
		if blocked {
			return blocked, rules
		}
	}
	return false, nil
}

func isBlockedByCategory(key string, domain string, category types.Category) (bool, []Rule) {
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
		base := reverseDomain(string(prefix))
		if !strings.HasSuffix(domain, base) {
			return false, nil
		}

		rules := raw.([]Rule)
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
func reverseDomain(domain string) string {
	parts := strings.Split(domain, ".")
	for i := 0; i < len(parts)/2; i++ {
		j := len(parts) - i - 1
		parts[i], parts[j] = parts[j], parts[i]
	}
	return strings.Join(parts, ".")
}

func resolve(r *dnslib.Msg) (*dnslib.Msg, error) {
	qn := r.Question[0]
	key := fmt.Sprintf("%s-%d-%d", qn.Name, qn.Qtype, qn.Qclass)
	cached, ok := Cache.Get(key)
	if ok {
		m := cached.Copy()
		m.Id = r.Id
		return m, nil
	}

	m, err := forwardToUpstream(r)
	if err != nil {
		return nil, err
	}

	maxUint32 := uint32(math.MaxUint32)
	cacheTtl := minAnswerTtl(maxUint32, m.Answer)
	cacheTtl = minAnswerTtl(cacheTtl, m.Ns)
	cacheTtl = minAnswerTtl(cacheTtl, m.Extra)
	if cacheTtl > 30 && cacheTtl != maxUint32 {
		Cache.Set(key, m, time.Duration(cacheTtl-15)*time.Second)
	}

	return m, nil
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

func forwardToUpstream(r *dnslib.Msg) (*dnslib.Msg, error) {
	c := new(dnslib.Client)
	// TODO: Replace this
	addr := fmt.Sprintf("%s:53", config.All.DNS.Upstreams[0])
	m, _, err := c.Exchange(r, addr)
	return m, err
}
