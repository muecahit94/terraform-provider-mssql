// Copyright (c) 2024 muecahit94
// SPDX-License-Identifier: MIT

package provider

import (
	"context"
	"testing"

	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestDatabaseResourceSchema(t *testing.T) {
	ctx := context.Background()
	resp := &fwresource.SchemaResponse{}

	NewDatabaseResource().Schema(ctx, fwresource.SchemaRequest{}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema() returned errors: %v", resp.Diagnostics.Errors())
	}
	if diags := resp.Schema.ValidateImplementation(ctx); diags.HasError() {
		t.Errorf("ValidateImplementation() returned errors: %v", diags.Errors())
	}
	for _, name := range []string{"collation", "compatibility_level", "recovery_model"} {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Fatalf("attribute %q is missing", name)
		}
		// Optional+Computed keeps existing configurations free of a diff.
		if !attr.IsOptional() || !attr.IsComputed() {
			t.Errorf("attribute %q must be Optional and Computed", name)
		}
	}
}

// databaseObject builds a raw value for the database resource schema.
func databaseObject(t *testing.T, schemaType tftypes.Type, name string, collation, level, recovery interface{}) tftypes.Value {
	t.Helper()
	objType := schemaType.(tftypes.Object)
	values := map[string]tftypes.Value{}
	for attr, typ := range objType.AttributeTypes {
		values[attr] = tftypes.NewValue(typ, nil)
	}
	values["id"] = tftypes.NewValue(tftypes.String, "7")
	values["name"] = tftypes.NewValue(tftypes.String, name)
	values["collation"] = tftypes.NewValue(tftypes.String, collation)
	values["compatibility_level"] = tftypes.NewValue(tftypes.Number, level)
	values["recovery_model"] = tftypes.NewValue(tftypes.String, recovery)
	return tftypes.NewValue(objType, values)
}

func TestDatabaseValidateConfig(t *testing.T) {
	ctx := context.Background()
	schemaResp := &fwresource.SchemaResponse{}
	r := NewDatabaseResource()
	r.Schema(ctx, fwresource.SchemaRequest{}, schemaResp)
	tfType := schemaResp.Schema.Type().TerraformType(ctx)

	tests := []struct {
		name      string
		level     interface{}
		recovery  interface{}
		wantError bool
	}{
		{"nothing set", nil, nil, false},
		{"valid", 160, "SIMPLE", false},
		{"bulk logged", nil, "BULK_LOGGED", false},
		{"lower case recovery model", nil, "simple", true},
		{"unknown recovery model", nil, "LOGGED", true},
		{"zero level", 0, nil, true},
		{"negative level", -1, nil, true},
		{"unknown recovery model value is not rejected", nil, tftypes.UnknownValue, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := fwresource.ValidateConfigRequest{
				Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: databaseObject(t, tfType, "db", nil, tt.level, tt.recovery)},
			}
			resp := &fwresource.ValidateConfigResponse{}
			r.(fwresource.ResourceWithValidateConfig).ValidateConfig(ctx, req, resp)

			if resp.Diagnostics.HasError() != tt.wantError {
				t.Errorf("ValidateConfig() error = %v, wantError %v", resp.Diagnostics.Errors(), tt.wantError)
			}
		})
	}
}

func TestDatabaseModifyPlanCollation(t *testing.T) {
	ctx := context.Background()
	schemaResp := &fwresource.SchemaResponse{}
	r := NewDatabaseResource()
	r.Schema(ctx, fwresource.SchemaRequest{}, schemaResp)
	tfType := schemaResp.Schema.Type().TerraformType(ctx)

	const current = "Ukrainian_CI_AS"
	state := databaseObject(t, tfType, "db", current, 160, "SIMPLE")

	tests := []struct {
		name      string
		planned   interface{}
		wantError bool
	}{
		{"unchanged", current, false},
		{"same name in another case", "ukrainian_ci_as", false},
		{"changed", "SQL_Latin1_General_CP1_CI_AS", true},
		{"unknown", tftypes.UnknownValue, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := fwresource.ModifyPlanRequest{
				State: tfsdk.State{Schema: schemaResp.Schema, Raw: state},
				Plan:  tfsdk.Plan{Schema: schemaResp.Schema, Raw: databaseObject(t, tfType, "db", tt.planned, 160, "SIMPLE")},
			}
			resp := &fwresource.ModifyPlanResponse{Plan: req.Plan}
			r.(fwresource.ResourceWithModifyPlan).ModifyPlan(ctx, req, resp)

			if resp.Diagnostics.HasError() != tt.wantError {
				t.Errorf("ModifyPlan() error = %v, wantError %v", resp.Diagnostics.Errors(), tt.wantError)
			}
		})
	}

	t.Run("create is not blocked", func(t *testing.T) {
		req := fwresource.ModifyPlanRequest{
			State: tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(tfType, nil)},
			Plan:  tfsdk.Plan{Schema: schemaResp.Schema, Raw: databaseObject(t, tfType, "db", "Ukrainian_CI_AS", 160, "SIMPLE")},
		}
		resp := &fwresource.ModifyPlanResponse{Plan: req.Plan}
		r.(fwresource.ResourceWithModifyPlan).ModifyPlan(ctx, req, resp)
		if resp.Diagnostics.HasError() {
			t.Errorf("ModifyPlan() on create returned errors: %v", resp.Diagnostics.Errors())
		}
	})
}

func TestKeepCase(t *testing.T) {
	tests := []struct {
		name       string
		configured types.String
		actual     string
		want       string
	}{
		{"same", types.StringValue("Ukrainian_CI_AS"), "Ukrainian_CI_AS", "Ukrainian_CI_AS"},
		{"configured in another case keeps the configured spelling", types.StringValue("ukrainian_ci_as"), "Ukrainian_CI_AS", "ukrainian_ci_as"},
		{"different value takes the server value", types.StringValue("Latin1_General_CI_AS"), "Ukrainian_CI_AS", "Ukrainian_CI_AS"},
		{"null takes the server value", types.StringNull(), "Ukrainian_CI_AS", "Ukrainian_CI_AS"},
		{"unknown takes the server value", types.StringUnknown(), "Ukrainian_CI_AS", "Ukrainian_CI_AS"},
	}
	for _, tt := range tests {
		if got := keepCase(tt.configured, tt.actual); got.ValueString() != tt.want {
			t.Errorf("%s: keepCase() = %q, want %q", tt.name, got.ValueString(), tt.want)
		}
	}
}
