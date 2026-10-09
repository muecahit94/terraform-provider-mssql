---
page_title: "mssql_agent_job Resource - terraform-provider-mssql"
subcategory: ""
description: |-
  Manages a SQL Server Agent job with its steps and schedules.
---

# mssql_agent_job (Resource)

Manages a SQL Server Agent job of the local server with its steps and schedules, through `msdb.dbo.sp_add_job`,
`sp_add_jobstep`, `sp_add_jobschedule` and `sp_add_jobserver`. The job is assigned to the local server.

The steps and the schedules use the **codes of those stored procedures**, so every kind of step and schedule can be
described, not only the common ones. The tables below list the values.

## Example Usage

```hcl
resource "mssql_agent_job" "nightly_import" {
  name        = "nightly_import"
  description = "Loads the files of the day"

  steps = [
    {
      name              = "load"
      command           = "EXEC dbo.usp_LoadFiles"
      database_name     = "app"
      on_success_action = 3 # go to the next step
      retry_attempts    = 2
      retry_interval    = 5
    },
    {
      name      = "archive"
      subsystem = "CmdExec"
      command   = "robocopy D:\\in D:\\archive /MOVE"
    },
  ]

  schedules = {
    # Monday to Friday at 02:00
    "weekdays 02:00" = {
      freq_type              = 8
      freq_interval          = 62
      freq_recurrence_factor = 1
      active_start_time      = 20000
    }
    # every 30 minutes, every day
    "every 30 minutes" = {
      freq_type            = 4
      freq_subday_type     = 4
      freq_subday_interval = 30
    }
  }
}
```

## Argument Reference

- `name` - (Required) The name of the job. Changing this forces a new resource.
- `description` - (Optional) The description. SQL Server uses `No description available.` when it is not set.
- `enabled` - (Optional) Whether the job is enabled. Defaults to `true`.
- `owner_login_name` - (Optional) The login that owns the job. Defaults to the login that creates it.
- `category_name` - (Optional) The category. Defaults to `[Uncategorized (Local)]`.
- `steps` - (Optional) The steps, in order: the first is step 1. A change of any step replaces **all** steps of the job; the job and its history stay. Each step has:
  - `name` - (Required) The name of the step.
  - `subsystem` - (Optional) `TSQL` (default), `CmdExec`, `PowerShell`, `SSIS`, ...
  - `command` - (Required) T-SQL, a command line or a script, depending on the subsystem.
  - `database_name` - (Optional) The database a `TSQL` step runs in; SQL Server uses `master` when it is not set.
  - `on_success_action` / `on_fail_action` - (Optional) `1` quit with success, `2` quit with failure, `3` go to the next step, `4` go to `on_success_step_id` / `on_fail_step_id`. Default `1` and `2`.
  - `on_success_step_id` / `on_fail_step_id` - (Optional) The step to go to for action `4`. Default `0`.
  - `retry_attempts` - (Optional) How often the step is retried. Default `0`.
  - `retry_interval` - (Optional) The minutes between retries. Default `0`.
- `schedules` - (Optional) The schedules, by name. A change of any schedule replaces **all** schedules of the job. Each schedule has:
  - `enabled` - (Optional) Default `true`.
  - `freq_type` - (Required) `1` once, `4` daily, `8` weekly, `16` monthly, `32` monthly relative to `freq_interval`, `64` when SQL Server Agent starts, `128` when the computer is idle.
  - `freq_interval` - (Optional) Which days: for `4` every N days; for `8` a sum of the days (Sunday `1`, Monday `2`, Tuesday `4`, Wednesday `8`, Thursday `16`, Friday `32`, Saturday `64`; Monday to Friday is `62`); for `16` the day of the month. Default `1`.
  - `freq_subday_type` - (Optional) The unit within a day: `1` at the given time, `2` seconds, `4` minutes, `8` hours. Default `1`.
  - `freq_subday_interval` - (Optional) How many `freq_subday_type` units between runs. Default `0`.
  - `freq_relative_interval` - (Optional) For `freq_type` `32`: `1` first, `2` second, `4` third, `8` fourth, `16` last. Default `0`.
  - `freq_recurrence_factor` - (Optional) How many weeks or months between runs. Default `0`.
  - `active_start_date` / `active_end_date` - (Optional) The days the schedule is active, as `yyyymmdd`. The start defaults to today, the end to `99991231`.
  - `active_start_time` / `active_end_time` - (Optional) The time of day the schedule starts and ends, as `hhmmss`. Defaults `0` and `235959`.

## Attribute Reference

- `id` - The job ID (a GUID).

## Behaviour to know

- Deleting the resource deletes the job and the schedules only it used.
- Schedules shared with other jobs are read like any other schedule, but replacing the schedules of a job detaches them (and deletes them only when no other job uses them).
- Alerts, operator notifications, proxies and job server targets other than the local server are not managed.

## Import

Jobs can be imported using the name:

```shell
terraform import mssql_agent_job.nightly_import nightly_import
```

The steps and schedules are read from the server, so an imported job can be described in the configuration by copying them from the state.
