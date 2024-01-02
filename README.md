# Epok Forwarder

An easily configurable, multi-functional network traffic forwarder that supports port mapping, port range forwarding, and HTTP/HTTPS reverse proxy based on host or SNI.

## Install

With Go 1.21+:

```bash
go install github.com/juzeon/epok-forwarder@latest
```

## Configure

### Daemon

```yaml
# config.yml

# Assume the server address is 1.1.1.1
api: 127.0.0.1:2035 # API binding addr. Default to 127.0.0.1:2035
secret: epok # Optional but recommended

http: 80 # Optional. Default to 80
https: 443 # Optional. Default to 443

deny: cn,8.8.8.8 # Optional. Can be IP CIDR, country code or IP address, separated by commas. Default to allowing all connections
allow: 223.0.0.0/8 # Optional.

hosts:
  - host: 172.16.1.2
    deny: ... # Omitted
    allow: ...
    forwards:
      - type: port # TCP + UDP port mapping
        src: 2023 # Listen on 0.0.0.0:2023 on the server
        dst: 2024 # Forward to 172.16.1.2:2024
        deny: ... # Omitted
        allow: ...
        # Uncomment this to disable UDP:
        # disable_udp: true

  - host: 172.16.1.3
    forwards:
      - type: port_range # TCP + UDP port range forwarding
        port_range: 700-710,715,716,720-730 # The server and the host listen on the same port numbers. Inclusive on both sides
        deny: ... # Omitted
        allow: ...
        # Uncomment this to disable UDP:
        # disable_udp: true

      - type: web # Host-based for HTTP, SNI-based for HTTPS (all TCP)
        http: 80 # Optional. Default to 80
        https: 443 # Optional. Default to 443
        deny: ... # Omitted
        allow: ...
        hostnames:
          - *example.com # Will match example.com, a.example.com, a.b.c.example.com, hello-example.com, etc
          - ?gg.com # Will match egg.com, ogg.com, etc
```

### CLI

The configuration of CLI is in `$HOME/.config/epok-forwarder/.env`:

```bash
EPOK_API=http://127.0.0.1:2035
EPOK_SECRET=epok
```

CLI is used for calling the API, performing hot reload, etc.

## Run

### Daemon

```bash
./epok-forwarder -c my_config.yml # Specify a configuration file
# Or
./epok-forwarder # Use config.yml in the current directory automatically
```

### CLI

```bash
./epok-forwarder
  -c string
    	specify a config file (default "config.yml")
  -g	generate cli env based on the config file
  -h	get help
  -r	perform hot reload
```

