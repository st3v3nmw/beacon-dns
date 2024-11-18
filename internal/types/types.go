package types

type Action string

const (
	ActionAllow Action = "allow"
	ActionBlock Action = "block"
)

type Category string

const (
	CategoryAds            Category = "ads"             // ads, trackers
	CategoryMalware        Category = "malware"         // malware, ransomware, phishing, cryptojacking, stalkerware
	CategoryAdult          Category = "adult"           // adult content
	CategoryDating         Category = "dating"          // dating
	CategorySocialMedia    Category = "social-media"    // social media
	CategoryVideoStreaming Category = "video-streaming" // video streaming platforms
	CategoryGambling       Category = "gambling"        // gambling
	CategoryGaming         Category = "gaming"          // gaming
	CategoryPiracy         Category = "piracy"          // piracy, torrents
	CategoryDrugs          Category = "drugs"           // drugs
)

type SourceFormat string

const (
	SourceFormatDomains SourceFormat = "domains"
	SourceFormatHosts   SourceFormat = "hosts"
	// TODO: Parse these
	SourceFormatIps SourceFormat = "ips"
)

type HostnameLookupMethod string

const (
	// TODO: Add options for DHCP lease files or other rDNS
	HostnameLookupTailscale HostnameLookupMethod = "tailscale"
)
