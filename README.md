# Cowpoke

A CLI tool for managing multiple Rancher servers and downloading kubeconfigs from all clusters across all servers.

## Features

- **Multi-Server Management**: Connect to multiple Rancher environments (prod, staging, dev) from one tool
- **One-Command Sync**: Get all your cluster access with a single `cowpoke sync` command
- **Smart Filtering**: Skip test/dev clusters with `--exclude` patterns when syncing production configs
- **No Name Conflicts**: Safely merge clusters with identical names from different Rancher servers
- **Fast Downloads**: Concurrent processing speeds up syncing from multiple servers
- **Secure by Default**: Never stores passwords, supports all Rancher authentication methods
- **Zero Configuration**: Automatically migrates from older versions

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

# Add a server with Active Directory (supports both formats)
cowpoke add --url https://rancher.internal.com --username DOMAIN\\user --authtype activedirectory
cowpoke add --url https://rancher.internal.com --username user@domain.com --authtype activedirectory
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

# Exclude clusters by name using regex patterns
cowpoke sync --exclude "^test-.*" --exclude ".*-staging$"

# Combine multiple options
cowpoke sync --output /custom/kubeconfig --exclude "^dev-.*" --cleanup-temp-files --insecure
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

### Configuration Format

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

Configuration is automatically migrated from older versions when you first run the tool.

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
- `activedirectory` - Active Directory authentication
- `github` - GitHub OAuth authentication
- `googleoauth` - Google OAuth authentication
- `shibboleth` - Shibboleth authentication
- `azuread` - Azure AD authentication
- `keycloak` - Keycloak authentication
- `ping` - Ping authentication
- `okta` - Okta authentication
- `freeipa` - FreeIPA authentication

## How Kubeconfig Merging Works

When syncing kubeconfigs from multiple Rancher servers, Cowpoke:
1. Authenticates with all Rancher servers concurrently
2. Discovers all available clusters from each server
3. Filters out clusters matching any `--exclude` patterns (before downloading)
4. Downloads kubeconfigs only for the remaining clusters
5. Appends a unique server ID to all resource names (clusters, users, contexts)
6. Merges all kubeconfigs into a single file
7. Prevents naming conflicts between different Rancher servers

Example: A cluster named `production` from server `rancher.example.com` becomes `production-55110d2f` in the merged kubeconfig.

### Cluster Filtering

Use the `--exclude` flag to filter out clusters by name using regex patterns. This is useful for:
- Excluding test/development clusters from production kubeconfigs
- Filtering out temporary or staging environments
- Reducing the size of merged kubeconfigs

```bash
# Exclude all clusters starting with "test-"
cowpoke sync --exclude "^test-.*"

# Exclude all clusters ending with "-staging"
cowpoke sync --exclude ".*-staging$"

# Multiple patterns (exclude clusters matching any pattern)
cowpoke sync --exclude "^test-.*" --exclude "^dev-.*" --exclude ".*-staging$"

# Complex patterns
cowpoke sync --exclude "^(test|dev|staging)-.*"
```

**Pattern Examples:**
- `^test-.*` - Matches clusters starting with "test-"
- `.*-staging$` - Matches clusters ending with "-staging"
- `^temp-cluster-[0-9]+$` - Matches "temp-cluster-123" but not "temp-cluster-abc"
- `^(dev|test|staging)-.*` - Matches clusters starting with "dev-", "test-", or "staging-"

## Building from Source

```bash
# Clone the repository
git clone https://github.com/imandrew/cowpoke.git
cd cowpoke

# Build the binary
make build
```

Requirements: Go 1.21+

## Troubleshooting

### Common Issues

1. **"No servers configured"**: Add at least one Rancher server using `cowpoke add`
2. **Authentication failures**: 
   - Verify your username and password are correct
   - Check that the auth type matches your Rancher server configuration
   - For non-interactive use, ensure `RANCHER_PASSWORD` environment variable is set
3. **Network errors**: 
   - Check connectivity to your Rancher servers
   - Use `--insecure` flag if you have self-signed certificates
4. **Permission denied**: Ensure you have write access to `~/.config/cowpoke/` and `~/.kube/`
5. **"No kubeconfigs downloaded"**: Check if clusters are being filtered out by `--exclude` patterns
6. **Invalid regex patterns**: Verify your `--exclude` patterns are valid regex expressions

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

