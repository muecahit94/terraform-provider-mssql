// Copyright (c) 2024 muecahit94
// SPDX-License-Identifier: MIT

package mssql

import "testing"

func TestQuoteName(t *testing.T) {
	tests := []struct{ in, want string }{
		{"app", "[app]"},
		{"my db", "[my db]"},
		{"a]b", "[a]]b]"},
		{"x]; DROP DATABASE y; --", "[x]]; DROP DATABASE y; --]"},
	}
	for _, tt := range tests {
		if got := quoteName(tt.in); got != tt.want {
			t.Errorf("quoteName(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestCollationPattern(t *testing.T) {
	valid := []string{"SQL_Latin1_General_CP1_CI_AS", "Ukrainian_CI_AS", "Latin1_General_100_CI_AS_SC_UTF8"}
	invalid := []string{"", "a b", "a;b", "a'b", "x]--", "Latin1 General", "a-b"}

	for _, c := range valid {
		if !collationPattern.MatchString(c) {
			t.Errorf("collation %q should be accepted", c)
		}
	}
	for _, c := range invalid {
		if collationPattern.MatchString(c) {
			t.Errorf("collation %q should be rejected", c)
		}
	}
}

func TestSetDatabaseRecoveryModelRejectsInvalid(t *testing.T) {
	// The model is validated before any statement is built, so no connection is needed.
	c := &Client{}
	for _, m := range []string{"", "full", "SIMPLE; DROP DATABASE x", "LOGGED"} {
		if err := c.SetDatabaseRecoveryModel(nil, "db", m); err == nil { //nolint:staticcheck // ctx is unused on the validation path
			t.Errorf("SetDatabaseRecoveryModel(%q) should fail", m)
		}
	}
}

func TestCreateDatabaseRejectsInvalidCollation(t *testing.T) {
	c := &Client{}
	if _, err := c.CreateDatabase(nil, CreateDatabaseOptions{Name: "db", Collation: "bad; DROP DATABASE x"}); err == nil { //nolint:staticcheck // ctx is unused on the validation path
		t.Error("CreateDatabase with an injected collation should fail before reaching the server")
	}
}
