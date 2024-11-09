package lists

import (
	"strings"

	"github.com/go-playground/validator/v10"
)

type SourceFormat string

const (
	SourceFormatDomains SourceFormat = "domains"
	SourceFormatHosts   SourceFormat = "hosts"
)

type Source struct {
	List
	Format SourceFormat
}

func parseDomains(data []byte, format SourceFormat) []string {
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
		if format == SourceFormatDomains {
			domain = line
		} else {

			domain = strings.Fields(line)[1]
		}

		err := validate.Var(domain, "fqdn")
		if err == nil {
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
			List: List{
				Name:        "olbat:ut1-blacklists:publicite",
				Description: "Collection of websites blacklists managed by the Université Toulouse Capitole",
				URL:         "https://raw.githubusercontent.com/olbat/ut1-blacklists/master/blacklists/publicite/domains",
				Category:    CategoryAds,
				Action:      ActionBlock,
			},
			Format: SourceFormatDomains,
		},
		{
			List: List{
				Name:        "peter-lowe:adservers",
				Description: "Blocklist for use with hosts files to block ads, trackers, and other nasty things",
				URL:         "https://pgl.yoyo.org/adservers/serverlist.php?hostformat=hosts",
				Category:    CategoryAds,
				Action:      ActionBlock,
			},
			Format: SourceFormatHosts,
		},
		{
			List: List{
				Name:        "firebog:Easyprivacy",
				Description: "Block tracking and improve end user privacy",
				URL:         "https://v.firebog.net/hosts/Easyprivacy.txt",
				Category:    CategoryAds,
				Action:      ActionBlock,
			},
			Format: SourceFormatDomains,
		},
		// malware, ransomware, phishing, cryptojacking, stalkerware
		{
			List: List{
				Name:        "olbat:ut1-blacklists:malware",
				Description: "Collection of websites blacklists managed by the Université Toulouse Capitole",
				URL:         "https://raw.githubusercontent.com/olbat/ut1-blacklists/master/blacklists/malware/domains",
				Category:    CategoryMalware,
				Action:      ActionBlock,
			},
			Format: SourceFormatDomains,
		},
		{
			List: List{
				Name:        "olbat:ut1-blacklists:phishing",
				Description: "Collection of websites blacklists managed by the Université Toulouse Capitole",
				URL:         "https://raw.githubusercontent.com/olbat/ut1-blacklists/master/blacklists/phishing/domains",
				Category:    CategoryMalware,
				Action:      ActionBlock,
			},
			Format: SourceFormatDomains,
		},
		{
			List: List{
				Name:        "olbat:ut1-blacklists:cryptojacking",
				Description: "Collection of websites blacklists managed by the Université Toulouse Capitole",
				URL:         "https://raw.githubusercontent.com/olbat/ut1-blacklists/master/blacklists/cryptojacking/domains",
				Category:    CategoryMalware,
				Action:      ActionBlock,
			},
			Format: SourceFormatDomains,
		},
		// adult content
		{
			List: List{
				Name:        "sinfonietta:hostfiles:pornography-hosts",
				Description: "A collection of category-specific host files",
				URL:         "https://raw.githubusercontent.com/Sinfonietta/hostfiles/master/pornography-hosts",
				Category:    CategoryAdult,
				Action:      ActionBlock,
			},
			Format: SourceFormatHosts,
		},
		{
			List: List{
				Name:        "steven-black:hosts:porn-only",
				Description: "Consolidating and extending hosts files from several well-curated sources",
				URL:         "https://raw.githubusercontent.com/StevenBlack/hosts/master/alternates/porn-only/hosts",
				Category:    CategoryAdult,
				Action:      ActionBlock,
			},
			Format: SourceFormatHosts,
		},
		// dating
		{
			List: List{
				Name:        "olbat:ut1-blacklists:dating",
				Description: "Collection of websites blacklists managed by the Université Toulouse Capitole",
				URL:         "https://raw.githubusercontent.com/olbat/ut1-blacklists/master/blacklists/dating/domains",
				Category:    CategoryDating,
				Action:      ActionBlock,
			},
			Format: SourceFormatDomains,
		},
		// social media
		{
			List: List{
				Name:        "olbat:ut1-blacklists:social_networks",
				Description: "Collection of websites blacklists managed by the Université Toulouse Capitole",
				URL:         "https://raw.githubusercontent.com/olbat/ut1-blacklists/master/blacklists/social_networks/domains",
				Category:    CategorySocialMedia,
				Action:      ActionBlock,
			},
			Format: SourceFormatDomains,
		},
		// video streaming platforms
		// gambling
		{
			List: List{
				Name:        "olbat:ut1-blacklists:gambling",
				Description: "Collection of websites blacklists managed by the Université Toulouse Capitole",
				URL:         "https://raw.githubusercontent.com/olbat/ut1-blacklists/master/blacklists/gambling/domains",
				Category:    CategoryGambling,
				Action:      ActionBlock,
			},
			Format: SourceFormatDomains,
		},
		{
			List: List{
				Name:        "sinfonietta:hostfiles:gambling-hosts",
				Description: "A collection of category-specific host files",
				URL:         "https://raw.githubusercontent.com/Sinfonietta/hostfiles/master/gambling-hosts",
				Category:    CategoryGambling,
				Action:      ActionBlock,
			},
			Format: SourceFormatHosts,
		},
		// gaming
		{
			List: List{
				Name:        "olbat:ut1-blacklists:games",
				Description: "Collection of websites blacklists managed by the Université Toulouse Capitole",
				URL:         "https://raw.githubusercontent.com/olbat/ut1-blacklists/master/blacklists/games/domains",
				Category:    CategoryGaming,
				Action:      ActionBlock,
			},
			Format: SourceFormatDomains,
		},
		// piracy, torrents
		{
			List: List{
				Name:        "hagezi:dns-blocklists:anti.piracy-onlydomains",
				Description: "DNS-Blocklists: For a better internet - keep the internet clean!",
				URL:         "https://raw.githubusercontent.com/hagezi/dns-blocklists/main/wildcard/anti.piracy-onlydomains.txt",
				Category:    CategoryPiracy,
				Action:      ActionBlock,
			},
			Format: SourceFormatDomains,
		},
		// drugs
		{
			List: List{
				Name:        "olbat:ut1-blacklists:drugs",
				Description: "Collection of websites blacklists managed by the Université Toulouse Capitole",
				URL:         "https://raw.githubusercontent.com/olbat/ut1-blacklists/master/blacklists/drogue/domains",
				Category:    CategoryDrugs,
				Action:      ActionBlock,
			},
			Format: SourceFormatDomains,
		},
	}

	// Allowlists have higher precedence than blocklists
	// We primarily use blocklists as filters and allowlists to
	// remove false positives in a category
	// There should be only one allowlist per category
	allowlists := []Source{
		// ads, trackers
		// malware, ransomware, phishing, cryptojacking, stalkerware
		// adult content
		// dating
		// social media
		// video streaming platforms
		// gambling
		// gaming
		// piracy, torrents
		// drugs
	}

	return append(blocklists, allowlists...)
}
