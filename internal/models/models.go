package models

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

// Lists

type List struct {
}
