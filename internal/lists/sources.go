package lists

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/st3v3nmw/beacon/internal/models"
)

type SourceFormat string

const (
	SourceFormatDomains SourceFormat = "domains"
	SourceFormatHosts   SourceFormat = "hosts"
)

type Source struct {
	Name     string          `json:"name"`
	URL      string          `json:"url"`
	Action   models.Action   `json:"action"`
	Category models.Category `json:"category"`
	LastSync time.Time       `json:"last_sync"`
	Domains  []string        `json:"domains"`
	Format   SourceFormat    `json:"-"`
}

func (s *Source) path() string {
	return fmt.Sprintf("%s/%s.json", DataDir, s.Name)
}

func (s *Source) existsOnFs() bool {
	_, err := os.Stat(s.path())
	return err == nil
}

func (s *Source) readFromFs() error {
	data, err := os.ReadFile(s.path())
	if err != nil {
		return err
	}

	return json.Unmarshal(data, &s)
}

func (s *Source) saveToFs() error {
	data, err := json.MarshalIndent(s, "", " ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.path(), data, 0755)
}

func (s *Source) fetchFromUpstream() error {
	resp, err := http.Get(s.URL)
	if err != nil {
		return fmt.Errorf("failed to fetch source: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %v", err)
	}

	s.LastSync = time.Now().UTC()
	s.Domains = s.parseDomains(body)

	return nil
}

func (s *Source) parseDomains(data []byte) []string {
	content := string(data)
	lines := strings.Split(content, "\n")

	domains := make([]string, 0, len(lines))
	validate := validator.New(validator.WithRequiredStructEnabled())
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		var domain string
		if s.Format == SourceFormatDomains {
			domain = line
		} else {
			domain = strings.Fields(line)[1]
		}

		if err := validate.Var(domain, "fqdn"); err == nil {
			domains = append(domains, domain)
		}
	}

	return domains
}

// Get the blocklist & allowlist sources
func getSources() []Source {
	// Blocklists
	blocklists := []Source{
		// ads, trackers
		{

			Name:     "olbat:ut1-blacklists:publicite",
			URL:      "https://raw.githubusercontent.com/olbat/ut1-blacklists/master/blacklists/publicite/domains",
			Category: models.CategoryAds,
			Action:   models.ActionBlock,
			Format:   SourceFormatDomains,
		},
		{

			Name:     "peter-lowe:adservers",
			URL:      "https://pgl.yoyo.org/adservers/serverlist.php?hostformat=hosts",
			Category: models.CategoryAds,
			Action:   models.ActionBlock,
			Format:   SourceFormatHosts,
		},
		{

			Name:     "firebog:easy-privacy",
			URL:      "https://v.firebog.net/hosts/Easyprivacy.txt",
			Category: models.CategoryAds,
			Action:   models.ActionBlock,
			Format:   SourceFormatDomains,
		},
		// malware, ransomware, phishing, cryptojacking, stalkerware
		{

			Name:     "olbat:ut1-blacklists:malware",
			URL:      "https://raw.githubusercontent.com/olbat/ut1-blacklists/master/blacklists/malware/domains",
			Category: models.CategoryMalware,
			Action:   models.ActionBlock,
			Format:   SourceFormatDomains,
		},
		{

			Name:     "olbat:ut1-blacklists:phishing",
			URL:      "https://raw.githubusercontent.com/olbat/ut1-blacklists/master/blacklists/phishing/domains",
			Category: models.CategoryMalware,
			Action:   models.ActionBlock,
			Format:   SourceFormatDomains,
		},
		{

			Name:     "olbat:ut1-blacklists:cryptojacking",
			URL:      "https://raw.githubusercontent.com/olbat/ut1-blacklists/master/blacklists/cryptojacking/domains",
			Category: models.CategoryMalware,
			Action:   models.ActionBlock,
			Format:   SourceFormatDomains,
		},
		// adult content
		{

			Name:     "sinfonietta:hostfiles:pornography-hosts",
			URL:      "https://raw.githubusercontent.com/Sinfonietta/hostfiles/master/pornography-hosts",
			Category: models.CategoryAdult,
			Action:   models.ActionBlock,
			Format:   SourceFormatHosts,
		},
		{

			Name:     "steven-black:hosts:porn-only",
			URL:      "https://raw.githubusercontent.com/StevenBlack/hosts/master/alternates/porn-only/hosts",
			Category: models.CategoryAdult,
			Action:   models.ActionBlock,
			Format:   SourceFormatHosts,
		},
		// dating
		{

			Name:     "olbat:ut1-blacklists:dating",
			URL:      "https://raw.githubusercontent.com/olbat/ut1-blacklists/master/blacklists/dating/domains",
			Category: models.CategoryDating,
			Action:   models.ActionBlock,
			Format:   SourceFormatDomains,
		},
		// social media
		{

			Name:     "olbat:ut1-blacklists:social_networks",
			URL:      "https://raw.githubusercontent.com/olbat/ut1-blacklists/master/blacklists/social_networks/domains",
			Category: models.CategorySocialMedia,
			Action:   models.ActionBlock,
			Format:   SourceFormatDomains,
		},
		// video streaming platforms
		{

			Name:     "beacon-dns-lists:blocklists:video-streaming",
			URL:      "https://raw.githubusercontent.com/st3v3nmw/beacon-dns-lists/main/blocklists/video-streaming",
			Category: models.CategoryVideoStreaming,
			Action:   models.ActionBlock,
			Format:   SourceFormatDomains,
		},
		// gambling
		{

			Name:     "olbat:ut1-blacklists:gambling",
			URL:      "https://raw.githubusercontent.com/olbat/ut1-blacklists/master/blacklists/gambling/domains",
			Category: models.CategoryGambling,
			Action:   models.ActionBlock,
			Format:   SourceFormatDomains,
		},
		{

			Name:     "sinfonietta:hostfiles:gambling-hosts",
			URL:      "https://raw.githubusercontent.com/Sinfonietta/hostfiles/master/gambling-hosts",
			Category: models.CategoryGambling,
			Action:   models.ActionBlock,
			Format:   SourceFormatHosts,
		},
		// gaming
		{

			Name:     "olbat:ut1-blacklists:games",
			URL:      "https://raw.githubusercontent.com/olbat/ut1-blacklists/master/blacklists/games/domains",
			Category: models.CategoryGaming,
			Action:   models.ActionBlock,
			Format:   SourceFormatDomains,
		},
		// piracy, torrents
		{

			Name:     "hagezi:dns-blocklists:anti.piracy-onlydomains",
			URL:      "https://raw.githubusercontent.com/hagezi/dns-blocklists/main/wildcard/anti.piracy-onlydomains.txt",
			Category: models.CategoryPiracy,
			Action:   models.ActionBlock,
			Format:   SourceFormatDomains,
		},
		// drugs
		{

			Name:     "olbat:ut1-blacklists:drugs",
			URL:      "https://raw.githubusercontent.com/olbat/ut1-blacklists/master/blacklists/drogue/domains",
			Category: models.CategoryDrugs,
			Action:   models.ActionBlock,
			Format:   SourceFormatDomains,
		},
	}

	// Allowlists have higher precedence than blocklists
	// We primarily use blocklists as filters and allowlists to
	// remove false positives in a category
	allowlists := []Source{
		// ads, trackers
		{

			Name:     "beacon-dns-lists:allowlists:ads",
			URL:      "https://raw.githubusercontent.com/st3v3nmw/beacon-dns-lists/main/allowlists/ads",
			Category: models.CategoryAds,
			Action:   models.ActionAllow,
			Format:   SourceFormatDomains,
		},
		// malware, ransomware, phishing, cryptojacking, stalkerware
		{

			Name:     "beacon-dns-lists:allowlists:malware",
			URL:      "https://raw.githubusercontent.com/st3v3nmw/beacon-dns-lists/main/allowlists/malware",
			Category: models.CategoryMalware,
			Action:   models.ActionAllow,
			Format:   SourceFormatDomains,
		},
		// adult content
		{

			Name:     "beacon-dns-lists:allowlists:adult",
			URL:      "https://raw.githubusercontent.com/st3v3nmw/beacon-dns-lists/main/allowlists/adult",
			Category: models.CategoryAdult,
			Action:   models.ActionAllow,
			Format:   SourceFormatDomains,
		},
		// dating
		{

			Name:     "beacon-dns-lists:allowlists:dating",
			URL:      "https://raw.githubusercontent.com/st3v3nmw/beacon-dns-lists/main/allowlists/dating",
			Category: models.CategoryDating,
			Action:   models.ActionAllow,
			Format:   SourceFormatDomains,
		},
		// social media
		{

			Name:     "beacon-dns-lists:allowlists:social-media",
			URL:      "https://raw.githubusercontent.com/st3v3nmw/beacon-dns-lists/main/allowlists/social-media",
			Category: models.CategorySocialMedia,
			Action:   models.ActionAllow,
			Format:   SourceFormatDomains,
		},
		// video streaming platforms
		{

			Name:     "beacon-dns-lists:allowlists:video-streaming",
			URL:      "https://raw.githubusercontent.com/st3v3nmw/beacon-dns-lists/main/allowlists/video-streaming",
			Category: models.CategoryVideoStreaming,
			Action:   models.ActionAllow,
			Format:   SourceFormatDomains,
		},
		// gambling
		{

			Name:     "beacon-dns-lists:allowlists:gambling",
			URL:      "https://raw.githubusercontent.com/st3v3nmw/beacon-dns-lists/main/allowlists/gambling",
			Category: models.CategoryGambling,
			Action:   models.ActionAllow,
			Format:   SourceFormatDomains,
		},
		// gaming
		{

			Name:     "beacon-dns-lists:allowlists:gaming",
			URL:      "https://raw.githubusercontent.com/st3v3nmw/beacon-dns-lists/main/allowlists/gaming",
			Category: models.CategoryGaming,
			Action:   models.ActionAllow,
			Format:   SourceFormatDomains,
		},
		// piracy, torrents
		{

			Name:     "beacon-dns-lists:allowlists:piracy",
			URL:      "https://raw.githubusercontent.com/st3v3nmw/beacon-dns-lists/main/allowlists/piracy",
			Category: models.CategoryPiracy,
			Action:   models.ActionAllow,
			Format:   SourceFormatDomains,
		},
		// drugs
		{

			Name:     "beacon-dns-lists:allowlists:drugs",
			URL:      "https://raw.githubusercontent.com/st3v3nmw/beacon-dns-lists/main/allowlists/drugs",
			Category: models.CategoryDrugs,
			Action:   models.ActionAllow,
			Format:   SourceFormatDomains,
		},
	}

	return append(blocklists, allowlists...)
}
