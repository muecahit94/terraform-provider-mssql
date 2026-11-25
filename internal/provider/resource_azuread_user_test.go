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

func TestAzureADUserResourceSchema(t *testing.T) {
	ctx := context.Background()
	resp := &fwresource.SchemaResponse{}

	NewAzureADUserResource().Schema(ctx, fwresource.SchemaRequest{}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema() returned errors: %v", resp.Diagnostics.Errors())
	}
	if diags := resp.Schema.ValidateImplementation(ctx); diags.HasError() {
		t.Errorf("ValidateImplementation() returned errors: %v", diags.Errors())
	}

	// type is read back from the database, so existing users get it without a plan diff.
	principalType := resp.Schema.Attributes["type"]
	if principalType == nil || !principalType.IsOptional() || !principalType.IsComputed() {
		t.Error("type must be optional and computed")
	}
}

func TestAzureADUserValidateConfig(t *testing.T) {
	ctx := context.Background()
	schemaResp := &fwresource.SchemaResponse{}
	r := NewAzureADUserResource()
	r.Schema(ctx, fwresource.SchemaRequest{}, schemaResp)
	objType := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)

	const objectID = "cbb9c7db-2777-47b7-8954-0269ae3dc553"
	tests := []struct {
		name          string
		objectID      interface{}
		principalType interface{}
		wantError     bool
	}{
		{"group by object ID", objectID, "X", false},
		{"user by object ID", objectID, "E", false},
		{"no type", objectID, nil, false},
		{"no type nor object ID", nil, nil, false},
		{"object ID known after apply", tftypes.UnknownValue, "X", false},
		{"type known after apply", objectID, tftypes.UnknownValue, false},
		{"unsupported type", objectID, "S", true},
		{"lower case type", objectID, "x", true},
		{"type without object ID", nil, "X", true},
		{"type with an empty object ID", "", "X", true},
	}
	for _, tt := range tests {
		values := map[string]tftypes.Value{}
		for n, typ := range objType.AttributeTypes {
			values[n] = tftypes.NewValue(typ, nil)
		}
		values["object_id"] = tftypes.NewValue(tftypes.String, tt.objectID)
		values["type"] = tftypes.NewValue(tftypes.String, tt.principalType)

		req := fwresource.ValidateConfigRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: tftypes.NewValue(objType, values)}}
		resp := &fwresource.ValidateConfigResponse{}
		r.(fwresource.ResourceWithValidateConfig).ValidateConfig(ctx, req, resp)
		if resp.Diagnostics.HasError() != tt.wantError {
			t.Errorf("%s: error = %v, wantError %v", tt.name, resp.Diagnostics.Errors(), tt.wantError)
		}
	}
}

func TestPrincipalTypeValue(t *testing.T) {
	if got := principalTypeValue(nil); !got.IsNull() {
		t.Errorf("principalTypeValue(nil) = %v, want null", got)
	}
	if got := principalTypeValue(&mssql.User{}); !got.IsNull() {
		t.Errorf("principalTypeValue(no type) = %v, want null", got)
	}
	if got := principalTypeValue(&mssql.User{Type: "X"}); !got.Equal(types.StringValue("X")) {
		t.Errorf("principalTypeValue(group) = %v, want X", got)
	}
}
