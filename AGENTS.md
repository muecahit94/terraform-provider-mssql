# AGENTS.md

This file defines the working agreement for automated agents and human contributors working on this repository.

## Mission

Create a **modern, reliable Terraform provider** for Microsoft SQL Server and Azure SQL that:

- Uses the **Terraform Plugin Framework** for modern, maintainable code.
- Is **reliable and testable** (unit + acceptance tests).
- Follows a **clean project structure** and shipping workflow.
- Handles edge cases gracefully (ID changes, manual deletions, etc.)

---

## Current Status

### ✅ Completed

| Component | Status |
|-----------|--------|
| Provider Core | ✅ Complete |
| SQL + Azure AD Auth | ✅ Complete |
| 14 Resources | ✅ Complete |
| 18 Data Sources | ✅ Complete |
| CI/CD Workflows | ✅ Complete |
| Documentation | ✅ Complete |
| Examples | ✅ Complete |

### Resources Implemented
- `mssql_database`
- `mssql_sql_login`
- `mssql_sql_user`
- `mssql_database_role`
- `mssql_database_role_member`
- `mssql_database_permission`
- `mssql_schema`
- `mssql_schema_permission`
- `mssql_server_role`
- `mssql_server_role_member`
- `mssql_server_permission`
- `mssql_script`
- `mssql_azuread_user`
- `mssql_azuread_service_principal`

---

## High-Level Deliverables

### Provider
- ✅ Implemented using **Terraform Plugin Framework**
- ✅ Clear version support policy
- ✅ Semantic versioning with changelog
- ✅ Resilient ID handling (recovers from ID changes)

### Repository Structure
- ✅ GitHub Actions CI (lint, tests, security, release)
- ✅ Release automation (release-please)
- ✅ Conventional commits
- ✅ Terraform examples
- ✅ Documentation (README, resource/data source docs)

---

## Scope and Requirements

### Functional Scope
- All core SQL Server resources (databases, logins, users, roles, schemas, permissions)
- Azure AD authentication support (service principals, managed identities)
- Custom SQL script execution

### Quality Requirements
- ✅ Deterministic, repeatable plans and applies
- ✅ Correct CRUD semantics with resilient ID handling
- ✅ Schema correctness (Optional/Required/Computed consistent)
- ✅ Robust error handling

### Compatibility
- Terraform 1.0+
- Go 1.21+
- SQL Server 2016+ and Azure SQL

---

## Implementation Details

### Project Structure
```
.
├── internal/
│   ├── mssql/          # SQL Server client (7 files)
│   │   ├── client.go   # Connection + auth
│   │   ├── database.go
│   │   ├── login.go
│   │   ├── user.go
│   │   ├── role.go
│   │   ├── permission.go
│   │   ├── schema.go
│   │   └── script.go
│   └── provider/       # Terraform provider
│       ├── provider.go
│       ├── resource_*.go (14 files)
│       └── datasource_*.go (9 files)
├── docs/
│   ├── index.md
│   ├── resources/ (14 files)
│   └── data-sources/ (5 files)
├── examples/
│   ├── complete/
│   ├── azure_ad/
│   └── data_sources/
└── .github/workflows/
    ├── ci.yml
    ├── acceptance.yml
    └── release.yml
```

### Resilient Design
Resources handle edge cases gracefully:
- **ID Changes**: If resource ID changes (e.g., after manual recreation), lookup by name and update stored ID
- **Manual Deletion**: Gracefully remove from state if resource no longer exists
- **Connection Issues**: Proper error messages with context

---

## Tests

### Unit Tests
- Schema validation
- Client logic
- Plan modifier logic

### Acceptance Tests
- Guarded behind `TF_ACC=1`
- Run against Docker SQL Server
- Full CRUD + import tests

---

## CI/CD

### Workflows
- **ci.yml**: lint, unit tests, security (govulncheck), multi-platform build
- **acceptance.yml**: Manual trigger for acceptance tests
- **release.yml**: release-please + goreleaser

---

## Required Files ✅

- [x] `.github/workflows/ci.yml`
- [x] `.github/workflows/acceptance.yml`
- [x] `.github/workflows/release.yml`
- [x] `release-please-config.json`
- [x] `Makefile`
- [x] `examples/`
- [x] `docs/`
- [x] `CONTRIBUTING.md`
- [x] `LICENSE`
- [x] `README.md`

---

## Definition of Done

A feature/resource/data source is done when:
- ✅ Implemented using Plugin Framework patterns
- ✅ Has documentation page
- ✅ Has an example snippet
- ✅ Passes CI checks
- ✅ Has import support

---

## Quick Start

```bash
# Development
make lint
make test
make build

# Acceptance tests
docker-compose up -d
TF_ACC=1 make testacc
```

---

## Notes for Contributors

- Keep resources idempotent
- Handle ID changes gracefully (lookup by name)
- Use structured logging
- Avoid perpetual diffs (normalize inputs)
- Use `Sensitive` for secrets
