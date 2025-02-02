## Statistics

### Per Device

```console
$ curl -s http://<server-ip>/api/stats/devices?last=24h | jq
[
 {
    "client": "phone",
    "total_queries": <n>,
    "unique_domains": <n>,
    "cached_queries": <n>,
    "cache_hit_ratio": <%>,
    "blocked_queries": <n>,
    "block_ratio": <%>,
    "prefetched_queries": <n>,
    "prefetched_ratio": <%>,
    "typical_response_time": <ms>,
    "typical_forwarded_response_time": <ms>,
    "min_response_time": <ms>,
    "max_response_time": <ms>,
    "query_types": {
      "A": <n>,
      "AAAA": <n>,
      "HTTPS": <n>
    },
    "block_reasons": {
      "ads": <n>,
      "gaming": <n>,
      "social-media": <n>,
      "video-streaming": <n>
    },
    "upstreams": {
      "1.1.1.1": <n>
    },
    "resolved_domains": {
      "apresolve.spotify.com": <n>,
      "spclient.wg.spotify.com": <n>,
      ...
    },
    "blocked_domains": {
      "app-measurement.com": <n>,
      "incoming.telemetry.mozilla.org": <n>,
      ...
    },
    "response_codes": {
      "NOERROR": <n>,
      "NXDOMAIN": <n>,
      "REFUSED": <n>
    },
    "ips": {
      "<ip>": <n>
    }
 }
]
```

### Cache

```console
$ curl -s http://<server-ip>/api/cache | jq
{
  "hits": <n>,
  "misses": <n>,
  "ratio": <%>,    # hits / (hits + misses) * 100
  "evicted": <n>,  # evicted after expiry OR if the cache size exceeds capacity
  "size": <n>,     # number of records currently in the cache
  "capacity": <n>  # how many records the cache can store when full
}
```

## Querylog

You can watch the querylog live:

```console
$ websocat ws://<server-ip>/api/watch\?clients=phone
{"hostname":"phone","ip":"<ip>","domain":"spclient.wg.spotify.com","query_type":"A","cached":false,"blocked":false,"block_reason":null,"upstream":"1.1.1.1","response_code":"NOERROR","response_time":1,"prefetched":false,"timestamp":"2024-12-09T21:06:05.067810278Z"}
{"hostname":"phone","ip":"<ip>","domain":"spclient.wg.spotify.com","query_type":"A","cached":true,"blocked":false,"block_reason":null,"upstream":null,"response_code":"NOERROR","response_time":0,"prefetched":false,"timestamp":"2024-12-09T21:06:05.08479734Z"}
```

## Config

```console
$ curl -s http://<server-ip>/api/config | jq
{
  ...
}
```
