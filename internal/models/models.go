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
	// advertising, marketing, trackers, analytics
	CategoryAds Category = "ads"
	// malware, phishing, exploits
	CategoryMalware Category = "malware"
	// social media
	CategorySocialMedia Category = "social-media"
	// video & media platforms
	CategoryStreaming Category = "streaming"
	// adult content
	CategoryAdult Category = "adult"
	// user-defined
	CategoryCustom Category = "custom"
)

const (
	Sunday    uint8 = 1 << 0
	Monday    uint8 = 1 << 1
	Tuesday   uint8 = 1 << 2
	Wednesday uint8 = 1 << 3
	Thursday  uint8 = 1 << 4
	Friday    uint8 = 1 << 5
	Saturday  uint8 = 1 << 6
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

// Schedules

type Schedule struct {
	BaseModel
	Name    string   `json:"name" validate:"required"`
	Action  Action   `json:"action" validate:"required"`
	Enabled bool     `json:"enabled" validate:"required"`
	Timings []Timing `json:"timings"`
	Lists   []List   `gorm:"many2many:schedule_lists;" json:"lists"`
}

// Timings

type Timing struct {
	BaseModel
	Start      time.Duration `json:"start" validate:"required"` // time since midnight
	End        time.Duration `json:"end" validate:"required"`   // time since midnight
	Days       uint8         `json:"days" validate:"required"`
	ScheduleID ulid.ULID     `json:"schedule_id" validate:"required"`
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
