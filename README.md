# Beacon DNS

Beacon DNS is a recursive DNS resolver with customizable filtering for malware, trackers, ads, and unwanted content.

## Features

- [x] UDP 53
- [x] Filtering
- [x] Caching
- [ ] Call upstream resolvers over DNS over TLS (DoT)
- [ ] CLI Interface
- [ ] Global Network
- [ ] Safe Search
- [ ] DNSSEC Validation
- [ ] Schedules
- [ ] DHCP

## Usage

should be a set & forget...

creating a filter

common filters..

how dns level ad blocking works...

checking why a certain domain was blocked... (trace)

## Testing

Clone... commands to run...

websocat

### Linux

```console
$ sudoedit /etc/systemd/resolved.conf
[Resolve]
DNS=127.0.0.1
FallbackDNS=1.1.1.1
```

```console
$ sudo iptables -t nat -A OUTPUT -p udp -d 127.0.0.1 --dport 53 -j DNAT --to-destination 127.0.0.1:2053
```

```console
$ sudo systemctl restart systemd-resolved
```

## Building

```console
$ docker build .
```

## Deploying

--> move to separate doc

```console
$ IP="<ip>" USER="<user>" make deploy
```

## Contributing

Code, issues, reporting false positives...

## Support

## Credits

### Libraries

### Blocklists
