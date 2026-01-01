# Contributing to terraform-provider-mssql

Thank you for your interest in contributing! This document provides guidelines and instructions for contributing.

## Development Setup

### Prerequisites

- Go 1.21+
- Terraform 1.0+
- Docker (for running SQL Server locally)
- Make

### Getting Started

1. Clone the repository:
   ```bash
   git clone https://github.com/muecahit94/terraform-provider-mssql.git
   cd terraform-provider-mssql
   ```

2. Install dependencies:
   ```bash
   go mod download
   ```

3. Build the provider:
   ```bash
   make build
   ```

4. Run tests:
   ```bash
   make test
   ```

### Local Development

Start a local SQL Server instance:
```bash
docker-compose up -d
```

Run acceptance tests:
```bash
make testacc
```

### Using Local Provider

Create a `.terraformrc` file in your home directory:
```hcl
provider_installation {
  dev_overrides {
    "muecahit94/mssql" = "/path/to/terraform-provider-mssql"
  }
  direct {}
}
```

## Making Changes

### Code Style

- Follow Go best practices and conventions
- Run `go fmt` and `go vet` before committing
- Run `make lint` to check for issues

### Commit Messages

We use [Conventional Commits](https://www.conventionalcommits.org/):

- `feat:` New feature
- `fix:` Bug fix
- `docs:` Documentation only
- `refactor:` Code refactoring
- `test:` Adding tests
- `chore:` Maintenance tasks

Examples:
```
feat: add support for contained database users
fix: handle NULL default_schema in user resource
docs: update README with Azure AD examples
```

### Pull Request Process

1. Fork the repository
2. Create a feature branch: `git checkout -b feat/my-feature`
3. Make your changes
4. Add tests for new functionality
5. Run `make test` and `make lint`
6. Commit with conventional commit messages
7. Push and create a Pull Request

### Adding a New Resource

1. Add client methods in `internal/mssql/`
2. Create resource in `internal/provider/resource_*.go`
3. Add data source in `internal/provider/datasource_*.go`
4. Register in `internal/provider/provider.go`
5. Add documentation in `docs/resources/` and `docs/data-sources/`
6. Add examples in `examples/resources/`
7. Write acceptance tests

## Testing

### Unit Tests
```bash
make test
```

### Acceptance Tests
Acceptance tests run against a real SQL Server instance:
```bash
# Start SQL Server
docker-compose up -d

# Run tests
make testacc
```

### End-to-End Tests
E2E tests verify the full lifecycle including infrastructure provisioning:

```bash
# Run local E2E tests (using Docker)
make e2e-local

# Run Azure AD E2E tests (requires az login and sqlcmd/mssql-cli)
make e2e-azure

# Run full suite
make e2e-full
```

### Test Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `MSSQL_HOSTNAME` | SQL Server hostname | `localhost` |
| `MSSQL_PORT` | SQL Server port | `1433` |
| `MSSQL_USERNAME` | SA username | `sa` |
| `MSSQL_PASSWORD` | SA password | `P@ssw0rd123!` |

## Documentation

- Resource docs: `docs/resources/<resource_name>.md`
- Data source docs: `docs/data-sources/<data_source_name>.md`
- Examples: `examples/resources/<resource_name>/`

Generate documentation:
```bash
make docs
```

## Project Structure

```
.
├── docs/                  # Terraform registry documentation
├── examples/              # Example configurations
├── internal/
│   ├── mssql/            # SQL Server client
│   └── provider/         # Terraform provider implementation
├── main.go               # Provider entry point
├── Makefile              # Build and development tasks
└── docker-compose.yml    # Local SQL Server for testing
```

## Getting Help

- Open an issue for bugs or feature requests
- Check existing issues before creating new ones

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
