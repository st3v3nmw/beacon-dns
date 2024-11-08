package models

import (
	"time"
)

type Action string

const (
	ActionAllow Action = "allow"
	ActionBlock Action = "block"
)

type Category string

const (
	CategoryAds            Category = "ads"             // ads, trackers
	CategoryMalware        Category = "malware"         // malware, ransomware, phishing, cryptojacking
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
	ID          int
	Name        string
	Description string
	Source      string
	Category    Category
	LastSync    time.Time
}

// List Entries

type ListEntry struct {
	ID         int
	Domain     string
	Action     Action
	IsOverride bool // whether this overrides the upstream list
	ListID     int
}

// Schema

const schema = `
CREATE TABLE IF NOT EXISTS lists (
    id INTEGER PRIMARY KEY,
    name TEXT,
	description TEXT,
    source TEXT,
	category TEXT,
    last_sync INTEGER
);

CREATE TABLE IF NOT EXISTS entries (
	id INTEGER PRIMARY KEY,
    domain TEXT,
    list_id INTEGER,
    action TEXT,
    is_override BOOLEAN,
	FOREIGN KEY(list_id) REFERENCES lists(id)
    UNIQUE(domain, list_id, is_override)
);`
