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
		client, diags := d.getClient(ctx, SQLLoginDataSourceModel{})
		if !diags.HasError() {
			t.Errorf("expected error when getting client without default or server block, got client %v", client)
		}
	})

	t.Run("nil provider client returns error", func(t *testing.T) {
		d := &SQLLoginDataSource{
			client: nil,
		}
		client, diags := d.getClient(ctx, SQLLoginDataSourceModel{})
		if !diags.HasError() {
			t.Errorf("expected error when provider is nil, got client %v", client)
		}
	})

	t.Run("registered servers map resolves key and legacy block", func(t *testing.T) {
		client := mssql.NewMultiServerClient()
		node1 := &mssql.Config{Hostname: "sql1.example", Port: 1433, SQLAuth: &mssql.SQLAuthConfig{Username: "admin", Password: "current"}}
		node2 := &mssql.Config{Hostname: "sql2.example", Port: 1434, SQLAuth: &mssql.SQLAuthConfig{Username: "admin2", Password: "current2"}}
		client.SetServers(map[string]*mssql.Config{"node1": node1, "node2": node2})
		d := &SQLLoginDataSource{client: client}
		legacy := &ServerModel{
			Hostname: types.StringValue("SQL1.EXAMPLE."),
			SQLAuth:  &SQLAuthModel{Username: types.StringValue("old"), Password: types.StringValue("stale")},
		}

		got, diags := d.resolveServer(SQLLoginDataSourceModel{ServerKey: types.StringValue("node2")})
		if diags.HasError() || got != node2 {
			t.Fatalf("expected node2, got %v, errors %v", got, diags.Errors())
		}

		got, diags = d.resolveServer(SQLLoginDataSourceModel{Server: legacy})
		if diags.HasError() || got != node1 {
			t.Fatalf("expected legacy block to resolve node1 from registry, got %v, errors %v", got, diags.Errors())
		}

		if _, diags := d.resolveServer(SQLLoginDataSourceModel{ServerKey: types.StringValue("missing")}); !diags.HasError() {
			t.Error("missing server key must fail")
		}
		if _, diags := d.resolveServer(SQLLoginDataSourceModel{Server: legacy, ServerKey: types.StringValue("node2")}); !diags.HasError() {
			t.Error("mismatched server_key and server block must fail")
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
