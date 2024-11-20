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

func (c *Config) Trace(cat types.Category) (map[string]*GroupConfig, map[string]*ScheduleConfig) {
	groups := map[string]*GroupConfig{}
	for name, group := range c.Groups {
		if group.block[cat] {
			groups[name] = group
		}
	}

	schedules := map[string]*ScheduleConfig{}
	for name, sched := range c.Schedules {
		if sched.block[cat] {
			schedules[name] = sched
		}
	}

	return groups, schedules
}

func (c *Config) IsClientBlocked(client string, category types.Category) bool {
	// Check all groups
	groups := map[string]bool{}
	for name, group := range c.Groups {
		clientInGroup := len(group.Devices) == 0 || group.devices[client]
		if clientInGroup {
			groups[name] = true
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
					if minutes >= period.start && minutes < period.end {
						return true
					}
				} else {
					// Spans midnight
					if minutes >= period.start || minutes < period.end {
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
	Timezone  string   `yaml:"timezone" json:"timezone"`
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
	Devices    []string         `yaml:"devices" json:"devices,omitempty"`
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
