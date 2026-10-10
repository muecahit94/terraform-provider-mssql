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
	"github.com/muecahit94/terraform-provider-mssql/internal/mssql"
)

func TestWindowsLoginResourceSchema(t *testing.T) {
	ctx := context.Background()
	resp := &fwresource.SchemaResponse{}
	NewWindowsLoginResource().Schema(ctx, fwresource.SchemaRequest{}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema() returned errors: %v", resp.Diagnostics.Errors())
	}
	if diags := resp.Schema.ValidateImplementation(ctx); diags.HasError() {
		t.Errorf("ValidateImplementation() returned errors: %v", diags.Errors())
	}
	if !resp.Schema.Attributes["name"].IsRequired() {
		t.Error("name must be required")
	}
	// There is no password: a Windows login authenticates through Windows.
	if _, ok := resp.Schema.Attributes["password"]; ok {
		t.Error("a Windows login has no password attribute")
	}
	for _, name := range []string{"default_database", "default_language", "is_disabled"} {
		attr := resp.Schema.Attributes[name]
		if !attr.IsOptional() || !attr.IsComputed() {
			t.Errorf("%s must be Optional and Computed so that an imported login shows no diff", name)
		}
	}
}

func TestApplyWindowsLogin(t *testing.T) {
	var data WindowsLoginResourceModel
	applyWindowsLogin(&data, &mssql.WindowsLogin{
		PrincipalID: 270, Name: `CORP\ops`, Type: "WINDOWS_GROUP", DefaultDatabase: "master", DefaultLanguage: "", IsDisabled: true,
	})
	if data.ID.ValueString() != "270" || data.Name.ValueString() != `CORP\ops` || data.Type.ValueString() != "WINDOWS_GROUP" ||
		data.DefaultDatabase.ValueString() != "master" || !data.IsDisabled.ValueBool() {
		t.Errorf("applyWindowsLogin() = %+v", data)
	}
	// An empty language is a known value, so that a configuration without it shows no diff.
	if data.DefaultLanguage.IsNull() || data.DefaultLanguage.IsUnknown() {
		t.Error("an empty default language must be a known value")
	}
}

func TestApplyWindowsLoginKeepsConfiguredCase(t *testing.T) {
	data := WindowsLoginResourceModel{Name: types.StringValue(`corp\Alice`)}
	applyWindowsLogin(&data, &mssql.WindowsLogin{PrincipalID: 1, Name: `CORP\alice`, Type: "WINDOWS_LOGIN", DefaultDatabase: "master"})
	if got := data.Name.ValueString(); got != `corp\Alice` {
		t.Errorf("name = %q, want the configured spelling (otherwise every plan replaces the login)", got)
	}

	data = WindowsLoginResourceModel{Name: types.StringValue(`CORP\old`)}
	applyWindowsLogin(&data, &mssql.WindowsLogin{PrincipalID: 1, Name: `CORP\new`})
	if got := data.Name.ValueString(); got != `CORP\new` {
		t.Errorf("name = %q, a different name must come from the server", got)
	}
}

func TestWindowsLoginValidateConfig(t *testing.T) {
	ctx := context.Background()
	schemaResp := &fwresource.SchemaResponse{}
	r := NewWindowsLoginResource()
	r.Schema(ctx, fwresource.SchemaRequest{}, schemaResp)
	objType := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)

	tests := []struct {
		name      string
		value     interface{}
		wantError bool
	}{
		{"domain user", `CORP\alice`, false},
		{"local machine group", `BUILTIN\Administrators`, false},
		{"name with spaces", `CORP\db admins`, false},
		{"UPN", "alice@corp.example", true},
		{"no domain", "alice", true},
		{"two backslashes", `CORP\sub\alice`, true},
		{"empty user part", `CORP\`, true},
		{"unknown", tftypes.UnknownValue, false},
	}
	for _, tt := range tests {
		values := map[string]tftypes.Value{}
		for n, typ := range objType.AttributeTypes {
			values[n] = tftypes.NewValue(typ, nil)
		}
		values["name"] = tftypes.NewValue(tftypes.String, tt.value)
		req := fwresource.ValidateConfigRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: tftypes.NewValue(objType, values)}}
		resp := &fwresource.ValidateConfigResponse{}
		r.(fwresource.ResourceWithValidateConfig).ValidateConfig(ctx, req, resp)
		if resp.Diagnostics.HasError() != tt.wantError {
			t.Errorf("%s: error = %v, wantError %v", tt.name, resp.Diagnostics.Errors(), tt.wantError)
		}
	}
}
