// Copyright (c) 2024 muecahit94
// SPDX-License-Identifier: MIT

package mssql

import (
	"testing"
)

func TestCacheKey(t *testing.T) {
	tests := []struct {
		name     string
		cfg      *Config
		expected string
	}{
		{
			name: "sql auth with port",
			cfg: &Config{
				Hostname: "sql1.corp",
				Port:     1433,
				SQLAuth: &SQLAuthConfig{
					Username: "sa",
					Password: "password123",
				},
			},
			expected: "sql1.corp:1433:sql:sa:password123",
		},
		{
			name: "sql auth with custom port",
			cfg: &Config{
				Hostname: "sql2.corp",
				Port:     14333,
				SQLAuth: &SQLAuthConfig{
					Username: "admin",
					Password: "secure_pass",
				},
			},
			expected: "sql2.corp:14333:sql:admin:secure_pass",
		},
		{
			name: "azure auth",
			cfg: &Config{
				Hostname: "sqlserver.database.windows.net",
				Port:     1433,
				AzureAuth: &AzureAuthConfig{
					ClientID:     "app-id-123",
					ClientSecret: "secret-456",
					TenantID:     "tenant-789",
				},
			},
			expected: "sqlserver.database.windows.net:1433:az:tenant-789:app-id-123:secret-456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CacheKey(tt.cfg)
			if got != tt.expected {
				t.Errorf("CacheKey() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestNewMultiServerClient(t *testing.T) {
	client := NewMultiServerClient()
	if client == nil {
		t.Fatal("NewMultiServerClient returned nil")
	}
	if client.DB() != nil {
		t.Errorf("expected DB() to be nil for uninitialized multi-server client, got %v", client.DB())
	}
	if client.pool == nil {
		t.Error("expected client.pool to be non-nil")
	}
}

func TestClientPoolGetSet(t *testing.T) {
	pool := NewClientPool()
	key := "test-server:1433"

	if _, ok := pool.Get(key); ok {
		t.Error("expected key to not exist in empty pool")
	}

	mockClient := &Client{hostname: "test-server", port: 1433}
	pool.Set(key, mockClient)

	got, ok := pool.Get(key)
	if !ok {
		t.Fatal("expected key to exist in pool after Set")
	}
	if got.Hostname() != "test-server" || got.Port() != 1433 {
		t.Errorf("retrieved client doesn't match: got %s:%d, want test-server:1433", got.Hostname(), got.Port())
	}
}
