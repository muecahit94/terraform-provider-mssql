// Copyright (c) 2024 muecahit94
// SPDX-License-Identifier: MIT

package provider

import (
	"context"
	"testing"

	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestSQLUserResourceSchema(t *testing.T) {
	ctx := context.Background()
	resp := &fwresource.SchemaResponse{}

	NewSQLUserResource().Schema(ctx, fwresource.SchemaRequest{}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema() returned errors: %v", resp.Diagnostics.Errors())
	}
	if diags := resp.Schema.ValidateImplementation(ctx); diags.HasError() {
		t.Errorf("ValidateImplementation() returned errors: %v", diags.Errors())
	}

	// A user without a login (including an orphaned user) has no login_name.
	login := resp.Schema.Attributes["login_name"]
	if login.IsRequired() || !login.IsOptional() {
		t.Error("login_name must be optional")
	}
}

func TestLoginNameValue(t *testing.T) {
	if got := loginNameValue(""); !got.IsNull() {
		t.Errorf("loginNameValue(\"\") = %v, want null", got)
	}
	if got := loginNameValue(`DOMAIN\user`); got.ValueString() != `DOMAIN\user` {
		t.Errorf("loginNameValue() = %v", got)
	}
}

func TestLoginNameReplaceModifier(t *testing.T) {
	ctx := context.Background()
	objType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{"x": tftypes.String}}
	existing := tftypes.NewValue(objType, map[string]tftypes.Value{"x": tftypes.NewValue(tftypes.String, "v")})
	missing := tftypes.NewValue(objType, nil)

	tests := []struct {
		name        string
		state, plan tftypes.Value
		stateLogin  types.String
		planLogin   types.String
		wantReplace bool
	}{
		{"create", missing, existing, types.StringNull(), types.StringValue("l1"), false},
		{"destroy", existing, missing, types.StringValue("l1"), types.StringNull(), false},
		{"unchanged", existing, existing, types.StringValue("l1"), types.StringValue("l1"), false},
		{"another login: mapped in place", existing, existing, types.StringValue("l1"), types.StringValue("l2"), false},
		{"login removed: a mapped user cannot lose it", existing, existing, types.StringValue("l1"), types.StringNull(), true},
		{"unknown planned value", existing, existing, types.StringValue("l1"), types.StringUnknown(), false},
		// Whether a user without a login can be relinked depends on its authentication type, which is
		// read from the private state: with none known the plan does not replace it.
		{"no login to a login, type unknown", existing, existing, types.StringNull(), types.StringValue("l1"), false},
	}
	for _, tt := range tests {
		req := planmodifier.StringRequest{
			State:      tfsdk.State{Raw: tt.state},
			Plan:       tfsdk.Plan{Raw: tt.plan},
			StateValue: tt.stateLogin,
			PlanValue:  tt.planLogin,
		}
		resp := &planmodifier.StringResponse{}
		loginNameReplaceModifier{}.PlanModifyString(ctx, req, resp)
		if resp.RequiresReplace != tt.wantReplace {
			t.Errorf("%s: RequiresReplace = %v, want %v", tt.name, resp.RequiresReplace, tt.wantReplace)
		}
	}
}
