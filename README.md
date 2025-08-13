# Cowpoke

A CLI tool for managing multiple Rancher servers and downloading kubeconfigs from all clusters across all servers.

## Features

- **Add Rancher servers**: Configure multiple Rancher server connections with different authentication types
- **List servers**: View all configured Rancher servers
- **Remove servers**: Remove Rancher servers from configuration
- **Sync kubeconfigs**: Download and merge kubeconfigs from all clusters across all configured Rancher servers

## Installation

### Homebrew (macOS)

```bash
brew install imandrew/tap/cowpoke
```

### Manual Installation

Download the latest release from the [releases page](https://github.com/imandrew/cowpoke/releases) or build from source:

```bash
git clone https://github.com/imandrew/cowpoke.git
cd cowpoke
make build
```

## Usage

### Add a Rancher Server

```bash
cowpoke add --url https://rancher.example.com --username admin --authtype local
```

### List Configured Servers

```bash
cowpoke list
```

### Remove a Server

```bash
cowpoke remove --url https://rancher.example.com
```

### Sync Kubeconfigs

Download kubeconfigs from all clusters and merge into `~/.kube/config`:

```bash
cowpoke sync
```

Specify a custom output location:

```bash
cowpoke sync --output /path/to/kubeconfig
cowpoke sync --output /path/to/directory/  # saves as config in directory
```

## Configuration

Configuration is stored in `~/.config/cowpoke/config.yaml`. You can specify a custom config file:

```bash
cowpoke --config /path/to/config.yaml sync
```

### Configuration Format

```yaml
version: "1.0"
servers:
  - id: "server1"
    name: "Production Rancher"
    url: "https://rancher.prod.example.com"
    username: "admin"
    authType: "local"
  - id: "server2"
    name: "Development Rancher"
    url: "https://rancher.dev.example.com"
    username: "devuser"
    authType: "github"
```

## Authentication Types

- `local`: Local Rancher authentication
- `github`: GitHub OAuth authentication
- Other authentication types supported by your Rancher installation

## Development

### Requirements

- Go 1.21+
- golangci-lint

### Common Commands

```bash
make lint     # Run linter
make test     # Run tests
make build    # Build binary
make clean    # Clean artifacts
```

## License

MIT License - see [LICENSE](LICENSE) file for details.