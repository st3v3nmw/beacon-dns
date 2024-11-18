package config

import (
	"net"
	"os"

	"github.com/goccy/go-yaml"
	"github.com/st3v3nmw/beacon/internal/types"
)

var (
	All Config
)

type Config struct {
	DNS          DNSConfig          `yaml:"dns" json:"dns"`
	Cache        CacheConfig        `yaml:"cache" json:"cache"`
	API          APIConfig          `yaml:"api" json:"api"`
	ClientLookup ClientLookupConfig `yaml:"client_lookup" json:"client_lookup"`
	Groups       []GroupConfig      `yaml:"groups" json:"groups"`
	Schedules    []ScheduleConfig   `yaml:"schedules" json:"schedules"`
	Quotas       []QuotaConfig      `yaml:"quotas" json:"quotas"`
	QueryLog     QueryLogConfig     `yaml:"querylog" json:"querylog"`
	DHCP         DHCPConfig         `yaml:"dhcp" json:"dhcp"`
	Sources      SourcesConfig      `yaml:"sources" json:"sources"`
}

type DNSConfig struct {
	Port      uint16           `yaml:"port" json:"port"`
	Upstreams []string         `yaml:"upstreams" json:"upstreams"`
	Block     []types.Category `yaml:"block" json:"block"`
}

type CacheConfig struct {
	Size int            `yaml:"size" json:"size"`
	TTL  CacheTTLConfig `yaml:"ttl" json:"ttl"`
}

type CacheTTLConfig struct {
	Min string `yaml:"min" json:"min"`
	Max string `yaml:"max" json:"max"`
}

type APIConfig struct {
	Port   uint16   `yaml:"port" json:"port"`
	Tokens []string `yaml:"tokens" json:"tokens"`
}

type ClientLookupConfig struct {
	Upstream string                   `yaml:"upstream" json:"upstream"`
	Method   types.ClientLookupMethod `yaml:"method" json:"method"`
	Clients  map[string]net.IP        `yaml:"clients" json:"clients"`
}

type GroupConfig struct {
	Name       string           `yaml:"name" json:"name"`
	Devices    []string         `yaml:"devices" json:"devices"`
	Block      []types.Category `yaml:"block" json:"block"`
	SafeSearch bool             `yaml:"safe_search" json:"safe_search"`
}

type ScheduleConfig struct {
	Name    string           `yaml:"name" json:"name"`
	ApplyTo []string         `yaml:"apply_to" json:"apply_to"`
	When    []ScheduleWhen   `yaml:"when" json:"when"`
	Block   []types.Category `yaml:"block" json:"block"`
}

type ScheduleWhen struct {
	Days []string       `yaml:"days" json:"days"`
	Time []ScheduleTime `yaml:"time" json:"time"`
}

type ScheduleTime struct {
	Start string `yaml:"start" json:"start"`
	End   string `yaml:"end" json:"end"`
}

type QuotaConfig struct {
	Name       string   `yaml:"name" json:"name"`
	ApplyTo    []string `yaml:"apply_to" json:"apply_to"`
	DailyLimit string   `yaml:"daily_limit" json:"daily_limit"`
	Categories []string `yaml:"categories" json:"categories"`
}

type QueryLogConfig struct {
	Enabled        bool   `yaml:"enabled" json:"enabled"`
	LogClients     bool   `yaml:"log_clients" json:"log_clients"`
	QueryRetention string `yaml:"query_retention" json:"query_retention"`
	StatsRetention string `yaml:"stats_retention" json:"stats_retention"`
}

type LogConfig struct {
	Level string `yaml:"level" json:"level"`
	File  string `yaml:"file" json:"file"`
}

type DHCPConfig struct {
	Enabled bool `yaml:"enabled" json:"enabled"`
}

type SourcesConfig struct {
	UpdateInterval string             `yaml:"update_interval" json:"update_interval"`
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
	All.DNS = DNSConfig{
		Port:      53,
		Upstreams: []string{"1.1.1.1", "8.8.8.8"},
		Block:     []types.Category{types.CategoryAds, types.CategoryMalware},
	}

	All.Cache = CacheConfig{
		Size: 100_000,
		TTL: CacheTTLConfig{
			Min: "15s",
			Max: "24h",
		},
	}

	All.API = APIConfig{Port: 80}

	All.QueryLog = QueryLogConfig{
		Enabled:        true,
		LogClients:     true,
		QueryRetention: "90d",
		StatsRetention: "365d",
	}

	All.Sources.UpdateInterval = "24h"
	All.Sources.Lists = getDefaultSources()

	return yaml.Unmarshal(file, &All)
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
