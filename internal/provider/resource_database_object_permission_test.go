// Copyright (c) 2024 muecahit94
// SPDX-License-Identifier: MIT

package provider

import (
	"context"
	"testing"

	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestDatabaseObjectPermissionResourceSchema(t *testing.T) {
	ctx := context.Background()
	resp := &fwresource.SchemaResponse{}
	NewDatabaseObjectPermissionResource().Schema(ctx, fwresource.SchemaRequest{}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema() returned errors: %v", resp.Diagnostics.Errors())
	}
	if diags := resp.Schema.ValidateImplementation(ctx); diags.HasError() {
		t.Errorf("ValidateImplementation() returned errors: %v", diags.Errors())
	}
	for _, name := range []string{"database_name", "schema_name", "object_name", "principal_name", "permission"} {
		if !resp.Schema.Attributes[name].IsRequired() {
			t.Errorf("%s must be required", name)
		}
	}
	if !resp.Schema.Attributes["column_name"].IsOptional() {
		t.Error("column_name must be optional: a permission on the whole object has none")
	}
}

func TestObjectPermissionID(t *testing.T) {
	if got := objectPermissionID("db", "dbo", "T", "", "u", "select"); got != "db/dbo/T/u/SELECT" {
		t.Errorf("object id = %q", got)
	}
	if got := objectPermissionID("db", "dbo", "T", "c", "u", "update"); got != "db/dbo/T/c/u/UPDATE" {
		t.Errorf("column id = %q", got)
	}

	tests := []struct {
		id                                                string
		db, schema, object, column, principal, permission string
		wantErr                                           bool
	}{
		{"db/dbo/T/u/SELECT", "db", "dbo", "T", "", "u", "SELECT", false},
		{"db/dbo/T/c/u/UPDATE", "db", "dbo", "T", "c", "u", "UPDATE", false},
		{"db/dbo/T/u/VIEW DEFINITION", "db", "dbo", "T", "", "u", "VIEW DEFINITION", false},
		{"db/dbo/T/u", "", "", "", "", "", "", true},
		{"db/dbo/T/c/u/UPDATE/extra", "", "", "", "", "", "", true},
		{"", "", "", "", "", "", "", true},
	}
	for _, tt := range tests {
		db, schema, object, column, principal, permission, err := parseObjectPermissionID(tt.id)
		if (err != nil) != tt.wantErr {
			t.Errorf("parseObjectPermissionID(%q) error = %v, wantErr %v", tt.id, err, tt.wantErr)
			continue
		}
		if db != tt.db || schema != tt.schema || object != tt.object || column != tt.column || principal != tt.principal || permission != tt.permission {
			t.Errorf("parseObjectPermissionID(%q) = %q %q %q %q %q %q", tt.id, db, schema, object, column, principal, permission)
		}
	}
}

func TestDatabaseObjectPermissionValidateConfig(t *testing.T) {
	ctx := context.Background()
	schemaResp := &fwresource.SchemaResponse{}
	r := NewDatabaseObjectPermissionResource()
	r.Schema(ctx, fwresource.SchemaRequest{}, schemaResp)
	objType := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)

	tests := []struct {
		name       string
		permission interface{}
		wantError  bool
	}{
		{"upper case", "SELECT", false},
		{"two words", "VIEW DEFINITION", false},
		{"lower case", "select", true},
		{"injection", "SELECT; DROP TABLE T", true},
		{"unknown value is not checked yet", tftypes.UnknownValue, false},
	}
	for _, tt := range tests {
		values := map[string]tftypes.Value{}
		for name, typ := range objType.AttributeTypes {
			values[name] = tftypes.NewValue(typ, nil)
		}
		values["permission"] = tftypes.NewValue(tftypes.String, tt.permission)
		req := fwresource.ValidateConfigRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: tftypes.NewValue(objType, values)}}
		resp := &fwresource.ValidateConfigResponse{}
		r.(fwresource.ResourceWithValidateConfig).ValidateConfig(ctx, req, resp)
		if resp.Diagnostics.HasError() != tt.wantError {
			t.Errorf("%s: error = %v, wantError %v", tt.name, resp.Diagnostics.Errors(), tt.wantError)
		}
	}
}
