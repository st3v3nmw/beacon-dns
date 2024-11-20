package dns

import (
	"fmt"
	"maps"

	"github.com/go-playground/validator/v10"
	"github.com/st3v3nmw/beacon/internal/config"
	"github.com/st3v3nmw/beacon/internal/types"
)

type Trace struct {
	Lists     []Rule                            `json:"lists"`
	Groups    map[string]*config.GroupConfig    `json:"groups"`
	Schedules map[string]*config.ScheduleConfig `json:"schedules"`
}

func HandleTrace(fqdn, ipStr string) (*Trace, error) {
	validate := validator.New(validator.WithRequiredStructEnabled())
	if err := validate.Var(fqdn, "fqdn"); err != nil {
		return nil, fmt.Errorf("name must be a valid fqdn")
	}

	lists := findListsForDomain(fqdn)

	seenCats := map[types.Category]bool{}
	groups := map[string]*config.GroupConfig{}
	schedules := map[string]*config.ScheduleConfig{}
	for _, list := range lists {
		cat := *list.Category
		if !seenCats[cat] {
			gs, ss := config.All.Trace(cat)
			maps.Copy(groups, gs)
			maps.Copy(schedules, ss)
			seenCats[cat] = true
		}
	}

	return &Trace{
		Lists:     lists,
		Groups:    groups,
		Schedules: schedules,
	}, nil
}

func findListsForDomain(domain string) []Rule {
	treeMu.RLock()
	defer treeMu.RUnlock()

	key := reverseFQDN(domain)
	lists := []Rule{}
	for category := range root {
		_, rules := isBlockedByCategory(key, domain, category)
		lists = append(lists, rules...)
	}

	return lists
}
