################################################################################

%global crc_check pushd ../SOURCES ; sha512sum -c %{SOURCE100} ; popd

################################################################################

%define debug_package  %{nil}

################################################################################

%define _logdir  %{_localstatedir}/log

################################################################################

Summary:        Tool to periodically clean swap memory
Name:           swap-reaper
Version:        0.0.2
Release:        0%{?dist}
Group:          Applications/System
License:        Apache License, Version 2.0
URL:            https://kaos.sh/swap-reaper

Source0:        https://source.kaos.st/%{name}/%{name}-%{version}.tar.bz2

Source100:      checksum.sha512

BuildRoot:      %{_tmppath}/%{name}-%{version}-%{release}-root-%(%{__id_u} -n)

BuildRequires:  golang >= 1.23

Requires:       systemd

Provides:       %{name} = %{version}-%{release}

################################################################################

%description
Tool to periodically clean swap memory.

################################################################################

%prep
%{crc_check}

%setup -q
if [[ ! -d "%{name}/vendor" ]] ; then
  echo -e "----\nThis package requires vendored dependencies\n----"
  exit 1
elif [[ -f "%{name}/%{name}" ]] ; then
  echo -e "----\nSources must not contain precompiled binaries\n----"
  exit 1
fi

%build
pushd %{name}
  %{__make} %{?_smp_mflags} all
  cp LICENSE ..
popd

%install
rm -rf %{buildroot}

install -dDm 755 %{buildroot}%{_bindir}
install -dDm 755 %{buildroot}%{_sysconfdir}/logrotate.d
install -dDm 755 %{buildroot}%{_unitdir}
install -dDm 755 %{buildroot}%{_logdir}/%{name}

install -pm 755 %{name}/%{name} \
                %{buildroot}%{_bindir}/

install -pm 644 %{name}/common/%{name}.knf \
                %{buildroot}%{_sysconfdir}/

install -pm 644 %{name}/common/%{name}.logrotate \
                %{buildroot}%{_sysconfdir}/logrotate.d/%{name}

install -pm 644 %{name}/common/%{name}.service \
                %{buildroot}%{_unitdir}/

%clean
rm -rf %{buildroot}

%preun
if [[ $1 -eq 0 ]] ; then
  systemctl --no-reload disable %{name}.service &>/dev/null || :
  systemctl stop %{name}.service &>/dev/null || :
fi

%postun
if [[ $1 -ge 1 ]] ; then
  systemctl daemon-reload &>/dev/null || :
fi

################################################################################

%files
%defattr(-,root,root,-)
%doc LICENSE
%dir %{_logdir}/%{name}
%config(noreplace) %{_sysconfdir}/%{name}.knf
%config(noreplace) %{_sysconfdir}/logrotate.d/%{name}
%{_unitdir}/%{name}.service
%{_bindir}/%{name}

################################################################################

%changelog
* Thu May 15 2025 Anton Novojilov <andy@essentialkaos.com> - 0.0.2-0
- Code refactoring
- Dependencies update

* Thu Sep 12 2024 Anton Novojilov <andy@essentialkaos.com> - 0.0.1-0
- The very first version
