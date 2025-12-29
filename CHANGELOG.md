# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
