// Copyright (c) 2024 muecahit94
// SPDX-License-Identifier: MIT

package provider

import (
	"context"
	"testing"

	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestSQLLoginResourceSchema(t *testing.T) {
	ctx := context.Background()
	resp := &fwresource.SchemaResponse{}

	NewSQLLoginResource().Schema(ctx, fwresource.SchemaRequest{}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema() returned errors: %v", resp.Diagnostics.Errors())
	}

	// Catches illegal attribute combinations, such as a write-only attribute
	// that is also Computed.
	if diags := resp.Schema.ValidateImplementation(ctx); diags.HasError() {
		t.Errorf("ValidateImplementation() returned errors: %v", diags.Errors())
	}
}

func TestValidateLoginPassword(t *testing.T) {
	tests := []struct {
		name      string
		data      SQLLoginResourceModel
		wantError bool
	}{
		{
			name: "password only",
			data: SQLLoginResourceModel{
				Password:          types.StringValue("P@ssw0rd123!"),
				PasswordWO:        types.StringNull(),
				PasswordWOVersion: types.StringNull(),
			},
		},
		{
			name: "password_wo only",
			data: SQLLoginResourceModel{
				Password:          types.StringNull(),
				PasswordWO:        types.StringValue("P@ssw0rd123!"),
				PasswordWOVersion: types.StringNull(),
			},
		},
		{
			name: "password_wo with version",
			data: SQLLoginResourceModel{
				Password:          types.StringNull(),
				PasswordWO:        types.StringValue("P@ssw0rd123!"),
				PasswordWOVersion: types.StringValue("1"),
			},
		},
		{
			name: "unknown password_wo counts as set",
			data: SQLLoginResourceModel{
				Password:          types.StringNull(),
				PasswordWO:        types.StringUnknown(),
				PasswordWOVersion: types.StringValue("1"),
			},
		},
		{
			name: "both passwords set",
			data: SQLLoginResourceModel{
				Password:          types.StringValue("P@ssw0rd123!"),
				PasswordWO:        types.StringValue("P@ssw0rd123!"),
				PasswordWOVersion: types.StringNull(),
			},
			wantError: true,
		},
		{
			name: "no password set",
			data: SQLLoginResourceModel{
				Password:          types.StringNull(),
				PasswordWO:        types.StringNull(),
				PasswordWOVersion: types.StringNull(),
			},
			wantError: true,
		},
		{
			name: "version without password_wo",
			data: SQLLoginResourceModel{
				Password:          types.StringValue("P@ssw0rd123!"),
				PasswordWO:        types.StringNull(),
				PasswordWOVersion: types.StringValue("1"),
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diags := validateLoginPassword(tt.data)
			if diags.HasError() != tt.wantError {
				t.Errorf("validateLoginPassword() error = %v, want %v: %v", diags.HasError(), tt.wantError, diags.Errors())
			}
		})
	}
}

func TestLoginCreatePassword(t *testing.T) {
	tests := []struct {
		name     string
		plan     SQLLoginResourceModel
		config   SQLLoginResourceModel
		expected string
	}{
		{
			name:     "password from plan",
			plan:     SQLLoginResourceModel{Password: types.StringValue("from-plan")},
			config:   SQLLoginResourceModel{Password: types.StringValue("from-plan"), PasswordWO: types.StringNull()},
			expected: "from-plan",
		},
		{
			// Terraform strips write-only values from the plan, so the config is
			// the only place the value can be read from.
			name:     "password_wo from config",
			plan:     SQLLoginResourceModel{Password: types.StringNull(), PasswordWO: types.StringNull()},
			config:   SQLLoginResourceModel{Password: types.StringNull(), PasswordWO: types.StringValue("from-config")},
			expected: "from-config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := loginCreatePassword(tt.plan, tt.config)
			if got != tt.expected {
				t.Errorf("loginCreatePassword() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestLoginUpdatePassword(t *testing.T) {
	tests := []struct {
		name     string
		plan     SQLLoginResourceModel
		state    SQLLoginResourceModel
		config   SQLLoginResourceModel
		expected *string
	}{
		{
			name:     "password unchanged",
			plan:     SQLLoginResourceModel{Password: types.StringValue("same")},
			state:    SQLLoginResourceModel{Password: types.StringValue("same")},
			config:   SQLLoginResourceModel{PasswordWO: types.StringNull()},
			expected: nil,
		},
		{
			name:     "password changed",
			plan:     SQLLoginResourceModel{Password: types.StringValue("new")},
			state:    SQLLoginResourceModel{Password: types.StringValue("old")},
			config:   SQLLoginResourceModel{PasswordWO: types.StringNull()},
			expected: ptr("new"),
		},
		{
			name:     "empty planned password is ignored",
			plan:     SQLLoginResourceModel{Password: types.StringValue("")},
			state:    SQLLoginResourceModel{Password: types.StringValue("old")},
			config:   SQLLoginResourceModel{PasswordWO: types.StringNull()},
			expected: nil,
		},
		{
			name: "password_wo without a version change",
			plan: SQLLoginResourceModel{
				Password:          types.StringNull(),
				PasswordWOVersion: types.StringValue("1"),
			},
			state: SQLLoginResourceModel{
				Password:          types.StringNull(),
				PasswordWOVersion: types.StringValue("1"),
			},
			config:   SQLLoginResourceModel{PasswordWO: types.StringValue("rotated")},
			expected: nil,
		},
		{
			name: "password_wo with a bumped version",
			plan: SQLLoginResourceModel{
				Password:          types.StringNull(),
				PasswordWOVersion: types.StringValue("2"),
			},
			state: SQLLoginResourceModel{
				Password:          types.StringNull(),
				PasswordWOVersion: types.StringValue("1"),
			},
			config:   SQLLoginResourceModel{PasswordWO: types.StringValue("rotated")},
			expected: ptr("rotated"),
		},
		{
			// Migrating from `password` to `password_wo`: the old value is still
			// in state, so write the write-only one to converge.
			name: "migration from password to password_wo",
			plan: SQLLoginResourceModel{
				Password:          types.StringNull(),
				PasswordWOVersion: types.StringNull(),
			},
			state: SQLLoginResourceModel{
				Password:          types.StringValue("old"),
				PasswordWOVersion: types.StringNull(),
			},
			config:   SQLLoginResourceModel{PasswordWO: types.StringValue("write-only")},
			expected: ptr("write-only"),
		},
		{
			name: "password_wo without any version at all",
			plan: SQLLoginResourceModel{
				Password:          types.StringNull(),
				PasswordWOVersion: types.StringNull(),
			},
			state: SQLLoginResourceModel{
				Password:          types.StringNull(),
				PasswordWOVersion: types.StringNull(),
			},
			config:   SQLLoginResourceModel{PasswordWO: types.StringValue("rotated")},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := loginUpdatePassword(tt.plan, tt.state, tt.config)
			switch {
			case got == nil && tt.expected == nil:
			case got == nil || tt.expected == nil:
				t.Errorf("loginUpdatePassword() = %v, want %v", got, tt.expected)
			case *got != *tt.expected:
				t.Errorf("loginUpdatePassword() = %q, want %q", *got, *tt.expected)
			}
		})
	}
}

func ptr(s string) *string {
	return &s
}
