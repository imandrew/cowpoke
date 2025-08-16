# Cowpoke

A CLI tool for managing multiple Rancher servers and downloading kubeconfigs from all clusters across all servers.

## Features

- **Multi-Server Management**: Configure and manage connections to multiple Rancher servers
- **Automatic Kubeconfig Merging**: Download and intelligently merge kubeconfigs from all clusters
- **Conflict-Free Resource Naming**: Automatically handles naming conflicts when merging kubeconfigs from different servers
- **Secure Authentication**: Supports multiple authentication types with secure password handling
- **Automatic Configuration Migration**: Seamlessly migrates from v1.x to v2.0 configuration format

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
# Add a server with local authentication
cowpoke add --url https://rancher.example.com --username admin --authtype local

# Add a server with OpenLDAP authentication
cowpoke add --url https://rancher.corp.com --username jdoe --authtype openldap

# Add a server with Active Directory
cowpoke add --url https://rancher.internal.com --username DOMAIN\\user --authtype activeDirectory
```

### List Configured Servers

```bash
cowpoke list

# Example output:
# Configured Rancher Servers:
# 
# URL: https://rancher.prod.example.com
#   Username: admin
#   Auth Type: local
#   ID: 55110d2f
# 
# URL: https://rancher.staging.example.com
#   Username: devuser
#   Auth Type: openldap
#   ID: 955622f1
```

### Remove a Server

```bash
cowpoke remove --url https://rancher.example.com
```

### Sync Kubeconfigs

Download kubeconfigs from all clusters across all configured servers:

```bash
# Merge into default location (~/.kube/config)
cowpoke sync

# Specify a custom output file
cowpoke sync --output /path/to/kubeconfig

# Save to a directory (will create 'config' file in that directory)
cowpoke sync --output /path/to/directory/

# Skip TLS certificate verification (useful for self-signed certificates)
cowpoke sync --insecure

# Clean up temporary kubeconfig files after merging
cowpoke sync --cleanup-temp-files
```

### Global Options

```bash
# Verbose output for debugging
cowpoke --verbose sync

# Use custom configuration file
cowpoke --config /path/to/config.yaml list
```

## Configuration

Configuration is stored in `~/.config/cowpoke/config.yaml`.

### Configuration Format (v2.0)

```yaml
version: "2.0"
servers:
  - url: "https://rancher.prod.example.com"
    username: "admin"
    authType: "local"
  - url: "https://rancher.staging.example.com"
    username: "devuser"
    authType: "openldap"
```

### Automatic Migration from v1.x

When upgrading from v1.x to v2.0, your configuration will be automatically migrated on first use. The migration:
- Preserves all server connections
- Removes deprecated `id` and `name` fields
- Generates dynamic IDs based on server domains
- Updates the version to "2.0"

Your original configuration is replaced with the migrated version after successful migration.

## Authentication

### Password Handling

Cowpoke securely handles passwords for Rancher authentication:

1. **Interactive Mode** (default): Prompts for password with secure input (no echo)
2. **Non-Interactive Mode**: Use the `RANCHER_PASSWORD` environment variable
   ```bash
   export RANCHER_PASSWORD="your-password"
   cowpoke sync
   ```

### Supported Authentication Types

- `local` - Local Rancher authentication
- `openldap` - OpenLDAP authentication
- `activeDirectory` - Active Directory authentication
- `github` - GitHub OAuth authentication
- `googleoauth` - Google OAuth authentication
- `shibboleth` - Shibboleth authentication
- `azuread` - Azure AD authentication
- `keycloak` - Keycloak authentication
- `ping` - Ping authentication
- `okta` - Okta authentication
- `freeipa` - FreeIPA authentication

## How It Works

### Server ID Generation

Each Rancher server is assigned a unique 8-character ID generated from the SHA256 hash of its domain. This ensures:
- Consistent IDs across different installations
- No conflicts when merging kubeconfigs
- Idempotent operations (same domain always generates same ID)

### Kubeconfig Merging

When syncing kubeconfigs from multiple Rancher servers, Cowpoke:
1. Downloads kubeconfigs for all clusters from each server
2. Appends the server ID to all resource names (clusters, users, contexts)
3. Merges all kubeconfigs into a single file
4. Prevents naming conflicts between different Rancher servers

Example: A cluster named `production` from server `rancher.example.com` (ID: `55110d2f`) becomes `production-55110d2f` in the merged kubeconfig.

## Architecture (v2.0)

Cowpoke v2.0 features a clean, maintainable architecture:

- **Domain-Driven Design**: Clear separation of business logic and infrastructure
- **Dependency Injection**: Flexible, testable component composition
- **Interface-Based Design**: Loose coupling between components
- **Comprehensive Testing**: Unit tests with mocked dependencies
- **Security-First**: Secure password handling, proper file permissions

## Development

### Requirements

- Go 1.21+
- golangci-lint (for linting)
- make (for build automation)

### Building from Source

```bash
# Clone the repository
git clone https://github.com/imandrew/cowpoke.git
cd cowpoke

# Build the binary
make build

# Run tests
make test

# Run linter
make lint

# Clean build artifacts
make clean
```

### Project Structure

```
cowpoke/
├── cmd/                    # CLI commands (add, list, remove, sync)
├── internal/
│   ├── adapters/          # External interfaces (HTTP, filesystem, terminal)
│   ├── app/               # Application initialization and DI
│   ├── commands/          # Command implementations
│   ├── domain/            # Domain models and interfaces
│   ├── migrations/        # Configuration migration logic
│   ├── services/          # Business logic services
│   └── mocks/             # Test mocks
├── main.go                # Application entry point
└── README.md
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific package tests
go test ./internal/services/...
```

## Troubleshooting

### Common Issues

1. **"No servers configured"**: Add at least one Rancher server using `cowpoke add`
2. **Authentication failures**: Verify your username and password are correct
3. **Network errors**: Check connectivity to your Rancher servers
4. **Permission denied**: Ensure you have write access to `~/.config/cowpoke/` and `~/.kube/`

### Debug Mode

Run with `--verbose` flag for detailed debug output:

```bash
cowpoke --verbose sync
```

## Security Considerations

- Passwords are never stored in configuration files
- Passwords are cleared from memory immediately after use
- Configuration files are created with restricted permissions (0600)
- Kubeconfig files are saved with secure permissions (0600)
- Optional TLS certificate verification bypass for self-signed certificates

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

## License

MIT License - see [LICENSE](LICENSE) file for details.

