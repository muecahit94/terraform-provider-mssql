# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.4](https://github.com/muecahit94/terraform-provider-mssql/compare/v1.0.3...v1.0.4) (2026-01-01)


### Bug Fixes

* Add Azure AD authentication, database-specific connections ([d8b8d8d](https://github.com/muecahit94/terraform-provider-mssql/commit/d8b8d8d163da305e30218e93043926eaeb902374))

## [1.0.3](https://github.com/muecahit94/terraform-provider-mssql/compare/v1.0.2...v1.0.3) (2026-01-01)


### Miscellaneous

* Add pre-commit configuration for Go, Terraform, and general code quality checks ([363740a](https://github.com/muecahit94/terraform-provider-mssql/commit/363740a911299c866fe6ffcb09cd2f0a11c8c204))

## [1.0.2](https://github.com/muecahit94/terraform-provider-mssql/compare/v1.0.1...v1.0.2) (2026-01-01)


### Bug Fixes

* prevent `mssql_schema_permission` drift for `with_grant_option` and ensure `REVOKE CASCADE`. ([a67215a](https://github.com/muecahit94/terraform-provider-mssql/commit/a67215ab48251e748916c11a026270eedc0ad5d7))

## [1.0.1](https://github.com/muecahit94/terraform-provider-mssql/compare/v1.0.0...v1.0.1) (2025-12-31)


### Bug Fixes

* Update `mssql` provider version to `~> 1.0` in all examples and documentation. ([1856114](https://github.com/muecahit94/terraform-provider-mssql/commit/18561145b8a1df08964c8c0db2e4e75b2f69828f))

## 1.0.0 (2025-12-31)


### Features

* Add end-to-end testing framework and start provider versions from 0 ([a0a47bb](https://github.com/muecahit94/terraform-provider-mssql/commit/a0a47bb8e170ae72747b0b9559cbb504e5a32a94))
* disable GPG signing in GoReleaser and the release workflow. ([ea323c9](https://github.com/muecahit94/terraform-provider-mssql/commit/ea323c992ede2099a34771ece4211e7db324442b))
* Implement initial MSSQL Terraform provider with core resources, data sources, and documentation. ([26488ed](https://github.com/muecahit94/terraform-provider-mssql/commit/26488ed7c0349e4b7167a6c1bd75890d5fbc3f57))
* improve SQL login update logic, refactor database context handling, and update examples ([1e974ba](https://github.com/muecahit94/terraform-provider-mssql/commit/1e974bad2fcb24f46436adc030d36daf77b26531))
