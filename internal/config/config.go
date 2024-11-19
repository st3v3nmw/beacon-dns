package config

import (
	"fmt"
	"net"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/st3v3nmw/beacon/internal/types"
)

var (
	All      Config
	Location *time.Location
)

type Config struct {
	DNS          *DNSConfig                 `yaml:"dns" json:"dns"`
	Cache        *CacheConfig               `yaml:"cache" json:"cache"`
	API          *APIConfig                 `yaml:"api" json:"api"`
	ClientLookup *ClientLookupConfig        `yaml:"client_lookup" json:"client_lookup"`
	Groups       map[string]*GroupConfig    `yaml:"groups" json:"groups"`
	Schedules    map[string]*ScheduleConfig `yaml:"schedules" json:"schedules"`
	QueryLog     *QueryLogConfig            `yaml:"querylog" json:"querylog"`
	DHCP         *DHCPConfig                `yaml:"dhcp" json:"dhcp"`
	Sources      *SourcesConfig             `yaml:"sources" json:"sources"`
}

func (c *Config) precompute() {
	for _, group := range c.Groups {
		group.precompute()
	}

	for _, sched := range c.Schedules {
		sched.precompute()
	}
}

// Returns all the blocked categories across the groups & schedules.
func (c *Config) BlockedCategories() []types.Category {
	blocked := []types.Category{}
	for _, group := range c.Groups {
		blocked = append(blocked, group.Block...)
	}
	for _, sched := range c.Schedules {
		blocked = append(blocked, sched.Block...)
	}
	return blocked
}

// TODO: Refactor this... for performance
func (c *Config) IsCategoryBlocked(client string, category types.Category) bool {
	// Check all groups
	groups := map[string]bool{}
	for groupName, group := range c.Groups {
		clientInGroup := len(group.Devices) == 0 || group.devices[client]
		if clientInGroup {
			groups[groupName] = true
			if group.block[category] {
				return true
			}
		}
	}

	// Check schedules
	now := time.Now().In(Location)
	today := now.Weekday()
	minutes := now.Hour()*60 + now.Minute()
	for _, sched := range c.Schedules {
		if !sched.block[category] {
			continue
		}

		applies := false
		for _, group := range sched.ApplyTo {
			if groups[group] {
				applies = true
				break
			}
		}
		if !applies {
			continue
		}

		for _, when := range sched.When {
			if !slices.Contains(when.Days, today) {
				continue
			}

			for _, period := range when.Periods {
				if period.start <= period.end {
					// Normal case: within the same day
					if minutes >= period.start && minutes <= period.end {
						return true
					}
				} else {
					// Spans midnight
					if minutes >= period.start || minutes <= period.end {
						return true
					}
				}
			}
		}

	}

	return false
}

type DNSConfig struct {
	Port      uint16   `yaml:"port" json:"port"`
	Upstreams []string `yaml:"upstreams" json:"upstreams"`
	Timezone  string   `yaml:"timezone"`
}

type CacheConfig struct {
	Size int            `yaml:"size" json:"size"`
	TTL  CacheTTLConfig `yaml:"ttl" json:"ttl"`
}

type CacheTTLConfig struct {
	Min time.Duration `yaml:"min" json:"min"`
	Max time.Duration `yaml:"max" json:"max"`
}

type APIConfig struct {
	Port uint16 `yaml:"port" json:"port"`
}

type ClientLookupConfig struct {
	Upstream string                   `yaml:"upstream" json:"upstream"`
	Method   types.ClientLookupMethod `yaml:"method" json:"method"`
	Clients  map[string]net.IP        `yaml:"clients" json:"clients"`
}

type GroupConfig struct {
	Devices    []string         `yaml:"devices" json:"devices"`
	Block      []types.Category `yaml:"block" json:"block"`
	SafeSearch bool             `yaml:"safe_search" json:"safe_search"`

	// for quick lookups
	devices map[string]bool
	block   map[types.Category]bool
}

func (g *GroupConfig) precompute() {
	g.devices = make(map[string]bool)
	for _, device := range g.Devices {
		g.devices[device] = true
	}

	g.block = make(map[types.Category]bool)
	for _, cat := range g.Block {
		g.block[cat] = true
	}
}

type ScheduleConfig struct {
	ApplyTo []string         `yaml:"apply_to" json:"apply_to"`
	When    []*ScheduleWhen  `yaml:"when" json:"when"`
	Block   []types.Category `yaml:"block" json:"block"`

	// for quick lookups
	applyTo map[string]bool
	block   map[types.Category]bool
}

func (s *ScheduleConfig) precompute() {
	s.applyTo = make(map[string]bool)
	for _, group := range s.ApplyTo {
		s.applyTo[group] = true
	}

	s.block = make(map[types.Category]bool)
	for _, cat := range s.Block {
		s.block[cat] = true
	}

	for _, when := range s.When {
		when.precompute()
	}
}

type ScheduleWhen struct {
	Days    []time.Weekday    `yaml:"days" json:"days"`
	Periods []*SchedulePeriod `yaml:"periods" json:"periods"`
}

func (w *ScheduleWhen) precompute() {
	for _, period := range w.Periods {
		period.precompute()
	}
}

func (w *ScheduleWhen) UnmarshalYAML(data []byte) error {
	type Alias ScheduleWhen
	aux := &struct {
		Days    []string          `yaml:"days"`
		Periods []*SchedulePeriod `yaml:"periods"`
		*Alias
	}{
		Alias: (*Alias)(w),
	}
	if err := yaml.Unmarshal(data, aux); err != nil {
		return err
	}

	for _, day := range aux.Days {
		weekday, err := parseDay(day)
		if err != nil {
			return fmt.Errorf("invalid day: %s", day)
		}
		w.Days = append(w.Days, weekday)
	}

	w.Periods = aux.Periods
	return nil
}

func parseDay(day string) (time.Weekday, error) {
	switch strings.ToLower(day) {
	case "sun", "sunday":
		return time.Sunday, nil
	case "mon", "monday":
		return time.Monday, nil
	case "tue", "tuesday":
		return time.Tuesday, nil
	case "wed", "wednesday":
		return time.Wednesday, nil
	case "thur", "thu", "thursday":
		return time.Thursday, nil
	case "fri", "friday":
		return time.Friday, nil
	case "sat", "saturday":
		return time.Saturday, nil
	}
	return 0, fmt.Errorf("invalid day: %s", day)
}

type SchedulePeriod struct {
	Start time.Time `yaml:"start" json:"start"`
	End   time.Time `yaml:"end" json:"end"`

	// for quick lookups
	start int
	end   int
}

func (p *SchedulePeriod) precompute() {
	p.start = p.Start.Hour()*60 + p.Start.Minute()
	p.end = p.End.Hour()*60 + p.End.Minute()
}

func (p *SchedulePeriod) UnmarshalYAML(data []byte) error {
	type Alias SchedulePeriod
	aux := &struct {
		Start string `yaml:"start"`
		End   string `yaml:"end"`
		*Alias
	}{
		Alias: (*Alias)(p),
	}
	if err := yaml.Unmarshal(data, aux); err != nil {
		return err
	}

	layout := "15:04"
	start, err := time.Parse(layout, aux.Start)
	if err != nil {
		return fmt.Errorf("invalid start time: %s", aux.Start)
	}
	end, err := time.Parse(layout, aux.End)
	if err != nil {
		return fmt.Errorf("invalid end time: %s", aux.End)
	}

	if start.Compare(end) == 0 {
		return fmt.Errorf("start & end time cannot be the same: %s", aux.Start)
	}

	p.Start = start
	p.End = end
	return nil
}

type QueryLogConfig struct {
	Enabled        bool   `yaml:"enabled" json:"enabled"`
	LogClients     bool   `yaml:"log_clients" json:"log_clients"`
	QueryRetention string `yaml:"query_retention" json:"query_retention"`
	StatsRetention string `yaml:"stats_retention" json:"stats_retention"`
}

type DHCPConfig struct {
	Enabled bool `yaml:"enabled" json:"enabled"`
}

type SourcesConfig struct {
	UpdateInterval time.Duration      `yaml:"update_interval" json:"update_interval"`
	Lists          []SourceListConfig `yaml:"lists" json:"lists"`
}

type SourceListConfig struct {
	Name       string             `yaml:"name" json:"name"`
	URL        string             `yaml:"url" json:"url"`
	Categories []types.Category   `yaml:"categories" json:"categories"`
	Action     types.Action       `yaml:"action" json:"action"`
	Format     types.SourceFormat `yaml:"format" json:"format"`
}

func Read(filePath string) error {
	file, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	// Set defaults
	All.DNS = &DNSConfig{
		Port:      53,
		Upstreams: []string{"1.1.1.1", "8.8.8.8"},
		Timezone:  time.Now().Location().String(),
	}

	All.Cache = &CacheConfig{
		Size: 100_000,
		TTL: CacheTTLConfig{
			Min: 15 * time.Second,
			Max: 24 * time.Hour,
		},
	}

	All.API = &APIConfig{Port: 80}

	allDevicesGroup := &GroupConfig{
		Block: []types.Category{types.CategoryAds, types.CategoryMalware},
	}
	All.Groups = map[string]*GroupConfig{"all": allDevicesGroup}

	All.QueryLog = &QueryLogConfig{
		Enabled:        true,
		LogClients:     true,
		QueryRetention: "90d",
		StatsRetention: "365d",
	}

	minUpdateInterval := 24 * time.Hour
	All.Sources = &SourcesConfig{
		UpdateInterval: minUpdateInterval,
		Lists:          getDefaultSources(),
	}

	err = yaml.Unmarshal(file, &All)
	if err != nil {
		return err
	}

	All.precompute()

	if All.Sources.UpdateInterval < minUpdateInterval {
		All.Sources.UpdateInterval = minUpdateInterval
	}

	Location, err = time.LoadLocation(All.DNS.Timezone)
	return err
}

// Get the blocklist & allowlist sources
func getDefaultSources() []SourceListConfig {
	// Blocklists
	blocklists := []SourceListConfig{
		// ads, trackers
		{
			Name:       "olbat:ut1-blacklists:publicite",
			URL:        "https://raw.githubusercontent.com/olbat/ut1-blacklists/master/blacklists/publicite/domains",
			Categories: []types.Category{types.CategoryAds},
			Action:     types.ActionBlock,
			Format:     types.SourceFormatDomains,
		},
		{
			Name:       "peter-lowe:adservers",
			URL:        "https://pgl.yoyo.org/adservers/serverlist.php?hostformat=hosts",
			Categories: []types.Category{types.CategoryAds},
			Action:     types.ActionBlock,
			Format:     types.SourceFormatHosts,
		},
		{
			Name:       "firebog:easy-privacy",
			URL:        "https://v.firebog.net/hosts/Easyprivacy.txt",
			Categories: []types.Category{types.CategoryAds},
			Action:     types.ActionBlock,
			Format:     types.SourceFormatDomains,
		},
		// malware, ransomware, phishing, cryptojacking, stalkerware
		{
			Name:       "olbat:ut1-blacklists:malware",
			URL:        "https://raw.githubusercontent.com/olbat/ut1-blacklists/master/blacklists/malware/domains",
			Categories: []types.Category{types.CategoryMalware},
			Action:     types.ActionBlock,
			Format:     types.SourceFormatDomains,
		},
		{
			Name:       "olbat:ut1-blacklists:phishing",
			URL:        "https://raw.githubusercontent.com/olbat/ut1-blacklists/master/blacklists/phishing/domains",
			Categories: []types.Category{types.CategoryMalware},
			Action:     types.ActionBlock,
			Format:     types.SourceFormatDomains,
		},
		{
			Name:       "olbat:ut1-blacklists:cryptojacking",
			URL:        "https://raw.githubusercontent.com/olbat/ut1-blacklists/master/blacklists/cryptojacking/domains",
			Categories: []types.Category{types.CategoryMalware},
			Action:     types.ActionBlock,
			Format:     types.SourceFormatDomains,
		},
		{
			Name:       "stamparm:ipsum:level-3",
			URL:        "https://raw.githubusercontent.com/stamparm/ipsum/refs/heads/master/levels/3.txt",
			Categories: []types.Category{types.CategoryMalware},
			Action:     types.ActionBlock,
			Format:     types.SourceFormatIps,
		},
		// adult content
		{
			Name:       "sinfonietta:hostfiles:pornography-hosts",
			URL:        "https://raw.githubusercontent.com/Sinfonietta/hostfiles/master/pornography-hosts",
			Categories: []types.Category{types.CategoryAdult},
			Action:     types.ActionBlock,
			Format:     types.SourceFormatHosts,
		},
		{
			Name:       "steven-black:hosts:porn-only",
			URL:        "https://raw.githubusercontent.com/StevenBlack/hosts/master/alternates/porn-only/hosts",
			Categories: []types.Category{types.CategoryAdult},
			Action:     types.ActionBlock,
			Format:     types.SourceFormatHosts,
		},
		// dating
		{
			Name:       "olbat:ut1-blacklists:dating",
			URL:        "https://raw.githubusercontent.com/olbat/ut1-blacklists/master/blacklists/dating/domains",
			Categories: []types.Category{types.CategoryDating},
			Action:     types.ActionBlock,
			Format:     types.SourceFormatDomains,
		},
		// social media
		{
			Name:       "olbat:ut1-blacklists:social_networks",
			URL:        "https://raw.githubusercontent.com/olbat/ut1-blacklists/master/blacklists/social_networks/domains",
			Categories: []types.Category{types.CategorySocialMedia},
			Action:     types.ActionBlock,
			Format:     types.SourceFormatDomains,
		},
		// video streaming platforms
		{
			Name:       "beacon-dns-lists:blocklists:video-streaming",
			URL:        "https://raw.githubusercontent.com/st3v3nmw/beacon-dns-lists/main/blocklists/video-streaming",
			Categories: []types.Category{types.CategoryVideoStreaming},
			Action:     types.ActionBlock,
			Format:     types.SourceFormatDomains,
		},
		// gambling
		{
			Name:       "olbat:ut1-blacklists:gambling",
			URL:        "https://raw.githubusercontent.com/olbat/ut1-blacklists/master/blacklists/gambling/domains",
			Categories: []types.Category{types.CategoryGambling},
			Action:     types.ActionBlock,
			Format:     types.SourceFormatDomains,
		},
		{
			Name:       "sinfonietta:hostfiles:gambling-hosts",
			URL:        "https://raw.githubusercontent.com/Sinfonietta/hostfiles/master/gambling-hosts",
			Categories: []types.Category{types.CategoryGambling},
			Action:     types.ActionBlock,
			Format:     types.SourceFormatHosts,
		},
		// gaming
		{
			Name:       "olbat:ut1-blacklists:games",
			URL:        "https://raw.githubusercontent.com/olbat/ut1-blacklists/master/blacklists/games/domains",
			Categories: []types.Category{types.CategoryGaming},
			Action:     types.ActionBlock,
			Format:     types.SourceFormatDomains,
		},
		// piracy, torrents
		{
			Name:       "hagezi:dns-blocklists:anti.piracy-onlydomains",
			URL:        "https://raw.githubusercontent.com/hagezi/dns-blocklists/main/wildcard/anti.piracy-onlydomains.txt",
			Categories: []types.Category{types.CategoryPiracy},
			Action:     types.ActionBlock,
			Format:     types.SourceFormatDomains,
		},
		// drugs
		{
			Name:       "olbat:ut1-blacklists:drugs",
			URL:        "https://raw.githubusercontent.com/olbat/ut1-blacklists/master/blacklists/drogue/domains",
			Categories: []types.Category{types.CategoryDrugs},
			Action:     types.ActionBlock,
			Format:     types.SourceFormatDomains,
		},
	}

	// Allowlists have higher precedence than blocklists
	// We primarily use blocklists as filters and allowlists to
	// remove false positives in a category
	allowlists := []SourceListConfig{
		// ads, trackers
		{
			Name:       "beacon-dns-lists:allowlists:ads",
			URL:        "https://raw.githubusercontent.com/st3v3nmw/beacon-dns-lists/main/allowlists/ads",
			Categories: []types.Category{types.CategoryAds},
			Action:     types.ActionAllow,
			Format:     types.SourceFormatDomains,
		},
		// malware, ransomware, phishing, cryptojacking, stalkerware
		{
			Name:       "beacon-dns-lists:allowlists:malware",
			URL:        "https://raw.githubusercontent.com/st3v3nmw/beacon-dns-lists/main/allowlists/malware",
			Categories: []types.Category{types.CategoryMalware},
			Action:     types.ActionAllow,
			Format:     types.SourceFormatDomains,
		},
		// adult content
		{
			Name:       "beacon-dns-lists:allowlists:adult",
			URL:        "https://raw.githubusercontent.com/st3v3nmw/beacon-dns-lists/main/allowlists/adult",
			Categories: []types.Category{types.CategoryAdult},
			Action:     types.ActionAllow,
			Format:     types.SourceFormatDomains,
		},
		// dating
		{
			Name:       "beacon-dns-lists:allowlists:dating",
			URL:        "https://raw.githubusercontent.com/st3v3nmw/beacon-dns-lists/main/allowlists/dating",
			Categories: []types.Category{types.CategoryDating},
			Action:     types.ActionAllow,
			Format:     types.SourceFormatDomains,
		},
		// social media
		{
			Name:       "beacon-dns-lists:allowlists:social-media",
			URL:        "https://raw.githubusercontent.com/st3v3nmw/beacon-dns-lists/main/allowlists/social-media",
			Categories: []types.Category{types.CategorySocialMedia},
			Action:     types.ActionAllow,
			Format:     types.SourceFormatDomains,
		},
		// video streaming platforms
		{
			Name:       "beacon-dns-lists:allowlists:video-streaming",
			URL:        "https://raw.githubusercontent.com/st3v3nmw/beacon-dns-lists/main/allowlists/video-streaming",
			Categories: []types.Category{types.CategoryVideoStreaming},
			Action:     types.ActionAllow,
			Format:     types.SourceFormatDomains,
		},
		// gambling
		{
			Name:       "beacon-dns-lists:allowlists:gambling",
			URL:        "https://raw.githubusercontent.com/st3v3nmw/beacon-dns-lists/main/allowlists/gambling",
			Categories: []types.Category{types.CategoryGambling},
			Action:     types.ActionAllow,
			Format:     types.SourceFormatDomains,
		},
		// gaming
		{
			Name:       "beacon-dns-lists:allowlists:gaming",
			URL:        "https://raw.githubusercontent.com/st3v3nmw/beacon-dns-lists/main/allowlists/gaming",
			Categories: []types.Category{types.CategoryGaming},
			Action:     types.ActionAllow,
			Format:     types.SourceFormatDomains,
		},
		// piracy, torrents
		{
			Name:       "beacon-dns-lists:allowlists:piracy",
			URL:        "https://raw.githubusercontent.com/st3v3nmw/beacon-dns-lists/main/allowlists/piracy",
			Categories: []types.Category{types.CategoryPiracy},
			Action:     types.ActionAllow,
			Format:     types.SourceFormatDomains,
		},
		// drugs
		{
			Name:       "beacon-dns-lists:allowlists:drugs",
			URL:        "https://raw.githubusercontent.com/st3v3nmw/beacon-dns-lists/main/allowlists/drugs",
			Categories: []types.Category{types.CategoryDrugs},
			Action:     types.ActionAllow,
			Format:     types.SourceFormatDomains,
		},
	}

	return append(blocklists, allowlists...)
}
