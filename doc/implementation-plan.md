# Clashtars Implementation Plan

## Goal

Build a small Go wrapper for a Clash/Mihomo service:

```text
systemd -> prepare(fetch/convert providers + template synthesize) -> start(memfd exec mihomo) -> REST/UI on 9091
```

Keep it simple. Provider refresh is best effort. Fail only when synthesis cannot
produce a usable config and no old usable config exists.

## Runtime Layout

All service-owned files live under one prefix:

```text
/var/lib/clashtars/
  clash.conf          # admin settings, root-owned, readable by clashtars
  template.yaml       # admin-maintained Mihomo routing template
  config.yaml         # generated Mihomo config
  providers/          # raw, converted, and provider YAML files
  ui/                 # dashboard assets
  cache/
```

No app log files. stdout/stderr go to systemd/journald.

## Systemd

Run as a dedicated unprivileged user:

```ini
User=clashtars
Group=clashtars
AmbientCapabilities=CAP_NET_ADMIN CAP_NET_RAW
CapabilityBoundingSet=CAP_NET_ADMIN CAP_NET_RAW
WorkingDirectory=/var/lib/clashtars
ExecStartPre=/usr/bin/clashtars prepare --config /var/lib/clashtars/clash.conf
ExecStart=/usr/bin/clashtars start --config /var/lib/clashtars/clash.conf
```

The RPM creates the `clashtars` user/group in `%pre`. Ports are above 1024, so
root runtime is not needed, but redir/tproxy needs capabilities and still needs
host nftables/iptables redirect rules configured separately.

## Config

`/var/lib/clashtars/clash.conf` is YAML:

```yaml
subscription:
  timeout: 30s
  user-agent: clashtars/1.0
  providers:
    - name: main
      url: "https://example.invalid/sub"
      prefix: "[main] "

mihomo:
  port: 7890
  socks-port: 7891
  redir-port: 7892
  mixed-port: 7893
  allow-lan: false
  mode: rule
  log-level: info
  external-controller: "0.0.0.0:9091"
  external-ui: "/var/lib/clashtars/ui"
  secret: ""
  profile:
    tracing: true
    store-selected: true
  dns:
    enable: true
    ipv6: false
    listen: 0.0.0.0:53
    nameserver:
      - 100.100.100.100
      - 119.29.29.29
      - 223.5.5.5
```

`mihomo:` is overlaid onto the final generated Mihomo config, so short
operational settings stay in `clash.conf`. `template.yaml` is passed through as
structured YAML and should define long routing policy such as `proxy-groups` and
`rules`; use `__PROVIDER_PLACEHOLDER__` inside a group's `use` list to insert
all configured provider names.

Runtime files are written under the current working directory. The systemd unit
uses `/var/lib/clashtars` as `WorkingDirectory`.

## Go Shape

```text
clashtars/
  cmd/clashtars/main.go
  internal/
    config.go
    prepare.go
    fetch.go
    convert.go
    synthesize.go
    start.go
    memfd.go
  packaging/
  doc/
```

One CLI, two commands:

```text
clashtars prepare --config /var/lib/clashtars/clash.conf --template /var/lib/clashtars/template.yaml
clashtars start --config /var/lib/clashtars/clash.conf
```

`make build` is the primary build path. It stages downloaded assets in-place
first, then emits the usable single binary at `build/clashtars`.

## Prepare

1. Load `clash.conf`.
2. Load `template.yaml`.
3. For each `subscription.providers[]`, try to refresh its local provider file:
   - fetch the subscription;
   - use Clash/provider YAML directly when it already has `proxies`;
   - decode base64 YAML when needed;
   - otherwise extract and run embedded subconverter;
   - extract only top-level `proxies`;
   - write `providers/<name>.yaml` atomically.
4. Refresh failures are warnings and leave old provider files untouched.
5. Synthesize final config:
   - start from `template.yaml`;
   - overlay `mihomo:` from `clash.conf`;
   - inject `proxy-providers` for local provider files;
   - expand `__PROVIDER_PLACEHOLDER__`;
   - write `config.yaml` atomically.
6. If synthesis fails, use old `config.yaml` if present;
   otherwise fail.

## Start

1. Load `clash.conf`.
2. Extract embedded MetacubeXD UI assets into `mihomo.external-ui` or `ui/`.
3. Load embedded Mihomo bytes.
4. Execute embedded Mihomo from RAM with `memfd_create`.
5. Inherit stdout/stderr.
6. Use the current working directory as Mihomo config directory.

No binary selection. x86 only.

Subconverter is downloaded during release builds as
`subconverter_linux64.tar.gz`, embedded, and extracted under
`/var/lib/clashtars/cache/subconverter` only when conversion needs it.

## Packaging

RPM packaging is auxiliary. It builds from the same source shape and installs:

```text
/usr/bin/clashtars
/var/lib/clashtars/clash.conf
/var/lib/clashtars/template.yaml
/var/lib/clashtars/ui/
/usr/lib/systemd/system/clashtars.service
```

Mihomo is downloaded from the latest MetaCubeX/mihomo GitHub release during
asset staging. The release gzip is decompressed first; only the executable is
embedded. Subconverter is downloaded from the latest upstream GitHub release
during asset staging. MetacubeXD is downloaded from release `v1.251.3` as
`compressed-dist.tgz`, embedded, and extracted to `ui/` at start.

Geo DBs are not staged or packaged for now. If needed, users can provide them
through `template.yaml`.

## Tests

Cover only the risky wrapper behavior:

- config load and provider defaults;
- template/provider YAML synthesis;
- refresh failure with old config succeeds;
- refresh failure without old config fails;
- memfd start path can be constructed.
