## Configuration

We'll start by creating a starter configuration file (`config.yml`):

```yaml
dns:
  port: 53
  upstreams:
    - 1.1.1.1

api:
  port: 80

client_lookup:
  upstream: 192.168.8.1 # Router's IP
  method: rdns

groups:
  all:
    block:
      - ads
      - malware
      - adult
```

For more information on the available configuration options, check [this page](config.md).

## Installation

Installation with Docker is by far the easiest method and most recommended.

### Docker Compose

Create a `compose.yml` file with the following content:

```yaml
services:
  beacon-dns:
    container_name: beacon-dns
    image: ghcr.io/st3v3nmw/beacon-dns:latest
    volumes:
      - /home/${USER}/beacon-dns:/data
    environment:
      - BEACON_CONFIG_FILE=/data/config.yml
      - BEACON_DATA_DIR=/data
    restart: unless-stopped
    privileged: true
    network_mode: host
```

This assumes we're storing the service's data in `/home/${USER}/beacon-dns`. Create that folder and put the `config.yml` file there.
You're free to put the volume anywhere but make sure the `compose.yml` file matches.

Start the container by running `docker compose up -d`. You can check the logs by running `docker logs beacon-dns`.

#### Updating

```console
$ docker compose down
$ docker pull ghcr.io/st3v3nmw/beacon-dns:latest
$ docker compose up -d
```

### Standalone binary

Clone this repository: `git clone https://github.com/st3v3nmw/beacon-dns.git`

Build the binary: `make build`

Make it executable: `chmod +x beacon`

Download and extract [`sqlean` extensions](https://github.com/nalgeon/sqlean/releases/) in some folder (`BEACON_EXTENSIONS_DIR`).

Create a folder to store Beacon DNS' data (`BEACON_DATA_DIR`).

Export the `BEACON_EXTENSIONS_DIR`, `BEACON_DATA_DIR`, and `BEACON_CONFIG_FILE` environment variables where `BEACON_CONFIG_FILE` is the path to your `config.yml`.

Start the server: `./beacon`.

If you're on certain Linux distributions, you can create a systemd service file to start the service on boot automatically.

#### Ansible

[Here's](https://github.com/st3v3nmw/beacon-dns/blob/main/deploy/ansible/ubuntu.yml) an Ansible script to install and run Beacon DNS on Ubuntu. You'll need to follow the steps above to build the standalone binary and then run `make ansible-deploy`.
