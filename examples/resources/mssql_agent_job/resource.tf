resource "mssql_agent_job" "nightly_import" {
  name        = "nightly_import"
  description = "Loads the files of the day"

  steps = [
    {
      name          = "load"
      command       = "EXEC dbo.usp_LoadFiles"
      database_name = "app"
      # go to the next step on success; retry twice, five minutes apart
      on_success_action = 3
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
