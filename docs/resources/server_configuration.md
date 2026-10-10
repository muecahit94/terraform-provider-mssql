---
page_title: "mssql_server_configuration Resource - terraform-provider-mssql"
subcategory: ""
description: |-
  Manages one server configuration option (sp_configure).
---

# mssql_server_configuration (Resource)

Manages one server configuration option, the ones `sp_configure` lists and `sys.configurations` shows: for example
`max server memory (MB)`, `max degree of parallelism`, `clr enabled` or `xp_cmdshell`.

The change is applied with `sp_configure` and `RECONFIGURE`. An **advanced** option can only be changed while
`show advanced options` is on, so when it is off the provider switches it on for the change and off again.

## Example Usage

```hcl
# Limit the memory of SQL Server
resource "mssql_server_configuration" "max_memory" {
  name  = "max server memory (MB)"
  value = 8192
}

# Keep the operating system shell closed to T-SQL
resource "mssql_server_configuration" "xp_cmdshell" {
  name  = "xp_cmdshell"
  value = 0
}
```

## Argument Reference

- `name` - (Required) The option, spelled as in `sys.configurations` (for example `max degree of parallelism`). Changing this forces a new resource.
- `value` - (Required) The value. It is checked against the minimum and maximum the server reports for the option before anything runs.

## Attribute Reference

- `id` - The name of the option.
- `previous_value` - The value the option had when the resource took it over.
- `value_in_use` - The value the running server uses.
- `restart_required` - Whether the configured value takes effect only after the server is restarted (an option that is not dynamic).

## Behaviour to know

- **Destroying the resource restores `previous_value`**, the value the option had before Terraform changed it. For an
  imported option that is the value at the time of the import, so destroying it changes nothing.
- The configured value (`value`) is compared with the configuration, so a change made outside Terraform shows as a diff.
- Some options need a restart: the configured value is stored, `value_in_use` keeps the old one and `restart_required` is `true`.
- Changing server options needs the `ALTER SETTINGS` server permission (or `sysadmin`). Several options are security settings
  (`xp_cmdshell`, `Ole Automation Procedures`, `clr enabled`): review a plan for them as carefully as a change of permissions.

## Import

Options can be imported using the name:

```shell
terraform import mssql_server_configuration.max_memory 'max server memory (MB)'
```
