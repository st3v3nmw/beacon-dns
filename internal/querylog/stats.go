package querylog

import (
	"encoding/json"
	"time"
)

var (
	sqliteTimestampLayout = "2006-01-02 15:04:05.999999999-07:00"
)

const getDeviceStatsQuery = `
SELECT
    hostname,
    ip,
    COUNT(*) as total_queries,
    COUNT(DISTINCT domain) as unique_domains,

    -- Cache stats
    SUM(CASE WHEN cached THEN 1 ELSE 0 END) as cached_queries,
    ROUND(SUM(CASE WHEN cached THEN 1 ELSE 0 END) * 100.0 / COUNT(*), 2) as cache_hit_ratio,

    -- Blocking stats
    SUM(CASE WHEN blocked THEN 1 ELSE 0 END) as blocked_queries,
    ROUND(SUM(CASE WHEN blocked THEN 1 ELSE 0 END) * 100.0 / COUNT(*), 2) as block_ratio,

    -- Performance
    ROUND(AVG(response_time_ms), 2) as avg_response_time_ms,
    ROUND(COALESCE(AVG(CASE WHEN NOT cached THEN response_time_ms END), 0), 2) as avg_uncached_response_time_ms,
    MIN(response_time_ms) as min_response_time_ms,
    MAX(response_time_ms) as max_response_time_ms,

    -- Query types distribution
    (
        SELECT json_group_object(query_type, cnt)
        FROM (
            SELECT query_type, COUNT(*) as cnt
            FROM queries q2
            WHERE q2.hostname = q.hostname
            GROUP BY query_type
            ORDER BY cnt DESC
        )
    ) as query_types,

    -- Block reasons
    (
        SELECT json_group_object(block_reason, cnt)
        FROM (
            SELECT block_reason, COUNT(*) as cnt
            FROM queries q2
            WHERE q2.hostname = q.hostname AND block_reason IS NOT NULL
            GROUP BY block_reason
            ORDER BY cnt DESC
        )
    ) as block_reasons,

    -- Upstream distribution
    (
        SELECT json_group_object(upstream, cnt)
        FROM (
            SELECT upstream, COUNT(*) as cnt
            FROM queries q2
            WHERE q2.hostname = q.hostname AND upstream IS NOT NULL
            GROUP BY upstream
            ORDER BY cnt DESC
        )
    ) as upstreams,

    -- Top domains
    (
        SELECT json_group_object(domain, cnt)
        FROM (
            SELECT domain, COUNT(*) as cnt
            FROM queries q2
            WHERE q2.hostname = q.hostname AND q2.blocked IS FALSE
            GROUP BY domain
            ORDER BY cnt DESC
            LIMIT 10
        )
    ) as resolved_domains,

    (
        SELECT json_group_object(domain, cnt)
        FROM (
            SELECT domain, COUNT(*) as cnt
            FROM queries q2
            WHERE q2.hostname = q.hostname AND q2.blocked IS TRUE
            GROUP BY domain
            ORDER BY cnt DESC
            LIMIT 10
        )
    ) as blocked_domains,

    -- Response codes
    (
        SELECT json_group_object(response_code, cnt)
        FROM (
            SELECT response_code, COUNT(*) as cnt
            FROM queries q2
            WHERE q2.hostname = q.hostname
            GROUP BY response_code
            ORDER BY cnt DESC
            LIMIT 10
        )
    ) as response_codes,

    -- Time range
    MIN(timestamp) as first_seen,
    MAX(timestamp) as last_seen
FROM queries q
GROUP BY hostname
ORDER BY total_queries DESC;
`

type DeviceStats struct {
	Hostname                  string         `json:"hostname"`
	IP                        string         `json:"ip"`
	TotalQueries              int            `json:"total_queries"`
	UniqueDomains             int            `json:"unique_domains"`
	CachedQueries             int            `json:"cached_queries"`
	CacheHitRatio             float64        `json:"cached_hit_ratio"`
	BlockedQueries            int            `json:"blocked_queries"`
	BlockRatio                float64        `json:"block_ratio"`
	AvgResponseTimeMs         float64        `json:"avg_response_time_ms"`
	AvgUncachedResponseTimeMs float64        `json:"avg_uncached_response_time_ms"`
	MinResponseTimeMs         int            `json:"min_response_time_ms"`
	MaxResponseTimeMs         int            `json:"max_response_time_ms"`
	QueryTypes                map[string]int `json:"query_types"`
	BlockReasons              map[string]int `json:"block_reasons"`
	Upstreams                 map[string]int `json:"upstreams"`
	ResolvedDomains           map[string]int `json:"resolved_domains"`
	BlockedDomains            map[string]int `json:"blocked_domains"`
	ResponseCodes             map[string]int `json:"response_codes"`
	FirstSeen                 time.Time      `json:"first_seen"`
	LastSeen                  time.Time      `json:"last_seen"`
}

func GetDeviceStats() ([]DeviceStats, error) {
	rows, err := DB.Query(getDeviceStatsQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []DeviceStats
	var query_types, block_reasons, upstreams, resolved_domains string
	var blocked_domains, response_codes, first_seen, last_seen string
	for rows.Next() {
		var s DeviceStats
		err := rows.Scan(
			&s.Hostname,
			&s.IP,
			&s.TotalQueries,
			&s.UniqueDomains,
			&s.CachedQueries,
			&s.CacheHitRatio,
			&s.BlockedQueries,
			&s.BlockRatio,
			&s.AvgResponseTimeMs,
			&s.AvgUncachedResponseTimeMs,
			&s.MinResponseTimeMs,
			&s.MaxResponseTimeMs,
			&query_types,
			&block_reasons,
			&upstreams,
			&resolved_domains,
			&blocked_domains,
			&response_codes,
			&first_seen,
			&last_seen,
		)
		if err != nil {
			return nil, err
		}

		json.Unmarshal([]byte(query_types), &s.QueryTypes)
		json.Unmarshal([]byte(block_reasons), &s.BlockReasons)
		json.Unmarshal([]byte(upstreams), &s.Upstreams)
		json.Unmarshal([]byte(resolved_domains), &s.ResolvedDomains)
		json.Unmarshal([]byte(blocked_domains), &s.BlockedDomains)
		json.Unmarshal([]byte(response_codes), &s.ResponseCodes)
		s.FirstSeen, _ = time.Parse(sqliteTimestampLayout, first_seen)
		s.LastSeen, _ = time.Parse(sqliteTimestampLayout, last_seen)
		stats = append(stats, s)
	}

	return stats, nil
}
