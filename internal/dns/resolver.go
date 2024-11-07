package dns

import (
	"fmt"
	"strconv"

	dnslib "github.com/miekg/dns"
	"github.com/st3v3nmw/beacon/internal/models"
)

var (
	defaultDNSTTL uint32 = 300
)

type Filter struct {
	Ads            bool
	Malware        bool
	Adult          bool
	Dating         bool
	SocialMedia    bool
	VideoStreaming bool
	Gambling       bool
	Gaming         bool
	Piracy         bool
	Drugs          bool
}

func NewFilterFromStr(filterStr string) (*Filter, error) {
	mask, err := strconv.Atoi(filterStr)
	if err != nil {
		return nil, err
	}

	if mask >= 1024 {
		return nil, fmt.Errorf("filter must be less than 1024")
	}

	return &Filter{
		Ads:            mask&(1<<0) != 0,
		Malware:        mask&(1<<1) != 0,
		Adult:          mask&(1<<2) != 0,
		Dating:         mask&(1<<3) != 0,
		SocialMedia:    mask&(1<<4) != 0,
		VideoStreaming: mask&(1<<5) != 0,
		Gambling:       mask&(1<<6) != 0,
		Gaming:         mask&(1<<7) != 0,
		Piracy:         mask&(1<<8) != 0,
		Drugs:          mask&(1<<9) != 0,
	}, nil
}

func isBlocked(domain string, filter Filter) bool {
	var blocked bool
	err := models.DB.Model(&models.ListEntry{}).
		Select("count(*) > 0").
		Where("domain = ?", domain).
		Find(&blocked).
		Error

	if err != nil {
		// TODO: ERROR!
		return false
	}

	return blocked
}

func forwardToUpstream(r *dnslib.Msg) (*dnslib.Msg, error) {
	c := new(dnslib.Client)
	m, _, err := c.Exchange(r, "1.1.1.1:53")
	return m, err
}
