# beacon-dns

```console
$ sudo add-apt-repository ppa:dqlite/dev
$ sudo apt update
$ sudo apt install libdqlite-dev
```

```console
$ sudo iptables -t nat -A OUTPUT -p udp -d 127.0.0.1 --dport 53 -j DNAT --to-destination 127.0.0.1:2053
```
