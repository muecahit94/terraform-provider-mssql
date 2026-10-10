// Copyright (c) 2024 muecahit94
// SPDX-License-Identifier: MIT

package provider

import (
	"context"
	"testing"

	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/muecahit94/terraform-provider-mssql/internal/mssql"
)

func TestAgentJobResourceSchema(t *testing.T) {
	ctx := context.Background()
	resp := &fwresource.SchemaResponse{}
	NewAgentJobResource().Schema(ctx, fwresource.SchemaRequest{}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema() returned errors: %v", resp.Diagnostics.Errors())
	}
	if diags := resp.Schema.ValidateImplementation(ctx); diags.HasError() {
		t.Errorf("ValidateImplementation() returned errors: %v", diags.Errors())
	}
	if !resp.Schema.Attributes["name"].IsRequired() {
		t.Error("name must be required")
	}
	for _, name := range []string{"steps", "schedules"} {
		if !resp.Schema.Attributes[name].IsOptional() {
			t.Errorf("%s must be optional: a job may have neither", name)
		}
	}
	steps := resp.Schema.Attributes["steps"].(schema.ListNestedAttribute)
	if !steps.NestedObject.Attributes["command"].IsSensitive() {
		t.Error("a step command can carry credentials and must be sensitive")
	}
}

func TestAgentJobValidateConfig(t *testing.T) {
	ctx := context.Background()
	schemaResp := &fwresource.SchemaResponse{}
	r := NewAgentJobResource()
	r.Schema(ctx, fwresource.SchemaRequest{}, schemaResp)
	objType := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
	schedType := objType.AttributeTypes["schedules"].(tftypes.Map).ElementType.(tftypes.Object)

	schedule := func(freqType, recurrence interface{}) tftypes.Value {
		values := map[string]tftypes.Value{}
		for n, typ := range schedType.AttributeTypes {
			values[n] = tftypes.NewValue(typ, nil)
		}
		values["freq_type"] = tftypes.NewValue(tftypes.Number, freqType)
		values["freq_recurrence_factor"] = tftypes.NewValue(tftypes.Number, recurrence)
		return tftypes.NewValue(schedType, values)
	}

	tests := []struct {
		name       string
		freqType   interface{}
		recurrence interface{}
		wantError  bool
	}{
		{"daily without recurrence", 4, nil, false},
		{"weekly without recurrence", 8, nil, true},
		{"weekly with recurrence 0", 8, 0, true},
		{"weekly every week", 8, 1, false},
		{"monthly without recurrence", 16, nil, true},
		{"monthly relative every 2 months", 32, 2, false},
		{"unknown frequency", tftypes.UnknownValue, nil, false},
	}
	for _, tt := range tests {
		values := map[string]tftypes.Value{}
		for n, typ := range objType.AttributeTypes {
			values[n] = tftypes.NewValue(typ, nil)
		}
		values["name"] = tftypes.NewValue(tftypes.String, "job")
		values["schedules"] = tftypes.NewValue(objType.AttributeTypes["schedules"], map[string]tftypes.Value{"s": schedule(tt.freqType, tt.recurrence)})
		req := fwresource.ValidateConfigRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: tftypes.NewValue(objType, values)}}
		resp := &fwresource.ValidateConfigResponse{}
		r.(fwresource.ResourceWithValidateConfig).ValidateConfig(ctx, req, resp)
		if resp.Diagnostics.HasError() != tt.wantError {
			t.Errorf("%s: error = %v, wantError %v", tt.name, resp.Diagnostics.Errors(), tt.wantError)
		}
	}
}

func TestApplyAgentJobAndBack(t *testing.T) {
	job := &mssql.AgentJob{
		ID: "11111111-2222-3333-4444-555555555555", Name: "nightly", Description: "d", Enabled: true, OwnerLoginName: "sa", CategoryName: "[Uncategorized (Local)]",
		Steps: []mssql.AgentJobStep{
			{Name: "one", Subsystem: "TSQL", Command: "SELECT 1", DatabaseName: "master", OnSuccessAction: 3, OnFailAction: 2},
			{Name: "two", Subsystem: "CmdExec", Command: "dir", OnSuccessAction: 1, OnFailAction: 2, RetryAttempts: 1, RetryInterval: 2},
		},
		Schedules: []mssql.AgentJobSchedule{
			{Name: "b", Enabled: true, FreqType: 4, FreqInterval: 1, FreqSubdayType: 1, ActiveStartDate: 20260101, ActiveEndDate: 99991231, ActiveEndTime: 235959},
			{Name: "a", FreqType: 8, FreqInterval: 62, FreqRecurrenceFactor: 1, ActiveStartDate: 20260101, ActiveEndDate: 99991231, ActiveStartTime: 90000, ActiveEndTime: 235959},
		},
	}

	var data AgentJobResourceModel
	applyAgentJob(&data, job)
	if len(data.Steps) != 2 || data.Steps[0].Name.ValueString() != "one" || data.Steps[1].Subsystem.ValueString() != "CmdExec" {
		t.Errorf("steps = %+v", data.Steps)
	}
	if len(data.Schedules) != 2 || data.Schedules["a"].FreqInterval.ValueInt64() != 62 || !data.Schedules["b"].Enabled.ValueBool() {
		t.Errorf("schedules = %+v", data.Schedules)
	}

	// The way back keeps the steps in order and every schedule under its name.
	back := jobFromModel(data)
	if back.Name != "nightly" || len(back.Steps) != 2 || back.Steps[0].Name != "one" || back.Steps[1].RetryInterval != 2 {
		t.Errorf("jobFromModel() steps = %+v", back.Steps)
	}
	names := map[string]int{}
	for _, s := range back.Schedules {
		names[s.Name] = s.FreqType
	}
	if names["a"] != 8 || names["b"] != 4 || len(names) != 2 {
		t.Errorf("jobFromModel() schedules = %+v", back.Schedules)
	}

	// A job without steps or schedules reads back as null, so a configuration without them shows no diff.
	var empty AgentJobResourceModel
	applyAgentJob(&empty, &mssql.AgentJob{ID: "x", Name: "empty", Enabled: true})
	if empty.Steps != nil || empty.Schedules != nil {
		t.Errorf("an empty job must have null steps and schedules, got %v %v", empty.Steps, empty.Schedules)
	}
}

func TestStepsAndSchedulesEqual(t *testing.T) {
	step := func(command string) AgentJobStepModel {
		return AgentJobStepModel{
			Name: types.StringValue("s"), Subsystem: types.StringValue("TSQL"), Command: types.StringValue(command), DatabaseName: types.StringValue("master"),
			OnSuccessAction: types.Int64Value(1), OnSuccessStepID: types.Int64Value(0), OnFailAction: types.Int64Value(2), OnFailStepID: types.Int64Value(0),
			RetryAttempts: types.Int64Value(0), RetryInterval: types.Int64Value(0),
		}
	}
	if !stepsEqual([]AgentJobStepModel{step("a")}, []AgentJobStepModel{step("a")}) {
		t.Error("equal steps must be equal")
	}
	if stepsEqual([]AgentJobStepModel{step("a")}, []AgentJobStepModel{step("b")}) {
		t.Error("a changed command must differ")
	}
	if stepsEqual([]AgentJobStepModel{step("a")}, nil) {
		t.Error("a missing step must differ")
	}
	unknownDB := step("a")
	unknownDB.DatabaseName = types.StringUnknown()
	if !stepsEqual([]AgentJobStepModel{unknownDB}, []AgentJobStepModel{step("a")}) {
		t.Error("a database the server fills in must not count as a change")
	}

	sched := AgentJobScheduleModel{Enabled: types.BoolValue(true), FreqType: types.Int64Value(4), FreqInterval: types.Int64Value(1), FreqSubdayType: types.Int64Value(1),
		FreqSubdayInterval: types.Int64Value(0), FreqRelativeInterval: types.Int64Value(0), FreqRecurrenceFactor: types.Int64Value(0), ActiveStartDate: types.Int64Value(20260101),
		ActiveEndDate: types.Int64Value(99991231), ActiveStartTime: types.Int64Value(0), ActiveEndTime: types.Int64Value(235959)}
	other := sched
	other.FreqType = types.Int64Value(8)
	unknownDate := sched
	unknownDate.ActiveStartDate = types.Int64Unknown()
	if !schedulesEqual(map[string]AgentJobScheduleModel{"x": sched}, map[string]AgentJobScheduleModel{"x": sched}) {
		t.Error("equal schedules must be equal")
	}
	if schedulesEqual(map[string]AgentJobScheduleModel{"x": sched}, map[string]AgentJobScheduleModel{"x": other}) {
		t.Error("another frequency must differ")
	}
	if schedulesEqual(map[string]AgentJobScheduleModel{"x": sched}, map[string]AgentJobScheduleModel{"y": sched}) {
		t.Error("another name must differ")
	}
	if !schedulesEqual(map[string]AgentJobScheduleModel{"x": unknownDate}, map[string]AgentJobScheduleModel{"x": sched}) {
		t.Error("a start date the server fills in must not count as a change")
	}
}
