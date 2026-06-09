# Clashtars

Small Mihomo wrapper:

```text
prepare -> fetch/convert/synthesize config.yaml
start   -> extract UI, exec embedded Mihomo
```

## RPM

Build and install:

```bash
make rpm
sudo dnf install build/rpmbuild/RPMS/*/clashtars-*.rpm
```

Edit the installed config:

```bash
sudo nano /var/lib/clashtars/clash.conf
```

Set `subscription.url`, then run:

```bash
sudo systemctl enable --now clashtars
journalctl -u clashtars -f
```

Uninstall keeps edited config, runtime files, and the service user:

```bash
sudo dnf remove clashtars
```

Full local purge:

```bash
sudo rm -rf /var/lib/clashtars
sudo userdel clashtars
sudo groupdel clashtars
```

The package installs:

```text
/usr/bin/clashtars
/var/lib/clashtars/clash.conf
/var/lib/clashtars/ui/
/usr/lib/systemd/system/clashtars.service
```

## Single Binary

Build the real embedded binary:

```bash
make build
```

Use it from any writable directory:

```bash
mkdir -p /tmp/clashtars
cp build/clashtars /tmp/clashtars/
cp configs/clash.conf.example /tmp/clashtars/clash.conf
cd /tmp/clashtars
```

Edit `clash.conf`, set `subscription.url`, then run:

```bash
./clashtars prepare
./clashtars start
```

Runtime files stay beside `clash.conf`:

```text
config.yaml
subscription.yaml
converted.yaml
cache/
ui/
```

`make build-dev` is only an offline compile check; it is not the complete
embedded binary.
