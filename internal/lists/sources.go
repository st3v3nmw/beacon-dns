package lists

import (
	"fmt"
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
		} else {
			fmt.Println(strings.Fields(line))
			fmt.Println(err)
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
				Name:        "peter-lowe:adservers",
				Description: "Blocklist for use with hosts files to block ads, trackers, and other nasty things",
				URL:         "https://pgl.yoyo.org/adservers/serverlist.php?hostformat=hosts",
				Category:    CategoryAds,
				Action:      "block",
			},
			Format: SourceFormatHosts,
		},
		// malware, ransomware, phishing, cryptojacking, stalkerware
		{
			List: List{
				Name:        "olbat:ut1-blacklists:cryptojacking",
				Description: "Collection of websites blacklists managed by the Université Toulouse Capitole",
				URL:         "https://raw.githubusercontent.com/olbat/ut1-blacklists/master/blacklists/cryptojacking/domains",
				Category:    CategoryMalware,
				Action:      "block",
			},
			Format: SourceFormatDomains,
		},
		{
			List: List{
				Name:        "olbat:ut1-blacklists:stalkerware",
				Description: "Collection of websites blacklists managed by the Université Toulouse Capitole",
				URL:         "https://raw.githubusercontent.com/olbat/ut1-blacklists/master/blacklists/stalkerware/domains",
				Category:    CategoryMalware,
				Action:      "block",
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
				Action:      "block",
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
				Action:      "block",
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
				Action:      "block",
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
				Action:      "block",
			},
			Format: SourceFormatDomains,
		},
		{
			List: List{
				Name:        "sinfonietta:hostfiles:gambling-hosts",
				Description: "A collection of category-specific host files",
				URL:         "https://raw.githubusercontent.com/Sinfonietta/hostfiles/master/gambling-hosts",
				Category:    CategoryGambling,
				Action:      "block",
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
				Action:      "block",
			},
			Format: SourceFormatDomains,
		},
		// piracy, torrents
		// drugs
		{
			List: List{
				Name:        "olbat:ut1-blacklists:drugs",
				Description: "Collection of websites blacklists managed by the Université Toulouse Capitole",
				URL:         "https://raw.githubusercontent.com/olbat/ut1-blacklists/master/blacklists/drogue/domains",
				Category:    CategoryDrugs,
				Action:      "block",
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
