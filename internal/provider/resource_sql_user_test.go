// Copyright (c) 2024 muecahit94
// SPDX-License-Identifier: MIT

package provider

import (
	"context"
	"testing"

	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
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
