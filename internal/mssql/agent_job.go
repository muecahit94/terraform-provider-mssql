// Copyright (c) 2024 muecahit94
// SPDX-License-Identifier: MIT

package mssql

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
)

// AgentJobStep is a step of a SQL Server Agent job. The numeric fields use the codes of
// msdb.dbo.sp_add_jobstep.
type AgentJobStep struct {
	Name            string
	Subsystem       string
	Command         string
	DatabaseName    string
	OnSuccessAction int
	OnSuccessStepID int
	OnFailAction    int
	OnFailStepID    int
	RetryAttempts   int
	RetryInterval   int
}

// AgentJobSchedule is a schedule of a job. The numeric fields use the codes of
// msdb.dbo.sp_add_jobschedule: dates are yyyymmdd and times hhmmss.
type AgentJobSchedule struct {
	Name                 string
	Enabled              bool
	FreqType             int
	FreqInterval         int
	FreqSubdayType       int
	FreqSubdayInterval   int
	FreqRelativeInterval int
	FreqRecurrenceFactor int
	ActiveStartDate      int
	ActiveEndDate        int
	ActiveStartTime      int
	ActiveEndTime        int
}

// AgentJob is a SQL Server Agent job of the local server.
type AgentJob struct {
	ID             string
	Name           string
	Description    string
	Enabled        bool
	OwnerLoginName string
	CategoryName   string
	Steps          []AgentJobStep
	Schedules      []AgentJobSchedule
}

// GetAgentJob retrieves a job with its steps and schedules, or nil when it does not exist.
func (c *Client) GetAgentJob(ctx context.Context, name string) (*AgentJob, error) {
	var job AgentJob
	var jobID string
	err := c.QueryRowContext(ctx, `
		SELECT
			CONVERT(nvarchar(36), j.job_id),
			j.name,
			ISNULL(j.description, ''),
			j.enabled,
			ISNULL(SUSER_SNAME(j.owner_sid), ''),
			ISNULL(cat.name, '')
		FROM msdb.dbo.sysjobs j
		LEFT JOIN msdb.dbo.syscategories cat ON cat.category_id = j.category_id
		WHERE j.name = @p1`, name).Scan(&jobID, &job.Name, &job.Description, &job.Enabled, &job.OwnerLoginName, &job.CategoryName)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get agent job: %w", err)
	}
	job.ID = jobID

	steps, err := c.QueryContext(ctx, `
		SELECT
			step_name,
			subsystem,
			ISNULL(command, ''),
			ISNULL(database_name, ''),
			on_success_action,
			on_success_step_id,
			on_fail_action,
			on_fail_step_id,
			retry_attempts,
			retry_interval
		FROM msdb.dbo.sysjobsteps
		WHERE job_id = CONVERT(uniqueidentifier, @p1)
		ORDER BY step_id`, jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to get the steps of the agent job: %w", err)
	}
	defer steps.Close()
	for steps.Next() {
		var s AgentJobStep
		if err := steps.Scan(&s.Name, &s.Subsystem, &s.Command, &s.DatabaseName, &s.OnSuccessAction, &s.OnSuccessStepID,
			&s.OnFailAction, &s.OnFailStepID, &s.RetryAttempts, &s.RetryInterval); err != nil {
			return nil, fmt.Errorf("failed to scan an agent job step: %w", err)
		}
		job.Steps = append(job.Steps, s)
	}
	if err := steps.Err(); err != nil {
		return nil, err
	}

	schedules, err := c.QueryContext(ctx, `
		SELECT
			sc.name,
			sc.enabled,
			sc.freq_type,
			sc.freq_interval,
			sc.freq_subday_type,
			sc.freq_subday_interval,
			sc.freq_relative_interval,
			sc.freq_recurrence_factor,
			sc.active_start_date,
			sc.active_end_date,
			sc.active_start_time,
			sc.active_end_time
		FROM msdb.dbo.sysjobschedules js
		INNER JOIN msdb.dbo.sysschedules sc ON sc.schedule_id = js.schedule_id
		WHERE js.job_id = CONVERT(uniqueidentifier, @p1)
		ORDER BY sc.name`, jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to get the schedules of the agent job: %w", err)
	}
	defer schedules.Close()
	for schedules.Next() {
		var s AgentJobSchedule
		if err := schedules.Scan(&s.Name, &s.Enabled, &s.FreqType, &s.FreqInterval, &s.FreqSubdayType, &s.FreqSubdayInterval,
			&s.FreqRelativeInterval, &s.FreqRecurrenceFactor, &s.ActiveStartDate, &s.ActiveEndDate, &s.ActiveStartTime, &s.ActiveEndTime); err != nil {
			return nil, fmt.Errorf("failed to scan an agent job schedule: %w", err)
		}
		job.Schedules = append(job.Schedules, s)
	}
	return &job, schedules.Err()
}

// nullIfEmptyString maps an empty string to NULL, which makes the stored procedures use their defaults.
func nullIfEmptyString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

// addJobStepArgs returns the arguments of msdb.dbo.sp_add_jobstep, in the order of addJobStepSQL.
func addJobStepArgs(job string, id int, s AgentJobStep) []interface{} {
	return []interface{}{
		job, id, s.Name, s.Subsystem, s.Command, nullIfEmptyString(s.DatabaseName),
		s.OnSuccessAction, s.OnSuccessStepID, s.OnFailAction, s.OnFailStepID, s.RetryAttempts, s.RetryInterval,
	}
}

const addJobStepSQL = `EXEC msdb.dbo.sp_add_jobstep
	@job_name = @p1, @step_id = @p2, @step_name = @p3, @subsystem = @p4, @command = @p5, @database_name = @p6,
	@on_success_action = @p7, @on_success_step_id = @p8, @on_fail_action = @p9, @on_fail_step_id = @p10,
	@retry_attempts = @p11, @retry_interval = @p12`

// addJobScheduleArgs returns the arguments of msdb.dbo.sp_add_jobschedule, in the order of addJobScheduleSQL.
func addJobScheduleArgs(job string, s AgentJobSchedule) []interface{} {
	enabled := 0
	if s.Enabled {
		enabled = 1
	}
	return []interface{}{
		job, s.Name, enabled, s.FreqType, s.FreqInterval, s.FreqSubdayType, s.FreqSubdayInterval,
		s.FreqRelativeInterval, s.FreqRecurrenceFactor, s.ActiveStartDate, s.ActiveEndDate, s.ActiveStartTime, s.ActiveEndTime,
	}
}

const addJobScheduleSQL = `EXEC msdb.dbo.sp_add_jobschedule
	@job_name = @p1, @name = @p2, @enabled = @p3, @freq_type = @p4, @freq_interval = @p5, @freq_subday_type = @p6,
	@freq_subday_interval = @p7, @freq_relative_interval = @p8, @freq_recurrence_factor = @p9,
	@active_start_date = @p10, @active_end_date = @p11, @active_start_time = @p12, @active_end_time = @p13`

// CreateAgentJob creates a job with its steps and schedules on the local server.
func (c *Client) CreateAgentJob(ctx context.Context, job AgentJob) (*AgentJob, error) {
	enabled := 0
	if job.Enabled {
		enabled = 1
	}
	if _, err := c.ExecContext(ctx, `EXEC msdb.dbo.sp_add_job
		@job_name = @p1, @enabled = @p2, @description = @p3, @owner_login_name = @p4, @category_name = @p5`,
		job.Name, enabled, nullIfEmptyString(job.Description), nullIfEmptyString(job.OwnerLoginName), nullIfEmptyString(job.CategoryName)); err != nil {
		return nil, fmt.Errorf("failed to create the agent job: %w", err)
	}

	// Do not leave a half-built job behind.
	fail := func(err error) (*AgentJob, error) {
		_ = c.DeleteAgentJob(ctx, job.Name)
		return nil, err
	}
	if err := c.ReplaceAgentJobSteps(ctx, job.Name, job.Steps); err != nil {
		return fail(err)
	}
	if err := c.ReplaceAgentJobSchedules(ctx, job.Name, job.Schedules); err != nil {
		return fail(err)
	}
	// A job must be assigned to a server to run; this provider manages the local server.
	if _, err := c.ExecContext(ctx, "EXEC msdb.dbo.sp_add_jobserver @job_name = @p1, @server_name = N'(local)'", job.Name); err != nil {
		return fail(fmt.Errorf("failed to assign the agent job to the local server: %w", err))
	}
	return c.GetAgentJob(ctx, job.Name)
}

// UpdateAgentJobProperties changes the description, the enabled state, the owner and the category.
func (c *Client) UpdateAgentJobProperties(ctx context.Context, job AgentJob) error {
	enabled := 0
	if job.Enabled {
		enabled = 1
	}
	if _, err := c.ExecContext(ctx, `EXEC msdb.dbo.sp_update_job
		@job_name = @p1, @enabled = @p2, @description = @p3, @owner_login_name = @p4, @category_name = @p5`,
		job.Name, enabled, job.Description, nullIfEmptyString(job.OwnerLoginName), nullIfEmptyString(job.CategoryName)); err != nil {
		return fmt.Errorf("failed to update the agent job: %w", err)
	}
	return nil
}

// ReplaceAgentJobSteps replaces all steps of a job with the given ones, in order (step 1, 2, ...).
func (c *Client) ReplaceAgentJobSteps(ctx context.Context, jobName string, steps []AgentJobStep) error {
	// step_id 0 deletes all steps of the job.
	if _, err := c.ExecContext(ctx, "EXEC msdb.dbo.sp_delete_jobstep @job_name = @p1, @step_id = 0", jobName); err != nil {
		return fmt.Errorf("failed to delete the steps of the agent job: %w", err)
	}
	for i, step := range steps {
		if _, err := c.ExecContext(ctx, addJobStepSQL, addJobStepArgs(jobName, i+1, step)...); err != nil {
			return fmt.Errorf("failed to add step %q to the agent job: %w", step.Name, err)
		}
	}
	return nil
}

// ReplaceAgentJobSchedules replaces all schedules of a job with the given ones.
func (c *Client) ReplaceAgentJobSchedules(ctx context.Context, jobName string, schedules []AgentJobSchedule) error {
	current, err := c.GetAgentJob(ctx, jobName)
	if err != nil {
		return err
	}
	if current != nil {
		for _, existing := range current.Schedules {
			if _, err := c.ExecContext(ctx, "EXEC msdb.dbo.sp_detach_schedule @job_name = @p1, @schedule_name = @p2, @delete_unused_schedule = 1",
				jobName, existing.Name); err != nil {
				return fmt.Errorf("failed to remove the schedule %q of the agent job: %w", existing.Name, err)
			}
		}
	}

	// A stable order keeps the result independent of the order of a map.
	sorted := append([]AgentJobSchedule(nil), schedules...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Name < sorted[j].Name })
	for _, schedule := range sorted {
		if _, err := c.ExecContext(ctx, addJobScheduleSQL, addJobScheduleArgs(jobName, schedule)...); err != nil {
			return fmt.Errorf("failed to add the schedule %q to the agent job: %w", schedule.Name, err)
		}
	}
	return nil
}

// DeleteAgentJob deletes a job and the schedules that only it used.
func (c *Client) DeleteAgentJob(ctx context.Context, name string) error {
	if _, err := c.ExecContext(ctx, "EXEC msdb.dbo.sp_delete_job @job_name = @p1, @delete_unused_schedule = 1", name); err != nil {
		return fmt.Errorf("failed to delete the agent job: %w", err)
	}
	return nil
}
