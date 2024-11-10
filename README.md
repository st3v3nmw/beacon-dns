# Beacon DNS

Beacon DNS is a recursive DNS resolver with customizable filtering for malware, trackers, ads, and unwanted content.

The project is in beta and evolving fast. While core functionality is working, I'm working towards full DNS RFC compliance.

## Features

- [x] UDP 53
- [x] Filtering
- [x] Caching
- [x] DNS over HTTP (DoH)
- [ ] DNS over TLS (DoT)
- [ ] Web Interface
- [ ] Global Network
- [ ] Safe Search
- [ ] DNSSEC Validation
- [x] Private! IPs & DNS queries are NOT logged & accounts are not required.

- rate limiting
- rotate providers

## Usage

should be a set & forget...

creating a filter

common filters..

how dns level ad blocking works...

checking why a certain domain was blocked...

### Advanced

automations? how to set dns on schedule

## Architecture

... learning project (built on my free time)

### Internals

### Nodes

### Network

- [ ] Frankfurt
- [ ] Johannesburg
- [ ] Los Angeles
- [ ] Sydney
- [ ] New York
- [ ] Tokyo
- [ ] São Paulo
- [ ] Singapore
- [ ] Dubai
- [ ] Mumbai
- [ ] London
- [ ] Nairobi
- [ ] Santiago
- [ ] Lagos
- [ ] Hong Kong
- [ ] Miami
- [ ] Stockholm
- [ ] Seattle
- [ ] Madrid
- [ ] Istanbul

### Ideal

## Testing

Clone... commands to run...

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

```console
$ linuxkit build --format iso-efi image.yml
```

## Contributing

Code, issues, reporting false positives...

## Support

## Credits

### Libraries

### Blocklists
