# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.8.0](https://github.com/muecahit94/terraform-provider-mssql/compare/v1.7.0...v1.8.0) (2026-10-10)


### Features

* add mssql_agent_job ([5a05a60](https://github.com/muecahit94/terraform-provider-mssql/commit/5a05a60ba159815129e4609e247bbe85c41451c0))
* add mssql_database_object_permission ([176b92c](https://github.com/muecahit94/terraform-provider-mssql/commit/176b92c2d3de7434f18f90a286af45adac866524))
* add mssql_server_configuration ([d31e4e7](https://github.com/muecahit94/terraform-provider-mssql/commit/d31e4e738a91f1fe8c3339bc815013aa6cd817da))
* add mssql_windows_login ([d3dcaf4](https://github.com/muecahit94/terraform-provider-mssql/commit/d3dcaf4efceb0df0defb358abbaa3587d5e4ea76))
* **database:** add deletion_protection ([75722ce](https://github.com/muecahit94/terraform-provider-mssql/commit/75722ceac84077bb3ea29a31745cb7ca3851eda8))
* **database:** manage auto_close, auto_shrink, page_verify, snapshot isolation, query store and trustworthy ([04633ef](https://github.com/muecahit94/terraform-provider-mssql/commit/04633ef44ae8ee1c1d2c29b385ed38350103e27d))
* **database:** manage the owner of a database ([28d1b28](https://github.com/muecahit94/terraform-provider-mssql/commit/28d1b28cda068f05dea8820bf88c989cd8cdc6a9))
* **sql_login:** make the password optional for existing logins ([6323d54](https://github.com/muecahit94/terraform-provider-mssql/commit/6323d542c2f250f70c65a8cd8b5b74b2db5fb8af))
* **sql_user:** map a user to another login in place and fix orphaned users ([0ec05c1](https://github.com/muecahit94/terraform-provider-mssql/commit/0ec05c119e517aa690ad30fb4a423da77495c395))
* **sql_user:** support Windows groups and users without a login ([1d54c09](https://github.com/muecahit94/terraform-provider-mssql/commit/1d54c098013b39bd3f20b57b3e251bdf1cc1bf4c))


### Bug Fixes

* **agent_job:** change jobs in one transaction, detach schedules by ID, hide step commands, validate recurrence ([af6f1b0](https://github.com/muecahit94/terraform-provider-mssql/commit/af6f1b0c15281cca6df48d7578178d1c873e1409))
* **database:** read the collation of an auto-closed database ([2cb801d](https://github.com/muecahit94/terraform-provider-mssql/commit/2cb801db9d1581d12aef79db79cf01e1ba79afd3))
* **security:** quote identifiers and the password literal, validate permission names in all statements ([080c634](https://github.com/muecahit94/terraform-provider-mssql/commit/080c63469d44cf57361a1c66f04e970fa783bc0b))
* **server_configuration:** allow 0 (automatic) below the reported minimum so defaults can be restored ([f0894dd](https://github.com/muecahit94/terraform-provider-mssql/commit/f0894ddcd93f9e0d85bcf76872c662d3fd34ba52))
* **server_configuration:** send RECONFIGURE as a batch ([f19a1dd](https://github.com/muecahit94/terraform-provider-mssql/commit/f19a1ddc7dbe5ee4073d11675c504fbee11b1f9d))
* **server_configuration:** serialize option changes and keep the configured name spelling ([73f816a](https://github.com/muecahit94/terraform-provider-mssql/commit/73f816adf2e53f857e3b431b2aa513b87db9841d))
* **sql_login:** require a non-empty password when a login is created or replaced ([117babe](https://github.com/muecahit94/terraform-provider-mssql/commit/117babe8c051eb0f9f51b60dd74c6cfe59e759b5))
* **windows_login:** drop a half-created login and document that groups cannot be disabled ([e7ffdc2](https://github.com/muecahit94/terraform-provider-mssql/commit/e7ffdc2a80c1cd6bd022703d36c8c1665d26898b))
* **windows_login:** keep the configured name spelling and reject UPN names at plan time ([f76698d](https://github.com/muecahit94/terraform-provider-mssql/commit/f76698dc6017396e77061f7a09590e842e9bc0af))

## [1.7.0](https://github.com/muecahit94/terraform-provider-mssql/compare/v1.6.0...v1.7.0) (2026-10-09)


### Features

* add mssql_linked_server and mssql_linked_server_login resources ([#32](https://github.com/muecahit94/terraform-provider-mssql/issues/32)) ([af15a14](https://github.com/muecahit94/terraform-provider-mssql/commit/af15a1482510c936f87bd9c08116c47e4f78685c))
* **database:** manage collation, compatibility level and recovery model ([#33](https://github.com/muecahit94/terraform-provider-mssql/issues/33)) ([bb12472](https://github.com/muecahit94/terraform-provider-mssql/commit/bb124728e9b6262135daa6f0f0e5fe943053be72))

## [1.6.0](https://github.com/muecahit94/terraform-provider-mssql/compare/v1.5.0...v1.6.0) (2026-10-05)


### Features

* **login:** 29 - add per-resource server override and multi-server s… ([c62c155](https://github.com/muecahit94/terraform-provider-mssql/commit/c62c15504c6ae048b28dd530ef755eb5e0e4dcb8))
* **login:** 29 - add per-resource server override and multi-server support to sql login resource and provider ([0128426](https://github.com/muecahit94/terraform-provider-mssql/commit/0128426de36b99f2851746085d2260d92ca95ffc))


### Miscellaneous

* update Go dependencies and cleanup test whitespace ([4b78638](https://github.com/muecahit94/terraform-provider-mssql/commit/4b78638301f1b30b5d439eb2342f2d33470e1188))

## [1.5.0](https://github.com/muecahit94/terraform-provider-mssql/compare/v1.4.0...v1.5.0) (2026-08-30)


### Features

* **mssql_sql_login:** support ephemeral passwords via write-only password_wo ([0382900](https://github.com/muecahit94/terraform-provider-mssql/commit/03829000227c292c44e9de51219cfaa64949cd10))
* **mssql_sql_login:** support ephemeral passwords via write-only password_wo ([d8cca03](https://github.com/muecahit94/terraform-provider-mssql/commit/d8cca036ab1c4641f33b62bf365964896b93b6d5))


### Bug Fixes

* ensure write-only login state preservation and add E2E verification for password rotation behavior ([6acd7c6](https://github.com/muecahit94/terraform-provider-mssql/commit/6acd7c6833e0a26bb569b03c9457b888f232ee49))

## [1.4.0](https://github.com/muecahit94/terraform-provider-mssql/compare/v1.3.4...v1.4.0) (2026-08-19)


### Features

* **mssql_sql_login:** [#24](https://github.com/muecahit94/terraform-provider-mssql/issues/24) - support optional custom SID for SQL logins with normalization and state verification ([ae85f30](https://github.com/muecahit94/terraform-provider-mssql/commit/ae85f30cb644e3c899b3d924ce75824bc9904241))
* **mssql_sql_login:** [#24](https://github.com/muecahit94/terraform-provider-mssql/issues/24) - support optional custom SID for SQL logins with normalization and state verification ([dda7509](https://github.com/muecahit94/terraform-provider-mssql/commit/dda75093991a9ff413bd7254ce7fc693e833b086))


### Miscellaneous

* update go dependencies for x/text, genproto, grpc, and protobuf ([813e899](https://github.com/muecahit94/terraform-provider-mssql/commit/813e899f24329bdf5a8511998d7c7efe5d4ea3c2))

## [1.3.4](https://github.com/muecahit94/terraform-provider-mssql/compare/v1.3.3...v1.3.4) (2026-07-12)


### Bug Fixes

* **update:** upgrade Go version to 1.25.0 and update project dependen… ([36afb55](https://github.com/muecahit94/terraform-provider-mssql/commit/36afb55e357b0f0b02bdccef4c83678958f75e58))

## [1.3.3](https://github.com/muecahit94/terraform-provider-mssql/compare/v1.3.2...v1.3.3) (2026-07-12)


### Bug Fixes

* **mssql_script:** [#19](https://github.com/muecahit94/terraform-provider-mssql/issues/19) - ensure consistent connection handling during SQL script execution ([093b891](https://github.com/muecahit94/terraform-provider-mssql/commit/093b891172672031f255e7c3c3ee975962ceee44))
* **mssql_script:** [#19](https://github.com/muecahit94/terraform-provider-mssql/issues/19) - ensure consistent connection handling during SQL script execution ([2816a95](https://github.com/muecahit94/terraform-provider-mssql/commit/2816a95d2312745542b8911a27216824f5fdb87e))


### Miscellaneous

* **test:** [#19](https://github.com/muecahit94/terraform-provider-mssql/issues/19) - add regression test for connection drops and update E2E script to support multiple SQL CLI tools ([86b865d](https://github.com/muecahit94/terraform-provider-mssql/commit/86b865d4beb6a5450885c6bd00c87758a3eb43f1))

## [1.3.2](https://github.com/muecahit94/terraform-provider-mssql/compare/v1.3.1...v1.3.2) (2026-01-05)


### Miscellaneous

* support EXTERNAL_GROUP user type in user queries, implement state migration, and standardize Azure AD user ID format to a URL-based structure ([4571589](https://github.com/muecahit94/terraform-provider-mssql/commit/45715893ff5ea0c430463860824bbcd825f03afe))

## [1.3.1](https://github.com/muecahit94/terraform-provider-mssql/compare/v1.3.0...v1.3.1) (2026-01-05)


### Bug Fixes

* change azure AD authentication to use `azuresql` driver with `fedauth` parameters ([508cc15](https://github.com/muecahit94/terraform-provider-mssql/commit/508cc15a53d8b07a975a57c43753aa88bd613b3b))

## [1.3.0](https://github.com/muecahit94/terraform-provider-mssql/compare/v1.2.2...v1.3.0) (2026-01-04)


### Features

* Add `roles` attribute to `mssql_sql_user` and `mssql_azuread_user` for inline role assignment ([247911e](https://github.com/muecahit94/terraform-provider-mssql/commit/247911ea4cf8e044191b943c6ee5605a476b4c2b))

## [1.2.2](https://github.com/muecahit94/terraform-provider-mssql/compare/v1.2.1...v1.2.2) (2026-01-03)


### Bug Fixes

* make `object_id` optional for `mssql_azuread_user` to support email-based users via `FROM EXTERNAL PROVIDER` ([9921c2b](https://github.com/muecahit94/terraform-provider-mssql/commit/9921c2b23505e8b4620c4c510166a974e7cec072))

## [1.2.1](https://github.com/muecahit94/terraform-provider-mssql/compare/v1.2.0...v1.2.1) (2026-01-02)


### Bug Fixes

* Exclude ARM 32-bit builds for Windows, Darwin, and FreeBSD platforms ([e676e6e](https://github.com/muecahit94/terraform-provider-mssql/commit/e676e6e568427cdd494abdba81f8b139b151b0bf))

## [1.2.0](https://github.com/muecahit94/terraform-provider-mssql/compare/v1.1.0...v1.2.0) (2026-01-02)


### Features

* add 32-bit ARM (armv6, armv7) build support for Raspberry Pi ([0a40f1c](https://github.com/muecahit94/terraform-provider-mssql/commit/0a40f1ca001ed29e4cb425b081f1f6334f8737be))

## [1.1.0](https://github.com/muecahit94/terraform-provider-mssql/compare/v1.0.4...v1.1.0) (2026-01-01)


### Features

* Add and update data source docs and enhance existing resource/data source docs ([683b50c](https://github.com/muecahit94/terraform-provider-mssql/commit/683b50c24ee073d6bc93dbd3855d9cbf20f5fdb9))

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
