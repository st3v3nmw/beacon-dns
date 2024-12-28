package config

import (
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/st3v3nmw/beacon/internal/types"
)

var (
	All      Config
	Location *time.Location
)

type DurationValue struct {
	time.Duration
}

func (d *DurationValue) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}

	// Parse day-based duration
	if len(s) > 0 && s[len(s)-1] == 'd' {
		days, err := strconv.Atoi(s[:len(s)-1])
		if err != nil {
			return fmt.Errorf("invalid day duration: %v", err)
		}
		d.Duration = time.Duration(days) * 24 * time.Hour
		return nil
	}

	// Fall back to standard time.ParseDuration
	parsed, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	d.Duration = parsed
	return nil
}

type Config struct {
	System       *SystemConfig              `yaml:"system" json:"system"`
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

type SystemConfig struct {
	Timezone string `yaml:"timezone" json:"timezone"`
}

type DNSConfig struct {
	Port      uint16   `yaml:"port" json:"port"`
	Upstreams []string `yaml:"upstreams" json:"upstreams"`
}

type CacheConfig struct {
	Capacity      int                       `yaml:"capacity" json:"capacity"`
	ServeStale    *CacheServeStaleConfig    `yaml:"serve_stale" json:"serve_stale"`
	QueryPatterns *CacheQueryPatternsConfig `yaml:"query_patterns" json:"query_patterns"`
}

type CacheServeStaleConfig struct {
	For     DurationValue `yaml:"for" json:"for"`
	WithTTL DurationValue `yaml:"with_ttl" json:"with_ttl"`
}

type CacheQueryPatternsConfig struct {
	Follow   bool          `yaml:"follow" json:"follow"`
	LookBack DurationValue `yaml:"look_back" json:"look_back"`
}

type APIConfig struct {
	Port uint16 `yaml:"port" json:"port"`
}

type ClientLookupConfig struct {
	Upstream     string                   `yaml:"upstream" json:"upstream"`
	Method       types.ClientLookupMethod `yaml:"method" json:"method"`
	Clients      map[string]string        `yaml:"clients" json:"clients,omitempty"`
	RefreshEvery DurationValue            `yaml:"refresh_every" json:"refresh_every"`
}

type GroupConfig struct {
	Devices    []string         `yaml:"devices" json:"devices,omitempty"`
	Block      []types.Category `yaml:"block" json:"block,omitempty"`
	SafeSearch bool             `yaml:"safe_search" json:"safe_search"`

	// For quick lookups
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

	// For quick lookups
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
	Periods []*SchedulePeriod `yaml:"periods" json:"periods,omitempty"`
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

	// For quick lookups
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
	Enabled    bool          `yaml:"enabled" json:"enabled"`
	LogClients bool          `yaml:"log_clients" json:"log_clients"`
	Retention  DurationValue `yaml:"retention" json:"retention"`
}

type DHCPConfig struct {
	Enabled bool `yaml:"enabled" json:"enabled"`
}

type SourcesConfig struct {
	UpdateInterval DurationValue      `yaml:"update_interval" json:"update_interval"`
	Lists          []SourceListConfig `yaml:"lists" json:"lists"`
}

type SourceListConfig struct {
	Name     string             `yaml:"name" json:"name"`
	URL      string             `yaml:"url" json:"url"`
	Category types.Category     `yaml:"category" json:"category"`
	Action   types.Action       `yaml:"action" json:"action"`
	Format   types.SourceFormat `yaml:"format" json:"format"`
}

func Read(filePath string) error {
	file, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	// Set defaults
	All.System = &SystemConfig{
		Timezone: time.Now().Location().String(),
	}

	All.DNS = &DNSConfig{
		Port:      53,
		Upstreams: []string{"1.1.1.1", "8.8.8.8"},
	}

	All.Cache = &CacheConfig{Capacity: 1_000}
	All.Cache.ServeStale = &CacheServeStaleConfig{
		For:     DurationValue{5 * time.Minute},
		WithTTL: DurationValue{15 * time.Second},
	}
	All.Cache.QueryPatterns = &CacheQueryPatternsConfig{
		Follow:   true,
		LookBack: DurationValue{14 * 24 * time.Hour},
	}

	All.API = &APIConfig{Port: 80}

	All.ClientLookup = &ClientLookupConfig{
		RefreshEvery: DurationValue{1 * time.Hour},
	}

	allDevicesGroup := &GroupConfig{
		Block: []types.Category{types.CategoryAds, types.CategoryMalware},
	}
	All.Groups = map[string]*GroupConfig{"all": allDevicesGroup}

	All.QueryLog = &QueryLogConfig{
		Enabled:    true,
		LogClients: true,
		Retention:  DurationValue{90 * 24 * time.Hour},
	}

	oneDay := DurationValue{24 * time.Hour}
	All.Sources = &SourcesConfig{
		UpdateInterval: oneDay,
		Lists:          getDefaultSources(),
	}

	err = yaml.Unmarshal(file, &All)
	if err != nil {
		return err
	}

	// Enforce minimums
	if All.Cache.QueryPatterns.LookBack.Duration < oneDay.Duration {
		All.Cache.QueryPatterns.LookBack = oneDay
	}

	if All.Sources.UpdateInterval.Duration < oneDay.Duration {
		All.Sources.UpdateInterval = oneDay
	}

	// Precompute
	All.precompute()

	Location, err = time.LoadLocation(All.System.Timezone)
	return err
}
