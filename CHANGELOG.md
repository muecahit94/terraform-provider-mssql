# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.3.0](https://github.com/muecahit94/terraform-provider-mssql/compare/v1.2.1...v1.3.0) (2025-12-30)


### Features

* disable GPG signing in GoReleaser and the release workflow. ([ea323c9](https://github.com/muecahit94/terraform-provider-mssql/commit/ea323c992ede2099a34771ece4211e7db324442b))
* Implement initial MSSQL Terraform provider with core resources, data sources, and documentation. ([26488ed](https://github.com/muecahit94/terraform-provider-mssql/commit/26488ed7c0349e4b7167a6c1bd75890d5fbc3f57))


### Miscellaneous

* bump project version to 1.2.1 ([3ace538](https://github.com/muecahit94/terraform-provider-mssql/commit/3ace538a56c64a334a2ad03d8bac42269abc0e11))
* **main:** release 1.1.0 ([0ff8429](https://github.com/muecahit94/terraform-provider-mssql/commit/0ff8429a61b13ca62a1d0076ffc1b04509e14b32))
* **main:** release 1.1.0 ([611912b](https://github.com/muecahit94/terraform-provider-mssql/commit/611912baf2bdd79b800507ce220d56f2b0c41d6c))
* **main:** release 1.2.0 ([7c3e946](https://github.com/muecahit94/terraform-provider-mssql/commit/7c3e94643a5d80ca690fef010ff9fca831a6b545))
* **main:** release 1.2.0 ([464edc4](https://github.com/muecahit94/terraform-provider-mssql/commit/464edc400709c75ec02584a798f6f0f375503ec7))

## [1.2.0](https://github.com/muecahit94/terraform-provider-mssql/compare/v1.1.0...v1.2.0) (2025-12-29)


### Features

* disable GPG signing in GoReleaser and the release workflow. ([ea323c9](https://github.com/muecahit94/terraform-provider-mssql/commit/ea323c992ede2099a34771ece4211e7db324442b))

## [1.1.0](https://github.com/muecahit94/terraform-provider-mssql/compare/v1.0.0...v1.1.0) (2025-12-29)


### Features

* Implement initial MSSQL Terraform provider with core resources, data sources, and documentation. ([26488ed](https://github.com/muecahit94/terraform-provider-mssql/commit/26488ed7c0349e4b7167a6c1bd75890d5fbc3f57))

## [Unreleased]

### Added
- Initial provider implementation with Terraform Plugin Framework
- SQL and Azure AD authentication support
- Resources: database, sql_login, sql_user, database_role, database_role_member
- Resources: database_permission, schema, schema_permission
- Resources: server_role, server_role_member, server_permission
- Resources: script, azuread_user, azuread_service_principal
- Data sources for all resources
- Query data source for custom SQL queries
- Full import support for all resources
- CI/CD with GitHub Actions
- Acceptance tests with Docker SQL Server
