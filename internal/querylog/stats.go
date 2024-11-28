package querylog

import (
	"encoding/json"
)

const getDeviceStatsQuery = `
SELECT
    hostname as client,
    COUNT(*) as total_queries,
    COUNT(DISTINCT domain) as unique_domains,

    -- Cache stats
    SUM(CASE WHEN cached THEN 1 ELSE 0 END) as cached_queries,
    CASE
        WHEN (COUNT(*) - SUM(CASE WHEN blocked THEN 1 ELSE 0 END)) > 0
        THEN ROUND(SUM(CASE WHEN cached THEN 1 ELSE 0 END) * 100.0 /
            (COUNT(*) - SUM(CASE WHEN blocked THEN 1 ELSE 0 END)), 2)
        ELSE 0
    END as cache_hit_ratio,

    -- Blocking stats
    SUM(CASE WHEN blocked THEN 1 ELSE 0 END) as blocked_queries,
    ROUND(SUM(CASE WHEN blocked THEN 1 ELSE 0 END) * 100.0 / COUNT(*), 2) as block_ratio,

    -- Prefetching stats
    SUM(CASE WHEN prefetched THEN 1 ELSE 0 END) as prefetched_queries,
    CASE
        WHEN (COUNT(*) - SUM(CASE WHEN blocked THEN 1 ELSE 0 END)) > 0
        THEN ROUND(SUM(CASE WHEN prefetched THEN 1 ELSE 0 END) * 100.0 /
            (COUNT(*) - SUM(CASE WHEN blocked THEN 1 ELSE 0 END)), 2)
        ELSE 0
    END as prefetched_ratio,

    -- Performance
    ROUND(AVG(response_time_ms), 2) as avg_response_time_ms,
    ROUND(COALESCE(AVG(CASE WHEN upstream THEN response_time_ms END), 0), 2) as avg_forwarded_response_time_ms,
    MIN(response_time_ms) as min_response_time_ms,
    MAX(response_time_ms) as max_response_time_ms,

    -- Query types distribution
    (
        SELECT json_group_object(query_type, cnt)
        FROM (
            SELECT query_type, COUNT(*) as cnt
            FROM queries q2
            WHERE q2.hostname = q.hostname AND timestamp >= datetime('now', '-2 hours')
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
            WHERE q2.hostname = q.hostname AND block_reason IS NOT NULL AND timestamp >= datetime('now', '-2 hours')
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
            WHERE q2.hostname = q.hostname AND upstream IS NOT NULL AND timestamp >= datetime('now', '-2 hours')
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
            WHERE q2.hostname = q.hostname AND q2.blocked IS FALSE AND timestamp >= datetime('now', '-2 hours')
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
            WHERE q2.hostname = q.hostname AND q2.blocked IS TRUE AND timestamp >= datetime('now', '-2 hours')
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
            WHERE q2.hostname = q.hostname AND timestamp >= datetime('now', '-2 hours')
            GROUP BY response_code
            ORDER BY cnt DESC
            LIMIT 10
        )
    ) as response_codes,

    -- IPs
    (
        SELECT json_group_object(ip, cnt)
        FROM (
            SELECT ip, COUNT(*) as cnt
            FROM queries q2
            WHERE q2.hostname = q.hostname AND timestamp >= datetime('now', '-2 hours')
            GROUP BY ip
            ORDER BY cnt DESC
            LIMIT 10
        )
    ) as ips
FROM queries q
WHERE timestamp >= datetime('now', '-2 hours')
GROUP BY hostname
ORDER BY total_queries DESC;
`

type DeviceStats struct {
	Client                     string         `json:"client"`
	TotalQueries               int            `json:"total_queries"`
	UniqueDomains              int            `json:"unique_domains"`
	CachedQueries              int            `json:"cached_queries"`
	CacheHitRatio              float64        `json:"cache_hit_ratio"`
	BlockedQueries             int            `json:"blocked_queries"`
	BlockRatio                 float64        `json:"block_ratio"`
	PrefetchedQueries          int            `json:"prefetched_queries"`
	PrefetchedRatio            float64        `json:"prefetched_ratio"`
	AvgResponseTimeMs          float64        `json:"avg_response_time_ms"`
	AvgForwardedResponseTimeMs float64        `json:"avg_forwarded_response_time_ms"`
	MinResponseTimeMs          int            `json:"min_response_time_ms"`
	MaxResponseTimeMs          int            `json:"max_response_time_ms"`
	QueryTypes                 map[string]int `json:"query_types"`
	BlockReasons               map[string]int `json:"block_reasons"`
	Upstreams                  map[string]int `json:"upstreams"`
	ResolvedDomains            map[string]int `json:"resolved_domains"`
	BlockedDomains             map[string]int `json:"blocked_domains"`
	ResponseCodes              map[string]int `json:"response_codes"`
	IPs                        map[string]int `json:"ips"`
}

func GetDeviceStats() ([]DeviceStats, error) {
	rows, err := DB.Query(getDeviceStatsQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := []DeviceStats{}
	var query_types, block_reasons, upstreams, resolved_domains string
	var blocked_domains, response_codes, ips string
	for rows.Next() {
		var s DeviceStats
		err := rows.Scan(
			&s.Client,
			&s.TotalQueries,
			&s.UniqueDomains,
			&s.CachedQueries,
			&s.CacheHitRatio,
			&s.BlockedQueries,
			&s.BlockRatio,
			&s.PrefetchedQueries,
			&s.PrefetchedRatio,
			&s.AvgResponseTimeMs,
			&s.AvgForwardedResponseTimeMs,
			&s.MinResponseTimeMs,
			&s.MaxResponseTimeMs,
			&query_types,
			&block_reasons,
			&upstreams,
			&resolved_domains,
			&blocked_domains,
			&response_codes,
			&ips,
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
		json.Unmarshal([]byte(ips), &s.IPs)
		stats = append(stats, s)
	}

	return stats, nil
}
