// Copyright (c) 2024 muecahit94
// SPDX-License-Identifier: MIT

package mssql

import (
	"testing"
)

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
