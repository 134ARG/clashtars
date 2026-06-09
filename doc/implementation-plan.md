# Clashtars Implementation Plan

## Goal

Build a small Go wrapper for a Clash/Mihomo service:

```text
systemd -> prepare(fetch/convert/synthesize) -> start(memfd exec mihomo) -> REST/UI on 9091
```

Keep it simple. Fail when there is no usable config. Stay up when an old usable
config exists and refresh fails.

## Runtime Layout

All service-owned files live under one prefix:

```text
/var/lib/clashtars/
  clash.conf          # admin settings, root-owned, readable by clashtars
  config.yaml         # generated Mihomo config
  subscription.yaml   # last fetched raw subscription
  converted.yaml      # converted Clash profile
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
  url: "https://example.invalid/sub"

mihomo:
  port: 7890
  socks-port: 7891
  redir-port: 7892
  allow-lan: true
  mode: rule
  log-level: silent
  external-controller: "0.0.0.0:9091"
  secret: ""
```

`mihomo:` is passed through as structured YAML, so extra final `config.yaml`
keys can be added without code changes.
YAML comments are fine; they are only template hints and are not preserved.

For standalone use, run from a directory containing `clash.conf`; generated
files go beside that config unless `runtime.root-dir` is set. The systemd unit
passes `/var/lib/clashtars/clash.conf` explicitly.

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
clashtars prepare --config /var/lib/clashtars/clash.conf
clashtars start --config /var/lib/clashtars/clash.conf
```

`make build` is the primary build path. It stages downloaded assets in-place
first, then emits the usable single binary at `build/clashtars`.

## Prepare

1. Load `clash.conf`.
2. Fetch subscription.
3. If fetch fails:
   - use existing non-empty `config.yaml` and exit success;
   - otherwise fail.
4. Convert only when needed:
   - Clash YAML: use directly;
   - base64 Clash YAML: decode;
   - otherwise extract and run embedded subconverter.
5. Parse YAML with `yaml.v3`.
6. Synthesize final config:
   - start from `mihomo:`;
   - embed subscription `proxies`, `proxy-groups`, `rules`, providers;
   - write `config.yaml` atomically.
7. If conversion/parsing/synthesis fails, use old `config.yaml` if present;
   otherwise fail.

## Start

1. Load `clash.conf`.
2. Extract embedded MetacubeXD UI assets into `mihomo.external-ui` or `ui/`.
3. Load embedded Mihomo bytes.
4. Execute embedded Mihomo from RAM with `memfd_create`.
5. Inherit stdout/stderr.
6. Use the config directory as Mihomo config directory.

No binary selection. x86 only.

Subconverter is downloaded during release builds as
`subconverter_linux64.tar.gz`, embedded, and extracted under
`/var/lib/clashtars/cache/subconverter` only when conversion needs it.

## Packaging

RPM packaging is auxiliary. It builds from the same source shape and installs:

```text
/usr/bin/clashtars
/var/lib/clashtars/clash.conf
/var/lib/clashtars/ui/
/usr/lib/systemd/system/clashtars.service
```

Mihomo is downloaded from the latest MetaCubeX/mihomo GitHub release during
asset staging. The release gzip is decompressed first; only the executable is
embedded. Subconverter is downloaded from the latest upstream GitHub release
during asset staging. MetacubeXD is downloaded from release `v1.251.3` as
`compressed-dist.tgz`, embedded, and extracted to `ui/` at start.

Geo DBs are not staged or packaged for now. If needed, users can provide them
through explicit Mihomo config.

## Tests

Cover only the risky wrapper behavior:

- config load and defaults;
- YAML synthesis;
- fetch failure with old config succeeds;
- fetch failure without old config fails;
- memfd start path can be constructed.
