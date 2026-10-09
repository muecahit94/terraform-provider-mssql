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
			name: "no password set (an existing login)",
			data: SQLLoginResourceModel{
				Password:          types.StringNull(),
				PasswordWO:        types.StringNull(),
				PasswordWOVersion: types.StringNull(),
			},
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

func TestGetLoginName(t *testing.T) {
	tests := []struct {
		name     string
		model    SQLLoginResourceModel
		expected string
	}{
		{
			name:     "from name",
			model:    SQLLoginResourceModel{Name: types.StringValue("user1"), LoginName: types.StringNull()},
			expected: "user1",
		},
		{
			name:     "from login_name",
			model:    SQLLoginResourceModel{Name: types.StringNull(), LoginName: types.StringValue("user2")},
			expected: "user2",
		},
		{
			name:     "both set returns name",
			model:    SQLLoginResourceModel{Name: types.StringValue("user1"), LoginName: types.StringValue("user1")},
			expected: "user1",
		},
		{
			name:     "neither set",
			model:    SQLLoginResourceModel{Name: types.StringNull(), LoginName: types.StringNull()},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getLoginName(tt.model)
			if got != tt.expected {
				t.Errorf("getLoginName() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestValidateLoginName(t *testing.T) {
	tests := []struct {
		name      string
		data      SQLLoginResourceModel
		wantError bool
	}{
		{
			name: "name only",
			data: SQLLoginResourceModel{
				Name:      types.StringValue("valid_user"),
				LoginName: types.StringNull(),
			},
			wantError: false,
		},
		{
			name: "login_name only",
			data: SQLLoginResourceModel{
				Name:      types.StringNull(),
				LoginName: types.StringValue("valid_user"),
			},
			wantError: false,
		},
		{
			name: "both set with same value",
			data: SQLLoginResourceModel{
				Name:      types.StringValue("valid_user"),
				LoginName: types.StringValue("valid_user"),
			},
			wantError: false,
		},
		{
			name: "both set with conflicting values",
			data: SQLLoginResourceModel{
				Name:      types.StringValue("user1"),
				LoginName: types.StringValue("user2"),
			},
			wantError: true,
		},
		{
			name: "neither set",
			data: SQLLoginResourceModel{
				Name:      types.StringNull(),
				LoginName: types.StringNull(),
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diags := validateLoginName(tt.data)
			if diags.HasError() != tt.wantError {
				t.Errorf("validateLoginName() error = %v, want %v: %v", diags.HasError(), tt.wantError, diags.Errors())
			}
		})
	}
}

func TestValidateServerConfig(t *testing.T) {
	tests := []struct {
		name      string
		server    *ServerModel
		wantError bool
	}{
		{
			name:      "nil server (provider default)",
			server:    nil,
			wantError: false,
		},
		{
			name: "valid with hostname and sql_auth",
			server: &ServerModel{
				Hostname: types.StringValue("sql1.corp"),
				Port:     types.Int64Value(1433),
				SQLAuth: &SQLAuthModel{
					Username: types.StringValue("sa"),
					Password: types.StringValue("pass123"),
				},
			},
			wantError: false,
		},
		{
			name: "valid with host and login alias",
			server: &ServerModel{
				Host: types.StringValue("sql2.corp"),
				Login: &SQLAuthModel{
					Username: types.StringValue("sa"),
					Password: types.StringValue("pass123"),
				},
			},
			wantError: false,
		},
		{
			name: "valid with hostname and azure_auth",
			server: &ServerModel{
				Hostname: types.StringValue("sqlserver.database.windows.net"),
				AzureAuth: &AzureAuthModel{
					ClientID:     types.StringValue("client-id"),
					ClientSecret: types.StringValue("client-secret"),
					TenantID:     types.StringValue("tenant-id"),
				},
			},
			wantError: false,
		},
		{
			name: "conflicting hostname and host",
			server: &ServerModel{
				Hostname: types.StringValue("sql1.corp"),
				Host:     types.StringValue("sql2.corp"),
				SQLAuth: &SQLAuthModel{
					Username: types.StringValue("sa"),
					Password: types.StringValue("pass123"),
				},
			},
			wantError: true,
		},
		{
			name: "missing hostname and host",
			server: &ServerModel{
				Hostname: types.StringNull(),
				Host:     types.StringNull(),
				SQLAuth: &SQLAuthModel{
					Username: types.StringValue("sa"),
					Password: types.StringValue("pass123"),
				},
			},
			wantError: true,
		},
		{
			name: "conflicting sql_auth and login",
			server: &ServerModel{
				Hostname: types.StringValue("sql1.corp"),
				SQLAuth: &SQLAuthModel{
					Username: types.StringValue("sa"),
					Password: types.StringValue("pass123"),
				},
				Login: &SQLAuthModel{
					Username: types.StringValue("sa"),
					Password: types.StringValue("pass123"),
				},
			},
			wantError: true,
		},
		{
			name: "conflicting sql_auth and azure_auth",
			server: &ServerModel{
				Hostname: types.StringValue("sql1.corp"),
				SQLAuth: &SQLAuthModel{
					Username: types.StringValue("sa"),
					Password: types.StringValue("pass123"),
				},
				AzureAuth: &AzureAuthModel{
					ClientID:     types.StringValue("client-id"),
					ClientSecret: types.StringValue("client-secret"),
				},
			},
			wantError: true,
		},
		{
			name: "missing authentication",
			server: &ServerModel{
				Hostname: types.StringValue("sql1.corp"),
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diags := validateServerConfig(tt.server)
			if diags.HasError() != tt.wantError {
				t.Errorf("validateServerConfig() error = %v, want %v: %v", diags.HasError(), tt.wantError, diags.Errors())
			}
		})
	}
}

func TestServerToConfig(t *testing.T) {
	t.Run("converts hostname and sql_auth", func(t *testing.T) {
		server := &ServerModel{
			Hostname: types.StringValue("sql1.corp"),
			Port:     types.Int64Value(14333),
			SQLAuth: &SQLAuthModel{
				Username: types.StringValue("sa"),
				Password: types.StringValue("password123"),
			},
		}

		cfg, diags := serverToConfig(server)
		if diags.HasError() {
			t.Fatalf("unexpected error: %v", diags.Errors())
		}
		if cfg.Hostname != "sql1.corp" {
			t.Errorf("expected Hostname 'sql1.corp', got %q", cfg.Hostname)
		}
		if cfg.Port != 14333 {
			t.Errorf("expected Port 14333, got %d", cfg.Port)
		}
		if cfg.SQLAuth == nil || cfg.SQLAuth.Username != "sa" || cfg.SQLAuth.Password != "password123" {
			t.Errorf("unexpected SQLAuth: %+v", cfg.SQLAuth)
		}
	})

	t.Run("converts host and login alias with default port", func(t *testing.T) {
		server := &ServerModel{
			Host: types.StringValue("sql2.corp"),
			Port: types.Int64Null(),
			Login: &SQLAuthModel{
				Username: types.StringValue("admin"),
				Password: types.StringValue("adminpass"),
			},
		}

		cfg, diags := serverToConfig(server)
		if diags.HasError() {
			t.Fatalf("unexpected error: %v", diags.Errors())
		}
		if cfg.Hostname != "sql2.corp" {
			t.Errorf("expected Hostname 'sql2.corp', got %q", cfg.Hostname)
		}
		if cfg.Port != 1433 {
			t.Errorf("expected default Port 1433, got %d", cfg.Port)
		}
		if cfg.SQLAuth == nil || cfg.SQLAuth.Username != "admin" || cfg.SQLAuth.Password != "adminpass" {
			t.Errorf("unexpected SQLAuth: %+v", cfg.SQLAuth)
		}
	})

	t.Run("converts azure_auth", func(t *testing.T) {
		server := &ServerModel{
			Hostname: types.StringValue("azure.database.windows.net"),
			AzureAuth: &AzureAuthModel{
				ClientID:     types.StringValue("client-123"),
				ClientSecret: types.StringValue("secret-456"),
				TenantID:     types.StringValue("tenant-789"),
			},
		}

		cfg, diags := serverToConfig(server)
		if diags.HasError() {
			t.Fatalf("unexpected error: %v", diags.Errors())
		}
		if cfg.AzureAuth == nil || cfg.AzureAuth.ClientID != "client-123" || cfg.AzureAuth.TenantID != "tenant-789" {
			t.Errorf("unexpected AzureAuth: %+v", cfg.AzureAuth)
		}
	})
}

func TestGetClient(t *testing.T) {
	ctx := context.Background()

	t.Run("unconfigured provider with no server block returns error", func(t *testing.T) {
		r := &SQLLoginResource{
			client: mssql.NewMultiServerClient(),
		}
		client, diags := r.getClient(ctx, nil)
		if !diags.HasError() {
			t.Errorf("expected error when getting client without default or server block, got client %v", client)
		}
	})

	t.Run("nil provider client returns error", func(t *testing.T) {
		r := &SQLLoginResource{
			client: nil,
		}
		client, diags := r.getClient(ctx, nil)
		if !diags.HasError() {
			t.Errorf("expected error when provider is nil, got client %v", client)
		}
	})
}

func TestValidateLoginPasswordOnCreate(t *testing.T) {
	tests := []struct {
		name      string
		config    SQLLoginResourceModel
		wantError bool
	}{
		{"no password", SQLLoginResourceModel{Password: types.StringNull(), PasswordWO: types.StringNull()}, true},
		{"password", SQLLoginResourceModel{Password: types.StringValue("P@ssw0rd123!"), PasswordWO: types.StringNull()}, false},
		{"write-only password", SQLLoginResourceModel{Password: types.StringNull(), PasswordWO: types.StringValue("P@ssw0rd123!")}, false},
		// an ephemeral value is unknown until apply: it counts as set
		{"unknown write-only password", SQLLoginResourceModel{Password: types.StringNull(), PasswordWO: types.StringUnknown()}, false},
		{"unknown password", SQLLoginResourceModel{Password: types.StringUnknown(), PasswordWO: types.StringNull()}, false},
	}
	for _, tt := range tests {
		diags := validateLoginPasswordOnCreate(tt.config)
		if diags.HasError() != tt.wantError {
			t.Errorf("%s: error = %v, wantError %v", tt.name, diags.Errors(), tt.wantError)
		}
	}
}
