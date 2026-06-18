# Clashtars

Small Mihomo wrapper:

```text
prepare -> fetch/convert providers, synthesize config.yaml from template.yaml
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
sudo nano /var/lib/clashtars/template.yaml
```

Set `subscription.providers[].url`, then run:

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
/var/lib/clashtars/template.yaml
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
cp configs/template.yaml.example /tmp/clashtars/template.yaml
cd /tmp/clashtars
```

Edit `clash.conf`, set `subscription.providers[].url`, and adjust `mihomo:`
ports/UI/controller settings as needed. Keep long routing policy in
`template.yaml`, then run:

```bash
./clashtars prepare
./clashtars start
```

Runtime files stay in the current working directory:

```text
config.yaml
providers/
cache/
ui/
```

`make build-dev` is only an offline compile check; it is not the complete
embedded binary.
