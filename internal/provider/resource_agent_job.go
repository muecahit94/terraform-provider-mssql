// Copyright (c) 2024 muecahit94
// SPDX-License-Identifier: MIT

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/muecahit94/terraform-provider-mssql/internal/mssql"
)

var _ resource.Resource = &AgentJobResource{}
var _ resource.ResourceWithImportState = &AgentJobResource{}

// NewAgentJobResource creates a new SQL Server Agent job resource.
func NewAgentJobResource() resource.Resource {
	return &AgentJobResource{}
}

// AgentJobResource manages a SQL Server Agent job of the local server, with its steps and schedules.
type AgentJobResource struct {
	client *mssql.Client
}

// AgentJobStepModel is a step of a job.
type AgentJobStepModel struct {
	Name            types.String `tfsdk:"name"`
	Subsystem       types.String `tfsdk:"subsystem"`
	Command         types.String `tfsdk:"command"`
	DatabaseName    types.String `tfsdk:"database_name"`
	OnSuccessAction types.Int64  `tfsdk:"on_success_action"`
	OnSuccessStepID types.Int64  `tfsdk:"on_success_step_id"`
	OnFailAction    types.Int64  `tfsdk:"on_fail_action"`
	OnFailStepID    types.Int64  `tfsdk:"on_fail_step_id"`
	RetryAttempts   types.Int64  `tfsdk:"retry_attempts"`
	RetryInterval   types.Int64  `tfsdk:"retry_interval"`
}

// AgentJobScheduleModel is a schedule of a job; its name is the key of the map.
type AgentJobScheduleModel struct {
	Enabled              types.Bool  `tfsdk:"enabled"`
	FreqType             types.Int64 `tfsdk:"freq_type"`
	FreqInterval         types.Int64 `tfsdk:"freq_interval"`
	FreqSubdayType       types.Int64 `tfsdk:"freq_subday_type"`
	FreqSubdayInterval   types.Int64 `tfsdk:"freq_subday_interval"`
	FreqRelativeInterval types.Int64 `tfsdk:"freq_relative_interval"`
	FreqRecurrenceFactor types.Int64 `tfsdk:"freq_recurrence_factor"`
	ActiveStartDate      types.Int64 `tfsdk:"active_start_date"`
	ActiveEndDate        types.Int64 `tfsdk:"active_end_date"`
	ActiveStartTime      types.Int64 `tfsdk:"active_start_time"`
	ActiveEndTime        types.Int64 `tfsdk:"active_end_time"`
}

// AgentJobResourceModel describes the resource data model.
type AgentJobResourceModel struct {
	ID             types.String                     `tfsdk:"id"`
	Name           types.String                     `tfsdk:"name"`
	Description    types.String                     `tfsdk:"description"`
	Enabled        types.Bool                       `tfsdk:"enabled"`
	OwnerLoginName types.String                     `tfsdk:"owner_login_name"`
	CategoryName   types.String                     `tfsdk:"category_name"`
	Steps          []AgentJobStepModel              `tfsdk:"steps"`
	Schedules      map[string]AgentJobScheduleModel `tfsdk:"schedules"`
}

func (r *AgentJobResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_agent_job"
}

func intAttr(description string, def int64) schema.Int64Attribute {
	return schema.Int64Attribute{Description: description, Optional: true, Computed: true, Default: int64default.StaticInt64(def)}
}

func (r *AgentJobResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a SQL Server Agent job of the local server with its steps and schedules (`msdb.dbo.sp_add_job`, `sp_add_jobstep`, `sp_add_jobschedule`).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The job ID (a GUID).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the job. Changing this forces a new resource to be created.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				Description: "The description of the job. SQL Server uses `No description available.` when it is not set.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the job is enabled. Defaults to `true`.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"owner_login_name": schema.StringAttribute{
				Description: "The login that owns the job. Defaults to the login that creates it.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"category_name": schema.StringAttribute{
				Description: "The category of the job. Defaults to `[Uncategorized (Local)]`.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"steps": schema.ListNestedAttribute{
				Description: "The steps of the job, in the order they are listed: the first is step 1. " +
					"A change of any step replaces all steps of the job (the job itself and its history stay).",
				Optional: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{Description: "The name of the step.", Required: true},
						"subsystem": schema.StringAttribute{
							Description: "The subsystem that runs the command: `TSQL`, `CmdExec`, `PowerShell`, `SSIS`, ... Defaults to `TSQL`.",
							Optional:    true, Computed: true, Default: stringdefault.StaticString("TSQL"),
						},
						"command": schema.StringAttribute{Description: "The command: T-SQL for `TSQL`, a command line for `CmdExec`, a script for `PowerShell`.", Required: true},
						"database_name": schema.StringAttribute{
							Description: "The database a `TSQL` step runs in. SQL Server uses `master` when it is not set.",
							Optional:    true, Computed: true,
							PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
						},
						"on_success_action":  intAttr("What happens when the step succeeds: `1` quit with success, `2` quit with failure, `3` go to the next step, `4` go to `on_success_step_id`. Defaults to `1`.", 1),
						"on_success_step_id": intAttr("The step to go to when `on_success_action` is `4`. Defaults to `0`.", 0),
						"on_fail_action":     intAttr("What happens when the step fails, with the same codes as `on_success_action`. Defaults to `2`.", 2),
						"on_fail_step_id":    intAttr("The step to go to when `on_fail_action` is `4`. Defaults to `0`.", 0),
						"retry_attempts":     intAttr("How many times the step is retried when it fails. Defaults to `0`.", 0),
						"retry_interval":     intAttr("The minutes between retries. Defaults to `0`.", 0),
					},
				},
			},
			"schedules": schema.MapNestedAttribute{
				Description: "The schedules of the job, by name. The numbers are the codes of `msdb.dbo.sp_add_jobschedule`: " +
					"dates are `yyyymmdd`, times `hhmmss`. A change of any schedule replaces all schedules of the job.",
				Optional: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"enabled": schema.BoolAttribute{Description: "Whether the schedule is enabled. Defaults to `true`.", Optional: true, Computed: true, Default: booldefault.StaticBool(true)},
						"freq_type": schema.Int64Attribute{
							Description: "When the job runs: `1` once, `4` daily, `8` weekly, `16` monthly, `32` monthly relative to `freq_interval`, `64` when SQL Server Agent starts, `128` when the computer is idle.",
							Required:    true,
						},
						"freq_interval":          intAttr("The days it runs on. `4` daily: every N days. `8` weekly: a sum of the days (Sunday `1`, Monday `2`, Tuesday `4`, Wednesday `8`, Thursday `16`, Friday `32`, Saturday `64`; Monday to Friday is `62`). `16` monthly: the day of the month. Defaults to `1`.", 1),
						"freq_subday_type":       intAttr("The unit within a day: `1` at the given time, `2` seconds, `4` minutes, `8` hours. Defaults to `1`.", 1),
						"freq_subday_interval":   intAttr("How many `freq_subday_type` units between runs. Defaults to `0`.", 0),
						"freq_relative_interval": intAttr("For `freq_type` `32`: `1` first, `2` second, `4` third, `8` fourth, `16` last. Defaults to `0`.", 0),
						"freq_recurrence_factor": intAttr("How many weeks or months between runs, for weekly and monthly schedules. Defaults to `0`.", 0),
						"active_start_date": schema.Int64Attribute{
							Description: "The first day the schedule is active, as `yyyymmdd`. SQL Server uses today when it is not set.",
							Optional:    true, Computed: true,
							PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
						},
						"active_end_date":   intAttr("The last day the schedule is active, as `yyyymmdd`. Defaults to `99991231`.", 99991231),
						"active_start_time": intAttr("The time of day the schedule starts, as `hhmmss`. Defaults to `0`.", 0),
						"active_end_time":   intAttr("The time of day the schedule ends, as `hhmmss`. Defaults to `235959`.", 235959),
					},
				},
			},
		},
	}
}

func (r *AgentJobResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*mssql.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("Expected *mssql.Client, got: %T.", req.ProviderData))
		return
	}
	r.client = client
}

// jobFromModel converts the planned model into the client type. Values that are unknown (not configured and
// computed by SQL Server) become empty and are left to the server.
func jobFromModel(data AgentJobResourceModel) mssql.AgentJob {
	job := mssql.AgentJob{
		Name:           data.Name.ValueString(),
		Description:    data.Description.ValueString(),
		Enabled:        data.Enabled.ValueBool(),
		OwnerLoginName: data.OwnerLoginName.ValueString(),
		CategoryName:   data.CategoryName.ValueString(),
	}
	for _, s := range data.Steps {
		job.Steps = append(job.Steps, mssql.AgentJobStep{
			Name:            s.Name.ValueString(),
			Subsystem:       s.Subsystem.ValueString(),
			Command:         s.Command.ValueString(),
			DatabaseName:    s.DatabaseName.ValueString(),
			OnSuccessAction: int(s.OnSuccessAction.ValueInt64()),
			OnSuccessStepID: int(s.OnSuccessStepID.ValueInt64()),
			OnFailAction:    int(s.OnFailAction.ValueInt64()),
			OnFailStepID:    int(s.OnFailStepID.ValueInt64()),
			RetryAttempts:   int(s.RetryAttempts.ValueInt64()),
			RetryInterval:   int(s.RetryInterval.ValueInt64()),
		})
	}
	for name, s := range data.Schedules {
		job.Schedules = append(job.Schedules, mssql.AgentJobSchedule{
			Name:                 name,
			Enabled:              s.Enabled.ValueBool(),
			FreqType:             int(s.FreqType.ValueInt64()),
			FreqInterval:         int(s.FreqInterval.ValueInt64()),
			FreqSubdayType:       int(s.FreqSubdayType.ValueInt64()),
			FreqSubdayInterval:   int(s.FreqSubdayInterval.ValueInt64()),
			FreqRelativeInterval: int(s.FreqRelativeInterval.ValueInt64()),
			FreqRecurrenceFactor: int(s.FreqRecurrenceFactor.ValueInt64()),
			ActiveStartDate:      int(s.ActiveStartDate.ValueInt64()),
			ActiveEndDate:        int(s.ActiveEndDate.ValueInt64()),
			ActiveStartTime:      int(s.ActiveStartTime.ValueInt64()),
			ActiveEndTime:        int(s.ActiveEndTime.ValueInt64()),
		})
	}
	return job
}

// applyAgentJob copies what the server has into the model. An empty list of steps or schedules is null, so
// that a configuration that does not set them shows no diff.
func applyAgentJob(data *AgentJobResourceModel, job *mssql.AgentJob) {
	data.ID = types.StringValue(job.ID)
	data.Name = types.StringValue(job.Name)
	data.Description = types.StringValue(job.Description)
	data.Enabled = types.BoolValue(job.Enabled)
	data.OwnerLoginName = types.StringValue(job.OwnerLoginName)
	data.CategoryName = types.StringValue(job.CategoryName)

	data.Steps = nil
	for _, s := range job.Steps {
		data.Steps = append(data.Steps, AgentJobStepModel{
			Name:            types.StringValue(s.Name),
			Subsystem:       types.StringValue(s.Subsystem),
			Command:         types.StringValue(s.Command),
			DatabaseName:    types.StringValue(s.DatabaseName),
			OnSuccessAction: types.Int64Value(int64(s.OnSuccessAction)),
			OnSuccessStepID: types.Int64Value(int64(s.OnSuccessStepID)),
			OnFailAction:    types.Int64Value(int64(s.OnFailAction)),
			OnFailStepID:    types.Int64Value(int64(s.OnFailStepID)),
			RetryAttempts:   types.Int64Value(int64(s.RetryAttempts)),
			RetryInterval:   types.Int64Value(int64(s.RetryInterval)),
		})
	}

	data.Schedules = nil
	for _, s := range job.Schedules {
		if data.Schedules == nil {
			data.Schedules = map[string]AgentJobScheduleModel{}
		}
		data.Schedules[s.Name] = AgentJobScheduleModel{
			Enabled:              types.BoolValue(s.Enabled),
			FreqType:             types.Int64Value(int64(s.FreqType)),
			FreqInterval:         types.Int64Value(int64(s.FreqInterval)),
			FreqSubdayType:       types.Int64Value(int64(s.FreqSubdayType)),
			FreqSubdayInterval:   types.Int64Value(int64(s.FreqSubdayInterval)),
			FreqRelativeInterval: types.Int64Value(int64(s.FreqRelativeInterval)),
			FreqRecurrenceFactor: types.Int64Value(int64(s.FreqRecurrenceFactor)),
			ActiveStartDate:      types.Int64Value(int64(s.ActiveStartDate)),
			ActiveEndDate:        types.Int64Value(int64(s.ActiveEndDate)),
			ActiveStartTime:      types.Int64Value(int64(s.ActiveStartTime)),
			ActiveEndTime:        types.Int64Value(int64(s.ActiveEndTime)),
		}
	}
}

func (r *AgentJobResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data AgentJobResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	job, err := r.client.CreateAgentJob(ctx, jobFromModel(data))
	if err != nil {
		resp.Diagnostics.AddError("Failed to create the agent job", err.Error())
		return
	}
	if job == nil {
		resp.Diagnostics.AddError("Failed to create the agent job", "The job was not found after creation.")
		return
	}

	applyAgentJob(&data, job)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AgentJobResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data AgentJobResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	job, err := r.client.GetAgentJob(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read the agent job", err.Error())
		return
	}
	if job == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	applyAgentJob(&data, job)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AgentJobResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state AgentJobResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Properties not configured keep the value the job has.
	if plan.Description.IsUnknown() {
		plan.Description = state.Description
	}
	if plan.OwnerLoginName.IsUnknown() {
		plan.OwnerLoginName = state.OwnerLoginName
	}
	if plan.CategoryName.IsUnknown() {
		plan.CategoryName = state.CategoryName
	}

	job := jobFromModel(plan)
	name := state.Name.ValueString()

	if err := r.client.UpdateAgentJobProperties(ctx, job); err != nil {
		resp.Diagnostics.AddError("Failed to update the agent job", err.Error())
		return
	}
	if !stepsEqual(plan.Steps, state.Steps) {
		if err := r.client.ReplaceAgentJobSteps(ctx, name, job.Steps); err != nil {
			resp.Diagnostics.AddError("Failed to update the steps of the agent job", err.Error())
			return
		}
	}
	if !schedulesEqual(plan.Schedules, state.Schedules) {
		if err := r.client.ReplaceAgentJobSchedules(ctx, name, job.Schedules); err != nil {
			resp.Diagnostics.AddError("Failed to update the schedules of the agent job", err.Error())
			return
		}
	}

	updated, err := r.client.GetAgentJob(ctx, name)
	if err != nil || updated == nil {
		resp.Diagnostics.AddError("Failed to read the agent job after the update", fmt.Sprint(err))
		return
	}

	applyAgentJob(&plan, updated)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AgentJobResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data AgentJobResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteAgentJob(ctx, data.Name.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to delete the agent job", err.Error())
	}
}

func (r *AgentJobResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	job, err := r.client.GetAgentJob(ctx, req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to import the agent job", err.Error())
		return
	}
	if job == nil {
		resp.Diagnostics.AddError("Agent job not found", fmt.Sprintf("No agent job named %q.", req.ID))
		return
	}

	var data AgentJobResourceModel
	applyAgentJob(&data, job)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// stepsEqual compares two lists of steps, treating unknown values as equal to anything: they are filled in by the server.
func stepsEqual(a, b []AgentJobStepModel) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		x, y := a[i], b[i]
		if !x.Name.Equal(y.Name) || !x.Subsystem.Equal(y.Subsystem) || !x.Command.Equal(y.Command) ||
			!(x.DatabaseName.IsUnknown() || x.DatabaseName.Equal(y.DatabaseName)) ||
			!x.OnSuccessAction.Equal(y.OnSuccessAction) || !x.OnSuccessStepID.Equal(y.OnSuccessStepID) ||
			!x.OnFailAction.Equal(y.OnFailAction) || !x.OnFailStepID.Equal(y.OnFailStepID) ||
			!x.RetryAttempts.Equal(y.RetryAttempts) || !x.RetryInterval.Equal(y.RetryInterval) {
			return false
		}
	}
	return true
}

// schedulesEqual compares two maps of schedules, treating an unknown start date as equal to anything.
func schedulesEqual(a, b map[string]AgentJobScheduleModel) bool {
	if len(a) != len(b) {
		return false
	}
	for name, x := range a {
		y, ok := b[name]
		if !ok {
			return false
		}
		if !x.Enabled.Equal(y.Enabled) || !x.FreqType.Equal(y.FreqType) || !x.FreqInterval.Equal(y.FreqInterval) ||
			!x.FreqSubdayType.Equal(y.FreqSubdayType) || !x.FreqSubdayInterval.Equal(y.FreqSubdayInterval) ||
			!x.FreqRelativeInterval.Equal(y.FreqRelativeInterval) || !x.FreqRecurrenceFactor.Equal(y.FreqRecurrenceFactor) ||
			!(x.ActiveStartDate.IsUnknown() || x.ActiveStartDate.Equal(y.ActiveStartDate)) ||
			!x.ActiveEndDate.Equal(y.ActiveEndDate) || !x.ActiveStartTime.Equal(y.ActiveStartTime) || !x.ActiveEndTime.Equal(y.ActiveEndTime) {
			return false
		}
	}
	return true
}
