// Copyright (c) 2024 muecahit94
// SPDX-License-Identifier: MIT

package provider

import (
	"context"
	"testing"

	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/muecahit94/terraform-provider-mssql/internal/mssql"
)

func TestServerConfigurationResourceSchema(t *testing.T) {
	ctx := context.Background()
	resp := &fwresource.SchemaResponse{}
	NewServerConfigurationResource().Schema(ctx, fwresource.SchemaRequest{}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema() returned errors: %v", resp.Diagnostics.Errors())
	}
	if diags := resp.Schema.ValidateImplementation(ctx); diags.HasError() {
		t.Errorf("ValidateImplementation() returned errors: %v", diags.Errors())
	}
	for _, name := range []string{"name", "value"} {
		if !resp.Schema.Attributes[name].IsRequired() {
			t.Errorf("%s must be required", name)
		}
	}
	for _, name := range []string{"previous_value", "value_in_use", "restart_required"} {
		if !resp.Schema.Attributes[name].IsComputed() {
			t.Errorf("%s must be computed", name)
		}
	}
}

func TestApplyServerConfiguration(t *testing.T) {
	var data ServerConfigurationResourceModel
	applyServerConfiguration(&data, &mssql.ServerConfiguration{Name: "max worker threads", Value: 512, ValueInUse: 0, IsDynamic: false})
	if data.ID.ValueString() != "max worker threads" || data.Value.ValueInt64() != 512 || data.ValueInUse.ValueInt64() != 0 {
		t.Errorf("applyServerConfiguration() = %+v", data)
	}
	if !data.RestartRequired.ValueBool() {
		t.Error("a configured value that is not in use, for an option that is not dynamic, needs a restart")
	}

	configured := ServerConfigurationResourceModel{Name: types.StringValue("Max Degree Of Parallelism")}
	applyServerConfiguration(&configured, &mssql.ServerConfiguration{Name: "max degree of parallelism", IsDynamic: true})
	if got := configured.Name.ValueString(); got != "Max Degree Of Parallelism" {
		t.Errorf("name = %q, want the configured spelling (otherwise every plan replaces the resource)", got)
	}
}
