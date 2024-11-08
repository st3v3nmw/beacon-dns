package dns

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/armon/go-radix"
	dnslib "github.com/miekg/dns"
	"github.com/st3v3nmw/beacon/internal/models"
)

var (
	lists         map[string]*radix.Tree
	defaultDNSTTL uint32 = 300
)

type Leaf struct {
	List       string // list name
	Action     string // allow or block
	IsOverride bool
}

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

func (f *Filter) Categories() []string {
	categories := make([]string, 0, 10)
	if f.Ads {
		categories = append(categories, "ads")
	}
	if f.Malware {
		categories = append(categories, "malware")
	}
	if f.Adult {
		categories = append(categories, "adult")
	}
	if f.Dating {
		categories = append(categories, "dating")
	}
	if f.SocialMedia {
		categories = append(categories, "social-media")
	}
	if f.VideoStreaming {
		categories = append(categories, "video-streaming")
	}
	if f.Gambling {
		categories = append(categories, "gambling")
	}
	if f.Gaming {
		categories = append(categories, "gaming")
	}
	if f.Piracy {
		categories = append(categories, "piracy")
	}
	if f.Drugs {
		categories = append(categories, "drugs")
	}

	return categories
}

func LoadLists() error {
	rows, err := models.DB.Query(`
		SELECT l.name, l.category, e.domain, e.action, e.is_override
		FROM lists l
		JOIN entries e ON e.list_id = l.id
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	lists = make(map[string]*radix.Tree)
	for rows.Next() {
		var name, category, domain, action string
		var isOverride bool
		err := rows.Scan(&name, &category, &domain, &action, &isOverride)
		if err != nil {
			return err
		}

		tree, ok := lists[category]
		if !ok {
			tree = radix.New()
			lists[category] = tree
		}

		key := reverseDomain(domain)

		var leaves []Leaf
		if val, found := tree.Get(key); found {
			leaves = val.([]Leaf)
		}
		leaves = append(leaves, Leaf{
			List:       name,
			Action:     action,
			IsOverride: isOverride,
		})

		tree.Insert(key, leaves)
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

func isBlockedByCategory(key, category string) (bool, []Leaf) {
	tree, ok := lists[category]
	if !ok {
		return false, nil
	}

	if val, found := tree.Get(key); found {
		leaves := val.([]Leaf)

		for _, leaf := range leaves {
			if leaf.IsOverride {
				return leaf.Action == "block", leaves
			}
		}

		return anyLeafBlocks(leaves), leaves
	}

	return false, nil
}

func anyLeafBlocks(leaves []Leaf) bool {
	for _, leaf := range leaves {
		if leaf.Action == "block" {
			return true
		}
	}
	return false
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
		fmt.Printf("got %s (%s) from the cache... %s\n", qn.Name, dnslib.TypeToString[qn.Qtype], key)
		fmt.Println(Cache.Stats())
		return m, nil
	}

	m, err := forwardToUpstream(r)
	if err != nil {
		return nil, err
	}

	maxUint32 := uint32(math.MaxUint32)
	cacheTtl := maxUint32
	for _, ans := range m.Answer {
		ttl := ans.Header().Ttl
		if ttl < cacheTtl {
			cacheTtl = ttl
		}
	}

	if cacheTtl > 30 && cacheTtl != maxUint32 {
		Cache.Set(key, m, time.Duration(cacheTtl-15)*time.Second)

		fmt.Printf("got %s (%s) from upstream, cached for %v\n", qn.Name, dnslib.TypeToString[qn.Qtype], time.Duration(cacheTtl-15)*time.Second)
	}

	return m, nil
}

func forwardToUpstream(r *dnslib.Msg) (*dnslib.Msg, error) {
	c := new(dnslib.Client)
	m, _, err := c.Exchange(r, "1.1.1.1:53")
	return m, err
}
