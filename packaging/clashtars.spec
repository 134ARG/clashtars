%global debug_package %{nil}
%global __os_install_post %{nil}

Name:           %{name}
Version:        %{version}
Release:        %{release}%{?dist}
Summary:        Clashtars Mihomo service wrapper
License:        Unknown
Source0:        %{name}-%{version}.tar.gz

BuildRequires:  golang
Requires:       systemd
AutoReqProv:    no

%description
Clashtars is a small Go wrapper that prepares a Mihomo config and starts an
embedded x86 Mihomo core from memory.

%prep
%autosetup -n %{name}-%{version}

%build
./scripts/stage-assets.sh
GOCACHE="${PWD}/build/go-cache" go build -trimpath -buildvcs=false -o build/clashtars ./cmd/clashtars

%install
rm -rf %{buildroot}

install -D -m 0755 build/clashtars %{buildroot}%{_bindir}/clashtars
install -D -m 0644 packaging/clashtars.service %{buildroot}%{_unitdir}/clashtars.service

install -d -m 0750 %{buildroot}%{_sharedstatedir}/clashtars
install -D -m 0640 configs/clash.conf.example %{buildroot}%{_sharedstatedir}/clashtars/clash.conf
install -D -m 0640 configs/template.yaml.example %{buildroot}%{_sharedstatedir}/clashtars/template.yaml

install -d -m 0750 %{buildroot}%{_sharedstatedir}/clashtars/ui

%pre
getent group clashtars >/dev/null || groupadd -r clashtars
getent passwd clashtars >/dev/null || \
  useradd -r -g clashtars -d %{_sharedstatedir}/clashtars -s /sbin/nologin \
    -c "Clashtars service user" clashtars

%post
%systemd_post clashtars.service

%preun
%systemd_preun clashtars.service

%postun
%systemd_postun_with_restart clashtars.service

%files
%{_bindir}/clashtars
%{_unitdir}/clashtars.service
%dir %attr(0750,clashtars,clashtars) %{_sharedstatedir}/clashtars
%config(noreplace) %attr(0640,root,clashtars) %{_sharedstatedir}/clashtars/clash.conf
%config(noreplace) %attr(0640,root,clashtars) %{_sharedstatedir}/clashtars/template.yaml
%dir %attr(0750,clashtars,clashtars) %{_sharedstatedir}/clashtars/ui

%changelog
* Tue Jun 09 2026 Codex <codex@example.invalid> - 0.1.0-1
- Initial clashtars wrapper package.
