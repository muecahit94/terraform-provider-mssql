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

func TestLinkedServerResourceSchema(t *testing.T) {
	ctx := context.Background()
	resp := &fwresource.SchemaResponse{}

	NewLinkedServerResource().Schema(ctx, fwresource.SchemaRequest{}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema() returned errors: %v", resp.Diagnostics.Errors())
	}
	if diags := resp.Schema.ValidateImplementation(ctx); diags.HasError() {
		t.Errorf("ValidateImplementation() returned errors: %v", diags.Errors())
	}
}

func TestLinkedServerProviderStringIsSensitive(t *testing.T) {
	ctx := context.Background()
	resp := &fwresource.SchemaResponse{}
	NewLinkedServerResource().Schema(ctx, fwresource.SchemaRequest{}, resp)

	if !resp.Schema.Attributes["provider_string"].IsSensitive() {
		t.Error("provider_string must be sensitive: connection strings commonly embed credentials")
	}
}

func TestLinkedServerLoginResourceSchema(t *testing.T) {
	ctx := context.Background()
	resp := &fwresource.SchemaResponse{}

	NewLinkedServerLoginResource().Schema(ctx, fwresource.SchemaRequest{}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema() returned errors: %v", resp.Diagnostics.Errors())
	}
	if diags := resp.Schema.ValidateImplementation(ctx); diags.HasError() {
		t.Errorf("ValidateImplementation() returned errors: %v", diags.Errors())
	}

	// password_wo must never be persisted, which the framework enforces for WriteOnly attributes.
	attr, ok := resp.Schema.Attributes["password_wo"]
	if !ok || !attr.IsWriteOnly() {
		t.Error("password_wo must be a write-only attribute")
	}
}

func TestLinkedServerLoginPasswordSchemaIsSensitive(t *testing.T) {
	ctx := context.Background()
	resp := &fwresource.SchemaResponse{}
	NewLinkedServerLoginResource().Schema(ctx, fwresource.SchemaRequest{}, resp)

	for _, name := range []string{"password", "password_wo"} {
		if !resp.Schema.Attributes[name].IsSensitive() {
			t.Errorf("%s must be sensitive", name)
		}
	}
}

func TestValidateLinkedServerLogin(t *testing.T) {
	tests := []struct {
		name      string
		data      LinkedServerLoginResourceModel
		wantError bool
	}{
		{
			name: "remote user with password",
			data: LinkedServerLoginResourceModel{
				UseSelf:    types.BoolValue(false),
				RemoteUser: types.StringValue("reader"),
				Password:   types.StringValue("P@ssw0rd123!"),
			},
		},
		{
			name: "remote user with write-only password and version",
			data: LinkedServerLoginResourceModel{
				UseSelf:           types.BoolValue(false),
				RemoteUser:        types.StringValue("reader"),
				PasswordWO:        types.StringValue("P@ssw0rd123!"),
				PasswordWOVersion: types.StringValue("1"),
			},
		},
		{
			name: "unknown write-only password counts as set",
			data: LinkedServerLoginResourceModel{
				UseSelf:           types.BoolValue(false),
				RemoteUser:        types.StringValue("reader"),
				PasswordWO:        types.StringUnknown(),
				PasswordWOVersion: types.StringValue("1"),
			},
		},
		{
			name: "use_self alone",
			data: LinkedServerLoginResourceModel{UseSelf: types.BoolValue(true)},
		},
		{
			name: "unknown use_self is not validated",
			data: LinkedServerLoginResourceModel{UseSelf: types.BoolUnknown()},
		},
		{
			name: "use_self with remote user",
			data: LinkedServerLoginResourceModel{
				UseSelf:    types.BoolValue(true),
				RemoteUser: types.StringValue("reader"),
			},
			wantError: true,
		},
		{
			name: "use_self with password",
			data: LinkedServerLoginResourceModel{
				UseSelf:  types.BoolValue(true),
				Password: types.StringValue("P@ssw0rd123!"),
			},
			wantError: true,
		},
		{
			name: "both passwords",
			data: LinkedServerLoginResourceModel{
				UseSelf:    types.BoolValue(false),
				RemoteUser: types.StringValue("reader"),
				Password:   types.StringValue("a"),
				PasswordWO: types.StringValue("b"),
			},
			wantError: true,
		},
		{
			name: "no password",
			data: LinkedServerLoginResourceModel{
				UseSelf:    types.BoolValue(false),
				RemoteUser: types.StringValue("reader"),
			},
			wantError: true,
		},
		{
			name: "no remote user",
			data: LinkedServerLoginResourceModel{
				UseSelf:  types.BoolValue(false),
				Password: types.StringValue("P@ssw0rd123!"),
			},
			wantError: true,
		},
		{
			name: "version without write-only password",
			data: LinkedServerLoginResourceModel{
				UseSelf:           types.BoolValue(false),
				RemoteUser:        types.StringValue("reader"),
				Password:          types.StringValue("P@ssw0rd123!"),
				PasswordWOVersion: types.StringValue("1"),
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diags := validateLinkedServerLogin(tt.data)
			if diags.HasError() != tt.wantError {
				t.Errorf("validateLinkedServerLogin() error = %v, wantError %v", diags.Errors(), tt.wantError)
			}
		})
	}
}

func TestLinkedServerLoginID(t *testing.T) {
	tests := []struct {
		id         string
		wantServer string
		wantLocal  string
		wantOK     bool
	}{
		{"SRV/app_login", "SRV", "app_login", true},
		{"SRV/", "SRV", "", true},
		{"SRV", "", "", false},
		{"/app_login", "", "", false},
		{"", "", "", false},
	}

	for _, tt := range tests {
		server, local, ok := parseLinkedServerLoginID(tt.id)
		if server != tt.wantServer || local != tt.wantLocal || ok != tt.wantOK {
			t.Errorf("parseLinkedServerLoginID(%q) = (%q, %q, %v), want (%q, %q, %v)",
				tt.id, server, local, ok, tt.wantServer, tt.wantLocal, tt.wantOK)
		}
	}

	if got := linkedServerLoginID("SRV", ""); got != "SRV/" {
		t.Errorf("linkedServerLoginID(SRV, \"\") = %q, want %q", got, "SRV/")
	}
}

func TestLinkedServerLoginPassword(t *testing.T) {
	plan := LinkedServerLoginResourceModel{Password: types.StringValue("from-plan")}
	if got := linkedServerLoginPassword(plan, LinkedServerLoginResourceModel{PasswordWO: types.StringNull()}); got != "from-plan" {
		t.Errorf("got %q, want the plan password", got)
	}

	config := LinkedServerLoginResourceModel{PasswordWO: types.StringValue("from-config")}
	if got := linkedServerLoginPassword(LinkedServerLoginResourceModel{Password: types.StringNull()}, config); got != "from-config" {
		t.Errorf("got %q, want the write-only config password", got)
	}
}

func TestLinkedServerValidateConfigSQLServerProduct(t *testing.T) {
	ctx := context.Background()
	schemaResp := &fwresource.SchemaResponse{}
	r := NewLinkedServerResource()
	r.Schema(ctx, fwresource.SchemaRequest{}, schemaResp)

	str := func(v interface{}) tftypes.Value { return tftypes.NewValue(tftypes.String, v) }
	tests := []struct {
		name       string
		product    tftypes.Value
		provider   tftypes.Value
		dataSource tftypes.Value
		wantError  bool
	}{
		{"SQL Server product alone", str("SQL Server"), str(nil), str(nil), false},
		{"SQL Server product with data source", str("SQL Server"), str(nil), str("host"), true},
		{"SQL Server product with provider and data source", str("SQL Server"), str("MSOLEDBSQL"), str("host"), false},
		{"empty product with provider and data source", str(""), str("MSOLEDBSQL"), str("host"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			objType := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
			values := map[string]tftypes.Value{}
			for name, typ := range objType.AttributeTypes {
				values[name] = tftypes.NewValue(typ, nil)
			}
			values["name"] = str("SRV")
			values["product"] = tt.product
			values["provider_name"] = tt.provider
			values["data_source"] = tt.dataSource

			req := fwresource.ValidateConfigRequest{
				Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: tftypes.NewValue(objType, values)},
			}
			resp := &fwresource.ValidateConfigResponse{}
			r.(fwresource.ResourceWithValidateConfig).ValidateConfig(ctx, req, resp)

			if resp.Diagnostics.HasError() != tt.wantError {
				t.Errorf("ValidateConfig() error = %v, wantError %v", resp.Diagnostics.Errors(), tt.wantError)
			}
		})
	}
}

func TestLinkedServerValidateConfigCollation(t *testing.T) {
	ctx := context.Background()
	schemaResp := &fwresource.SchemaResponse{}
	r := NewLinkedServerResource()
	r.Schema(ctx, fwresource.SchemaRequest{}, schemaResp)

	tests := []struct {
		name      string
		collation tftypes.Value
		useRemote tftypes.Value
		wantError bool
	}{
		{"no collation", tftypes.NewValue(tftypes.String, nil), tftypes.NewValue(tftypes.Bool, nil), false},
		{"collation with local collation", tftypes.NewValue(tftypes.String, "Latin1_General_CI_AS"), tftypes.NewValue(tftypes.Bool, false), false},
		{"collation with remote collation", tftypes.NewValue(tftypes.String, "Latin1_General_CI_AS"), tftypes.NewValue(tftypes.Bool, true), true},
		{"collation with default use_remote_collation", tftypes.NewValue(tftypes.String, "Latin1_General_CI_AS"), tftypes.NewValue(tftypes.Bool, nil), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			objType := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
			values := map[string]tftypes.Value{}
			for name, typ := range objType.AttributeTypes {
				values[name] = tftypes.NewValue(typ, nil)
			}
			values["name"] = tftypes.NewValue(tftypes.String, "SRV")
			values["collation_name"] = tt.collation
			values["use_remote_collation"] = tt.useRemote

			req := fwresource.ValidateConfigRequest{
				Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: tftypes.NewValue(objType, values)},
			}
			resp := &fwresource.ValidateConfigResponse{}
			r.(fwresource.ResourceWithValidateConfig).ValidateConfig(ctx, req, resp)

			if resp.Diagnostics.HasError() != tt.wantError {
				t.Errorf("ValidateConfig() error = %v, wantError %v", resp.Diagnostics.Errors(), tt.wantError)
			}
		})
	}
}
