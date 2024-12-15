package dns

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"slices"
	"sync"
	"time"

	"github.com/maypok86/otter"
	dnslib "github.com/miekg/dns"
	"github.com/st3v3nmw/beacon/internal/config"
	"github.com/st3v3nmw/beacon/internal/querylog"
)

var (
	Cache             otter.CacheWithVariableTTL[string, *Cached]
	serveStaleFor     uint32
	serveStaleWithTTL uint32
	Prefetch          = map[string]map[string][]string{}
	ongoingPrefetches sync.Map
)

func NewCache() (err error) {
	Cache, err = otter.MustBuilder[string, *Cached](config.All.Cache.Capacity).
		CollectStats().
		WithVariableTTL().
		Build()
	if err != nil {
		return err
	}

	serveStaleFor = uint32(config.All.Cache.ServeStale.For.Seconds())
	serveStaleWithTTL = uint32(config.All.Cache.ServeStale.WithTTL.Seconds())

	loadQueryPatterns()

	return nil
}

type Cached struct {
	Msg        *dnslib.Msg
	Touched    time.Time
	Stale      bool
	Prefetched bool
}

func (c *Cached) reduceTtl(rrs []dnslib.RR, elapsed time.Duration) {
	for _, answer := range rrs {
		header := answer.Header()
		if header.Ttl == 0 {
			continue
		}

		elapsed := uint32(elapsed.Seconds())
		if header.Ttl > elapsed {
			header.Ttl -= elapsed
		} else {
			c.Stale = true
			header.Ttl = serveStaleWithTTL
		}
	}
}

func (c *Cached) touch() bool {
	now := time.Now()
	wasStale := c.Stale

	elapsed := now.Sub(c.Touched)
	c.reduceTtl(c.Msg.Answer, elapsed)
	c.reduceTtl(c.Msg.Ns, elapsed)
	c.reduceTtl(c.Msg.Extra, elapsed)
	c.Touched = now

	return !wasStale && c.Stale
}

type CacheStats struct {
	Hits     int64   `json:"hits"`
	Misses   int64   `json:"misses"`
	Ratio    float64 `json:"ratio"`
	Evicted  int64   `json:"evicted"`
	Size     int     `json:"size"`
	Capacity int     `json:"capacity"`
}

func GetCacheStats() CacheStats {
	stats := Cache.Stats()
	return CacheStats{
		Hits:     stats.Hits(),
		Misses:   stats.Misses(),
		Ratio:    math.Round(10_000*stats.Ratio()) / 100,
		Evicted:  stats.EvictedCount(),
		Size:     Cache.Size(),
		Capacity: Cache.Capacity(),
	}
}

type domainStats struct {
	RecordTypes []string
	Count       float64
}

type QueryPattern struct {
	Domain      string              `json:"domain"`
	Occurrences int                 `json:"occurrences"`
	Prefetch    map[string][]string `json:"prefetch"`
}

func fetchQueries() ([]querylog.QueryLog, error) {
	query := `
		SELECT
			hostname,
			domain,
			query_type,
			timestamp
		FROM queries
		WHERE
			blocked IS FALSE
			AND response_code = 'NOERROR'
			AND query_type != 'UNKNOWN'
			AND timestamp >= datetime('now', ?)
		ORDER BY timestamp ASC
	`

	lookback := config.All.Cache.QueryPatterns.LookBack
	offset := fmt.Sprintf("-%d minutes", int(lookback.Minutes()))
	rows, err := querylog.DB.Query(query, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var queries []querylog.QueryLog
	for rows.Next() {
		var query querylog.QueryLog
		var timestamp string
		err = rows.Scan(&query.Hostname, &query.Domain, &query.QueryType, &timestamp)
		if err != nil {
			return nil, err
		}

		query.Timestamp, _ = time.Parse(time.RFC3339Nano, timestamp)
		queries = append(queries, query)
	}
	return queries, nil
}

func binQueries(queries []querylog.QueryLog) map[string]map[string]*domainStats {
	bins := map[string]map[string]*domainStats{}
	for i, lead := range queries {
		observedAfter, ok := bins[lead.Domain]
		if !ok {
			observedAfter = map[string]*domainStats{}
			bins[lead.Domain] = observedAfter
		}

		last := lead.Timestamp.Add(5 * time.Second)
		for _, query := range queries[i:] {
			if query.Hostname != lead.Hostname || query.Domain == lead.Domain {
				continue
			}

			if query.Timestamp.After(last) {
				break
			}

			details, exists := observedAfter[query.Domain]
			if exists {
				details.Count++
				details.RecordTypes = append(details.RecordTypes, query.QueryType)
			} else {
				observedAfter[query.Domain] = &domainStats{
					Count:       1,
					RecordTypes: []string{query.QueryType},
				}
			}
		}
	}
	return bins
}

func findQueryPatterns(bins map[string]map[string]*domainStats) []QueryPattern {
	var result []QueryPattern
	for domain, bin := range bins {
		maxCount := 0.0
		for _, details := range bin {
			if details.Count > maxCount {
				maxCount = details.Count
			}
		}

		if maxCount < 5 {
			continue
		}

		pattern := QueryPattern{
			Domain:      domain,
			Occurrences: int(maxCount),
			Prefetch:    map[string][]string{},
		}

		for relatedDomain, details := range bin {
			slices.Sort(details.RecordTypes)
			recordTypes := slices.Compact(details.RecordTypes)
			if details.Count/maxCount >= 0.8 {
				pattern.Prefetch[relatedDomain] = recordTypes
			}
		}

		result = append(result, pattern)
	}

	slices.SortFunc(result, func(a, b QueryPattern) int {
		return b.Occurrences - a.Occurrences
	})

	return result
}

func UpdateQueryPatterns() error {
	if !config.All.Cache.QueryPatterns.Follow {
		slog.Debug("Following query patterns disabled, skipping...")
		return nil
	}

	slog.Info("Updating query patterns...")

	queries, err := fetchQueries()
	if err != nil {
		return fmt.Errorf("error fetching queries: %w", err)
	}

	bins := binQueries(queries)
	patterns := findQueryPatterns(bins)

	tx, err := querylog.DB.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	_, err = querylog.DB.Exec("DELETE FROM query_patterns")
	if err != nil {
		return fmt.Errorf("failed to truncate table query_patterns: %w", err)
	}

	stmt, err := tx.Prepare(`
		INSERT INTO query_patterns (domain, occurrences, prefetch)
		VALUES ($1, $2, $3)
	`)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, pattern := range patterns {
		prefetch, _ := json.Marshal(pattern.Prefetch)

		_, err := stmt.Exec(pattern.Domain, pattern.Occurrences, prefetch)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to insert query: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	if err := loadQueryPatterns(); err != nil {
		return err
	}

	return nil
}

func loadQueryPatterns() error {
	if !config.All.Cache.QueryPatterns.Follow {
		return nil
	}

	newPrefetch := map[string]map[string][]string{}

	rows, err := querylog.DB.Query("SELECT domain, prefetch FROM query_patterns")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var domain, prefetchJSON string
		err := rows.Scan(&domain, &prefetchJSON)
		if err != nil {
			return err
		}

		var prefetchMap map[string][]string
		err = json.Unmarshal([]byte(prefetchJSON), &prefetchMap)
		if err != nil {
			return err
		}

		newPrefetch[domain] = prefetchMap
	}

	if err = rows.Err(); err != nil {
		return err
	}

	Prefetch = newPrefetch
	return nil
}
