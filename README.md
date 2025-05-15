<p align="center"><a href="#readme"><img src=".github/images/card.svg"/></a></p>

<p align="center">
  <a href="https://kaos.sh/y/swap-reaper"><img src="https://kaos.sh/y/47f9bd9f8b654f299891299b8df32e71.svg" alt="Codacy" /></a>
  <a href="https://kaos.sh/w/swap-reaper/ci"><img src="https://kaos.sh/w/swap-reaper/ci.svg" alt="GitHub Actions CI Status" /></a>
  <a href="https://kaos.sh/w/swap-reaper/codeql"><img src="https://kaos.sh/w/swap-reaper/codeql.svg" alt="GitHub Actions CodeQL Status" /></a>
  <a href="#license"><img src=".github/images/license.svg"/></a>
</p>

<p align="center"><a href="#installation">Installation</a> • <a href="#ci-status">CI Status</a> • <a href="#contributing">Contributing</a> • <a href="#license">License</a></p>

<br/>

`swap-reaper` is a service to periodically clean swap memory.

### Installation

#### From [ESSENTIAL KAOS Public Repository](https://kaos.sh/kaos-repo)

```bash
sudo dnf install -y https://pkgs.kaos.st/kaos-repo-latest.el$(grep 'CPE_NAME' /etc/os-release | tr -d '"' | cut -d':' -f5).noarch.rpm
sudo dnf install swap-reaper
```

### CI Status

| Branch | Status |
|--------|----------|
| `master` | [![CI](https://kaos.sh/w/swap-reaper/ci.svg?branch=master)](https://kaos.sh/w/swap-reaper/ci?query=branch:master) |
| `develop` | [![CI](https://kaos.sh/w/swap-reaper/ci.svg?branch=develop)](https://kaos.sh/w/swap-reaper/ci?query=branch:develop) |

### Contributing

Before contributing to this project please read our [Contributing Guidelines](https://github.com/essentialkaos/.github/blob/master/CONTRIBUTING.md).

### License

[Apache License, Version 2.0](http://www.apache.org/licenses/LICENSE-2.0)

<p align="center"><a href="https://kaos.dev"><img src="https://raw.githubusercontent.com/essentialkaos/.github/refs/heads/master/images/ekgh.svg"/></a></p>
