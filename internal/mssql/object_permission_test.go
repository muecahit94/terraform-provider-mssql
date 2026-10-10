// Copyright (c) 2024 muecahit94
// SPDX-License-Identifier: MIT

package mssql

import "testing"

func TestNormalizeObjectPermission(t *testing.T) {
	valid := map[string]string{
		"SELECT":           "SELECT",
		"select":           "SELECT",
		" view definition": "VIEW DEFINITION",
		"TAKE OWNERSHIP":   "TAKE OWNERSHIP",
		"EXECUTE":          "EXECUTE",
	}
	for in, want := range valid {
		got, err := NormalizeObjectPermission(in)
		if err != nil || got != want {
			t.Errorf("NormalizeObjectPermission(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, in := range []string{"", "SELECT; DROP TABLE x", "SELECT,INSERT", "SELECT (a)", "SELECT--", "1SELECT", "SELECT  INSERT"} {
		if got, err := NormalizeObjectPermission(in); err == nil {
			t.Errorf("NormalizeObjectPermission(%q) = %q, want an error", in, got)
		}
	}
}

func TestGrantObjectPermissionStatement(t *testing.T) {
	tests := []struct {
		name                                    string
		schema, object, column, principal, perm string
		withGrant                               bool
		want                                    string
	}{
		{"table", "dbo", "Orders", "", "app_reader", "SELECT", false, "GRANT SELECT ON OBJECT::[dbo].[Orders] TO [app_reader]"},
		{"column", "dbo", "Orders", "Total", "app_reader", "update", false, "GRANT UPDATE ON OBJECT::[dbo].[Orders] ([Total]) TO [app_reader]"},
		{"grant option", "sales", "usp_Close", "", "app_role", "EXECUTE", true, "GRANT EXECUTE ON OBJECT::[sales].[usp_Close] TO [app_role] WITH GRANT OPTION"},
		{"two-word permission", "dbo", "V", "", "u", "view definition", false, "GRANT VIEW DEFINITION ON OBJECT::[dbo].[V] TO [u]"},
		{"windows principal", "dbo", "T", "", `DOMAIN\user`, "SELECT", false, `GRANT SELECT ON OBJECT::[dbo].[T] TO [DOMAIN\user]`},
		{"closing bracket is escaped", "a]b", "c]d", "e]f", "g]h", "SELECT", false, "GRANT SELECT ON OBJECT::[a]]b].[c]]d] ([e]]f]) TO [g]]h]"},
	}
	for _, tt := range tests {
		got, err := GrantObjectPermissionStatement(tt.schema, tt.object, tt.column, tt.principal, tt.perm, tt.withGrant)
		if err != nil || got != tt.want {
			t.Errorf("%s: got %q, %v; want %q", tt.name, got, err, tt.want)
		}
	}
	if _, err := GrantObjectPermissionStatement("dbo", "T", "", "u", "SELECT; DROP TABLE T", false); err == nil {
		t.Error("an injected permission must be rejected before a statement is built")
	}
}

func TestRevokeObjectPermissionStatement(t *testing.T) {
	got, err := RevokeObjectPermissionStatement("dbo", "Orders", "Total", "app_reader", "select")
	want := "REVOKE SELECT ON OBJECT::[dbo].[Orders] ([Total]) FROM [app_reader] CASCADE"
	if err != nil || got != want {
		t.Errorf("got %q, %v; want %q", got, err, want)
	}
	if _, err := RevokeObjectPermissionStatement("dbo", "T", "", "u", "SELECT)"); err == nil {
		t.Error("an invalid permission must be rejected")
	}
}
