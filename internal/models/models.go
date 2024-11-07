package models

import (
	"time"

	"github.com/lib/pq"
	"github.com/oklog/ulid/v2"
	"gorm.io/gorm"

	"github.com/st3v3nmw/beacon/pkg/ids"
)

type Action string

const (
	ActionAllow Action = "allow"
	ActionBlock Action = "block"
)

type Category string

const (
	// Default = ads + malware
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

type Creatable interface {
	BeforeCreate(tx *gorm.DB) error
}

// base

type BaseModel struct {
	ID ulid.ULID `gorm:"primaryKey" json:"id"`
}

func (b *BaseModel) BeforeCreate(tx *gorm.DB) error {
	id, err := ids.GenerateULID()
	if err != nil {
		return err
	}
	b.ID = id
	return nil
}

// Lists

type List struct {
	BaseModel
	Name        string         `json:"name" validate:"required"`
	Description string         `json:"description" validate:"required"`
	URL         string         `json:"url" validate:"omitempty,http_url"`
	Categories  pq.StringArray `gorm:"type:text[]" json:"categories" validate:"required"`
	Entries     []ListEntry    `json:"entries"`
	LastSync    time.Time      `json:"last_sync"`
}

// List Entries

type ListEntry struct {
	BaseModel
	Domain string    `gorm:"uniqueIndex:idx_domain_list" json:"domain" validate:"fqdn,required"`
	ListID ulid.ULID `gorm:"uniqueIndex:idx_domain_list" json:"list_id" validate:"required"`
}
