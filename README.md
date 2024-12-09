<div align="center">
    <img src="docs/media/logo.png" width="100" />
    <h1>Beacon DNS</h1>
    <p><i>Runs on a single vCPU, a small hill of RAM, and pure determination.</i></p>
</div>

A DNS resolver with customizable & schedulable filtering for malware, trackers, ads, and unwanted content.

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

### Logo

<a href="https://www.flaticon.com/free-icons/lighthouse" title="lighthouse icons">Logo created by Freepik - Flaticon</a>
