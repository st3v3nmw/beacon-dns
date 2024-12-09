package types

import "sync"

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
)

type ClientLookupMethod string

const (
	// TODO: Add options for DHCP lease files or other rDNS
	ClientLookupTailscale ClientLookupMethod = "tailscale"
)

type ThreadSafeSlice[T any] struct {
	sync.RWMutex
	items []T
}

func (s *ThreadSafeSlice[T]) Append(item T) {
	s.Lock()
	defer s.Unlock()
	s.items = append(s.items, item)
}

func (s *ThreadSafeSlice[T]) Len() int {
	s.RLock()
	defer s.RUnlock()
	return len(s.items)
}

func (s *ThreadSafeSlice[T]) Iterator() <-chan T {
	ch := make(chan T)
	go func() {
		s.RLock()
		defer s.RUnlock()

		for _, item := range s.items {
			ch <- item
		}
		close(ch)
	}()
	return ch
}

func (s *ThreadSafeSlice[T]) Clear() {
	s.Lock()
	defer s.Unlock()
	s.items = s.items[:0]
}
