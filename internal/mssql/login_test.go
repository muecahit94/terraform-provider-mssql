// Copyright (c) 2024 muecahit94
// SPDX-License-Identifier: MIT

package mssql

import (
	"strings"
	"testing"
)

func TestCreateLoginStatement(t *testing.T) {
	base := CreateSQLLoginOptions{Name: "app", Password: "P@ss'w0rd", CheckPolicyEnabled: true}

	got, err := createLoginStatement(base, false)
	want := "CREATE LOGIN [app] WITH PASSWORD = N'P@ss''w0rd', DEFAULT_DATABASE = [master], CHECK_EXPIRATION = OFF, CHECK_POLICY = ON"
	if err != nil || got != want {
		t.Errorf("SQL Server: got %q, %v; want %q", got, err, want)
	}

	withSID := base
	withSID.SID = "0x0123456789ABCDEF0123456789ABCDEF"
	withSID.DefaultLanguage = "Deutsch"
	got, _ = createLoginStatement(withSID, false)
	if !strings.Contains(got, ", SID = 0x0123456789ABCDEF0123456789ABCDEF") || !strings.HasSuffix(got, ", DEFAULT_LANGUAGE = [Deutsch]") {
		t.Errorf("SQL Server with SID and language: %q", got)
	}

	// Azure SQL Database accepts only PASSWORD and SID.
	got, err = createLoginStatement(base, true)
	if err != nil || got != "CREATE LOGIN [app] WITH PASSWORD = N'P@ss''w0rd'" {
		t.Errorf("Azure: got %q, %v", got, err)
	}
	got, err = createLoginStatement(withSID, true)
	if err == nil {
		t.Errorf("Azure must reject a default language other than %s, got %q", azureDefaultLanguage, got)
	}

	azureDefaults := base
	azureDefaults.DefaultDatabase = "MASTER"
	azureDefaults.DefaultLanguage = "us_english"
	if _, err := createLoginStatement(azureDefaults, true); err != nil {
		t.Errorf("Azure must accept its own defaults: %v", err)
	}

	// The policy is always enforced on Azure, so CHECK_POLICY = OFF has no effect and is not an error.
	policyOff := base
	policyOff.CheckPolicyEnabled = false
	if got, err := createLoginStatement(policyOff, true); err != nil || strings.Contains(got, "CHECK_POLICY") {
		t.Errorf("Azure with check_policy_enabled = false: got %q, %v", got, err)
	}
}

func TestAzureLoginOptionsError(t *testing.T) {
	str := func(s string) *string { return &s }
	yes, no := true, false

	tests := []struct {
		name            string
		db, lang        *string
		expiration      *bool
		wantErrContains string
	}{
		{name: "nothing set"},
		{name: "defaults", db: str("master"), lang: str("us_english"), expiration: &no},
		{name: "other database", db: str("app"), wantErrContains: "default_database"},
		{name: "other language", lang: str("Deutsch"), wantErrContains: "default_language"},
		{name: "expiration", expiration: &yes, wantErrContains: "check_expiration_enabled"},
	}
	for _, tt := range tests {
		err := azureLoginOptionsError(tt.db, tt.lang, tt.expiration)
		if tt.wantErrContains == "" && err != nil {
			t.Errorf("%s: unexpected error %v", tt.name, err)
		}
		if tt.wantErrContains != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErrContains)) {
			t.Errorf("%s: error = %v, want it to mention %q", tt.name, err, tt.wantErrContains)
		}
	}
}

func TestFormatSID(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected string
	}{
		{
			name:     "empty byte slice",
			input:    nil,
			expected: "",
		},
		{
			name:     "valid 16-byte binary",
			input:    []byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef},
			expected: "0x0123456789ABCDEF0123456789ABCDEF",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatSID(tt.input)
			if got != tt.expected {
				t.Errorf("FormatSID() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestNormalizeSID(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    string
		expectError bool
	}{
		{
			name:        "valid with 0x prefix uppercase",
			input:       "0x0123456789ABCDEF0123456789ABCDEF",
			expected:    "0x0123456789ABCDEF0123456789ABCDEF",
			expectError: false,
		},
		{
			name:        "valid with 0x prefix lowercase",
			input:       "0x0123456789abcdef0123456789abcdef",
			expected:    "0x0123456789ABCDEF0123456789ABCDEF",
			expectError: false,
		},
		{
			name:        "valid with 0X prefix",
			input:       "0X0123456789ABCDEF0123456789ABCDEF",
			expected:    "0x0123456789ABCDEF0123456789ABCDEF",
			expectError: false,
		},
		{
			name:        "valid without 0x prefix",
			input:       "0123456789abcdef0123456789abcdef",
			expected:    "0x0123456789ABCDEF0123456789ABCDEF",
			expectError: false,
		},
		{
			name:        "valid with hyphens (GUID format)",
			input:       "01234567-89ab-cdef-0123-456789abcdef",
			expected:    "0x0123456789ABCDEF0123456789ABCDEF",
			expectError: false,
		},
		{
			name:        "valid with whitespace",
			input:       "  0x0123456789ABCDEF0123456789ABCDEF  ",
			expected:    "0x0123456789ABCDEF0123456789ABCDEF",
			expectError: false,
		},
		{
			name:        "invalid hex characters",
			input:       "0x0123456789ABCDEF0123456789ZZZZZZ",
			expected:    "",
			expectError: true,
		},
		{
			name:        "too short SID",
			input:       "0x0123456789ABCDEF",
			expected:    "",
			expectError: true,
		},
		{
			name:        "too long SID",
			input:       "0x0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF",
			expected:    "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeSID(tt.input)
			if (err != nil) != tt.expectError {
				t.Fatalf("NormalizeSID(%q) error = %v, expectError %v", tt.input, err, tt.expectError)
			}
			if !tt.expectError && got != tt.expected {
				t.Errorf("NormalizeSID(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}
