# Epok Forwarder

An easily configurable, multi-functional network traffic forwarder that supports port mapping, port range forwarding, and HTTP/HTTPS reverse proxy based on host or SNI.

## Install

With Go 1.21+:

```bash
go install github.com/juzeon/epok-forwarder@latest
```

## Configure

```yaml
# config.yml

# Assume the server address is 1.1.1.1
http: 80 # Default to 80
https: 443 # Default to 443
hosts:
  - host: 172.16.1.2
    forwards:
      - type: port # TCP + UDP port mapping
        src: 2023 # Listen on 0.0.0.0:2023 on the server
        dst: 2024 # Forward to 172.16.1.2:2024
  - host: 172.16.1.3
    http: 80 # Default to 80
    https: 443 # Default to 443
    forwards:
      - type: port_range # TCP + UDP port range forwarding
        port_range: 700-710,715,716,720-730 # The server and the host listen on the same port numbers. Inclusive on both sides
      - type: web # Host-based for HTTP, SNI-based for HTTPS
        hostnames:
          - *example.com # Will match example.com, a.example.com, a.b.c.example.com, hello-example.com, etc
          - ?gg.com # Will match egg.com, ogg.com, etc
```

## Run

```bash
./epok-forwarder -c my_config.yml # Specify a configuration file
# Or
./epok-forwarder # Use config.yml in the current directory automatically
```

