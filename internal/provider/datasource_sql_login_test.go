// Copyright (c) 2024 muecahit94
// SPDX-License-Identifier: MIT

package provider

import (
	"context"
	"testing"

	fwdatasource "github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/muecahit94/terraform-provider-mssql/internal/mssql"
)

func TestSQLLoginDataSourceSchema(t *testing.T) {
	ctx := context.Background()
	resp := &fwdatasource.SchemaResponse{}

	NewSQLLoginDataSource().Schema(ctx, fwdatasource.SchemaRequest{}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema() returned errors: %v", resp.Diagnostics.Errors())
	}

	if diags := resp.Schema.ValidateImplementation(ctx); diags.HasError() {
		t.Errorf("ValidateImplementation() returned errors: %v", diags.Errors())
	}
}

func TestSQLLoginDataSourceGetClient(t *testing.T) {
	ctx := context.Background()

	t.Run("unconfigured provider with no server block returns error", func(t *testing.T) {
		d := &SQLLoginDataSource{
			client: mssql.NewMultiServerClient(),
		}
		client, diags := d.getClient(ctx, nil)
		if !diags.HasError() {
			t.Errorf("expected error when getting client without default or server block, got client %v", client)
		}
	})

	t.Run("nil provider client returns error", func(t *testing.T) {
		d := &SQLLoginDataSource{
			client: nil,
		}
		client, diags := d.getClient(ctx, nil)
		if !diags.HasError() {
			t.Errorf("expected error when provider is nil, got client %v", client)
		}
	})
}

func TestValidateDataSourceLoginName(t *testing.T) {
	cases := []struct {
		name        string
		data        SQLLoginDataSourceModel
		expected    string
		expectError bool
	}{
		{
			name: "name only",
			data: SQLLoginDataSourceModel{
				Name:      types.StringValue("my_login"),
				LoginName: types.StringNull(),
			},
			expected:    "my_login",
			expectError: false,
		},
		{
			name: "login_name only",
			data: SQLLoginDataSourceModel{
				Name:      types.StringNull(),
				LoginName: types.StringValue("my_login"),
			},
			expected:    "my_login",
			expectError: false,
		},
		{
			name: "both set with same value",
			data: SQLLoginDataSourceModel{
				Name:      types.StringValue("my_login"),
				LoginName: types.StringValue("my_login"),
			},
			expected:    "my_login",
			expectError: false,
		},
		{
			name: "both set with conflicting values",
			data: SQLLoginDataSourceModel{
				Name:      types.StringValue("login_a"),
				LoginName: types.StringValue("login_b"),
			},
			expected:    "",
			expectError: true,
		},
		{
			name: "neither set",
			data: SQLLoginDataSourceModel{
				Name:      types.StringNull(),
				LoginName: types.StringNull(),
			},
			expected:    "",
			expectError: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, diags := validateDataSourceLoginName(tc.data)
			if tc.expectError && !diags.HasError() {
				t.Errorf("expected error, got none")
			}
			if !tc.expectError && diags.HasError() {
				t.Errorf("unexpected error: %v", diags.Errors())
			}
			if result != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, result)
			}
		})
	}
}

