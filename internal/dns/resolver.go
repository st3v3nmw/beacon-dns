package dns

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/armon/go-radix"
	"github.com/maypok86/otter"
	dnslib "github.com/miekg/dns"
	"github.com/st3v3nmw/beacon/internal/lists"
)

var (
	root          map[lists.Category]*radix.Tree
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

type Leaf struct {
	List     string         `json:"list"`
	Category lists.Category `json:"category"`
	Action   lists.Action   `json:"action"`
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

func (f *Filter) Categories() []lists.Category {
	categoryMap := map[lists.Category]bool{
		lists.CategoryAds:            f.Ads,
		lists.CategoryMalware:        f.Malware,
		lists.CategoryAdult:          f.Adult,
		lists.CategoryDating:         f.Dating,
		lists.CategorySocialMedia:    f.SocialMedia,
		lists.CategoryVideoStreaming: f.VideoStreaming,
		lists.CategoryGambling:       f.Gambling,
		lists.CategoryGaming:         f.Gaming,
		lists.CategoryPiracy:         f.Piracy,
		lists.CategoryDrugs:          f.Drugs,
	}

	categories := make([]lists.Category, 0, 10)
	for category, enabled := range categoryMap {
		if enabled {
			categories = append(categories, category)
		}
	}

	return categories
}

func LoadListsToMemory() error {
	root = make(map[lists.Category]*radix.Tree)
	for name, list := range lists.PersistedLists {
		tree, ok := root[list.Category]
		if !ok {
			tree = radix.New()
			root[list.Category] = tree
		}

		for _, domain := range list.Domains {
			key := reverseDomain(domain)

			var leaves []Leaf
			if val, ok := tree.Get(key); ok {
				leaves = val.([]Leaf)
			}
			leaves = append(leaves, Leaf{
				List:     name,
				Category: list.Category,
				Action:   list.Action,
			})

			tree.Insert(key, leaves)
		}
	}
	return nil
}

func isBlocked(domain string, filter *Filter) (bool, []Leaf) {
	key := reverseDomain(domain)
	for _, category := range filter.Categories() {
		blocked, leaves := isBlockedByCategory(key, category)
		if blocked {
			return blocked, leaves
		}
	}
	return false, nil
}

func isBlockedByCategory(key string, category lists.Category) (bool, []Leaf) {
	tree, ok := root[category]
	if !ok {
		return false, nil
	}

	_, val, found := tree.LongestPrefix(key)
	if found {
		leaves := val.([]Leaf)

		for _, leaf := range leaves {
			if leaf.Action == lists.ActionAllow {
				return false, leaves
			}
		}

		return true, leaves
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
	// TODO: include other vars from Request
	key := fmt.Sprintf("%s-%d-%d", qn.Name, qn.Qtype, qn.Qclass)
	cached, ok := Cache.Get(key)
	if ok {
		// TODO: make sure we're not caching geo-specific results
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
	m, _, err := c.Exchange(r, "1.1.1.1:53")
	return m, err
}
