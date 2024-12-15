<div align="center">
    <img src="docs/media/logo.png" width="100" />
    <h1>Beacon DNS</h1>
    <p><i>Runs on a single vCPU, a small hill of RAM, and pure determination.</i></p>
</div>

A DNS resolver with customizable & schedulable filtering for malware, trackers, ads, and other unwanted content.

## Features

### Blocking

Supports blocking of ads, malware, adult content, dating & social media sites, video streaming platforms, and [other content](https://github.com/st3v3nmw/beacon-dns/blob/main/internal/config/sources.go).

Blocking can be done network-wide or per device group:

```yaml
groups:
  # not specifying devices blocks on the entire network
  all:
    block:
      - ads
      - malware
      - adult

  screens:
    devices:
      - phone
      - laptop
      - tv
```

Blocking can also be scheduled so that certain content is only blocked at certain times:

```yaml
schedules:
  bedtime:
    apply_to:
      - screens
    when:
      - days: ["sun", "mon", "tue", "wed", "thur", "fri", "sat"]
        periods:
        - start: "22:00"
          end: "06:00"
    block:
      - social-media
      - video-streaming
      - gaming
      - dating
```

For now, rDNS queries for private IP ranges that reach the resolver are always blocked.

### Caching

Supports caching of DNS records for up to the record's TTL. This can then be served to other devices in the network thus speeding up DNS lookups.

But, DNS records on the internet use [ridicuously low TTLs](https://blog.apnic.net/2019/11/12/stop-using-ridiculously-low-dns-ttls/). The resolver can be configured to serve stale DNS records while it refreshes/prefetches the record in the background.

Beacon DNS also "learns" your query patterns to prefetch subsequent queries before the device makes them. For instance, when `github.com` is queried, `avatars.githubusercontent.com` & `github.githubassets.com` usually follow. So when the resolver sees `github.com`, it can prefetch the next two before the device queries for them.

```yaml
cache:
  capacity: 10000
  serve_stale:
    for: 5m
    with_ttl: 15s
  query_patterns:
    follow: true
    look_back: 14d
```

### Client Lookup

Supports looking up of the client's hostname either using tailscale's MagicDNS:

```yaml
client_lookup:
  upstream: 100.100.100.100
  method: tailscale
```

Or hardcoded based on the static IPs configured on your router:

```yaml
client_lookup:
  clients:
    192.168.0.102: laptop
    192.168.0.103: phone
```

### Statistics

Beacon DNS stores your queries for a configured retention period:

```yaml
querylog:
  enabled: true
  log_clients: true
  retention: 90d
```

You can watch the querylog live:

```console
$ websocat ws://mars/api/watch\?clients=phone
{"hostname":"phone","ip":"<ip>","domain":"spclient.wg.spotify.com","query_type":"A","cached":false,"blocked":false,"block_reason":null,"upstream":"1.1.1.1","response_code":"NOERROR","response_time_ms":1,"prefetched":false,"timestamp":"2024-12-09T21:06:05.067810278Z"}
{"hostname":"phone","ip":"<ip>","domain":"spclient.wg.spotify.com","query_type":"A","cached":true,"blocked":false,"block_reason":null,"upstream":null,"response_code":"NOERROR","response_time_ms":0,"prefetched":false,"timestamp":"2024-12-09T21:06:05.08479734Z"}
```

The querylog allows us to generate statistics and compute the query patterns:

```console
$ curl -s http://mars/api/stats/devices?last=24h | jq
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
    "avg_response_time_ms": <ms>,
    "avg_forwarded_response_time_ms": <ms>,
    "min_response_time_ms": <ms>,
    "max_response_time_ms": <ms>,
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

### TODO

- [ ] [CLI Interface](https://github.com/st3v3nmw/beaconctl)
- [ ] [Documentation website](https://www.beacondns.org/)
- [ ] Split DNS
- [ ] DHCP Support
- [ ] Call upstream resolvers over:
  - [ ] DNS over TLS (DoT)
  - [ ] DNS over HTTPS (DoH)
- [ ] Safe Search
- [ ] DNSSEC Validation

## Credits

### Logo

<a href="https://www.flaticon.com/free-icons/lighthouse" title="lighthouse icons">Logo created by Freepik - Flaticon</a>

## Support

<a href='https://ko-fi.com/M4M44DEN6' target='_blank'><img height='36' style='border:0px;height:36px;' src='https://cdn.ko-fi.com/cdn/kofi3.png?v=2' border='0' alt='Buy Me a Coffee at ko-fi.com' /></a>
