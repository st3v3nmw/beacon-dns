<div align="center">
    <img src="media/logo.png" width="100" />
    <h1>Beacon DNS</h1>
    <p><i>Runs on a single vCPU, a small hill of RAM, and pure determination.</i></p>
</div>

A DNS resolver with customizable & schedulable filtering for malware, trackers, ads, and other unwanted content.

## Features

- **Blocking**
    - Supports blocking of ads, malware, adult content, dating & social media sites, video streaming platforms, and [other content](https://github.com/st3v3nmw/beacon-dns/blob/main/internal/config/sources.go)
    - Blocking can be done network-wide or per device group
    - Blocking can also be scheduled so that certain content is only blocked at certain times
- **Caching**
    - Supports caching of DNS records for up to the record's TTL which speeds up DNS lookups
    - Supports serving stale DNS records while the record is refreshed in the background
- **Prefetching**
    - "Learns" your query patterns to prefetch subsequent queries before the device makes them
- **Client Lookup**
    - Supports looking up of the client's hostname
- **Statistics**
    - Allows you view statistics per device over a given period of time
- **API**
    - Allows you to get statistics
    - Allows you to watch queries live as they're being made
    - Allows you to get the current config

## Getting Started

Go to [this page](installation.md) to install and configure Beacon DNS.
