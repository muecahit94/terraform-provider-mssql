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

func TestAlterAuthorizationStatement(t *testing.T) {
	tests := []struct{ database, login, want string }{
		{"app", "app_owner", "ALTER AUTHORIZATION ON DATABASE::[app] TO [app_owner]"},
		{"my db", `DOMAIN\user`, `ALTER AUTHORIZATION ON DATABASE::[my db] TO [DOMAIN\user]`},
		{"a]b", "c]d", "ALTER AUTHORIZATION ON DATABASE::[a]]b] TO [c]]d]"},
		{"x", "y]; DROP LOGIN z; --", "ALTER AUTHORIZATION ON DATABASE::[x] TO [y]]; DROP LOGIN z; --]"},
	}
	for _, tt := range tests {
		if got := alterAuthorizationStatement(tt.database, tt.login); got != tt.want {
			t.Errorf("alterAuthorizationStatement(%q, %q) = %q, want %q", tt.database, tt.login, got, tt.want)
		}
	}
}

func TestDatabaseSettingsStatements(t *testing.T) {
	on, off, checksum := true, false, "CHECKSUM"

	got, err := DatabaseSettings{
		AutoClose:             &off,
		AutoShrink:            &off,
		PageVerify:            &checksum,
		SnapshotIsolation:     &on,
		ReadCommittedSnapshot: &on,
		QueryStore:            &on,
		Trustworthy:           &off,
	}.Statements("app")
	if err != nil {
		t.Fatalf("Statements() error = %v", err)
	}
	want := []string{
		"ALTER DATABASE [app] SET AUTO_CLOSE OFF",
		"ALTER DATABASE [app] SET AUTO_SHRINK OFF",
		"ALTER DATABASE [app] SET PAGE_VERIFY CHECKSUM",
		"ALTER DATABASE [app] SET ALLOW_SNAPSHOT_ISOLATION ON",
		"ALTER DATABASE [app] SET READ_COMMITTED_SNAPSHOT ON WITH NO_WAIT",
		"ALTER DATABASE [app] SET QUERY_STORE = ON",
		"ALTER DATABASE [app] SET TRUSTWORTHY OFF",
	}
	if len(got) != len(want) {
		t.Fatalf("Statements() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("statement %d = %q, want %q", i, got[i], want[i])
		}
	}

	// A nil field is left alone.
	if got, _ := (DatabaseSettings{}).Statements("app"); len(got) != 0 {
		t.Errorf("no settings must give no statements, got %v", got)
	}

	// The database name is quoted.
	got, _ = DatabaseSettings{AutoClose: &on}.Statements("a]b")
	if len(got) != 1 || got[0] != "ALTER DATABASE [a]]b] SET AUTO_CLOSE ON" {
		t.Errorf("the name must be quoted, got %v", got)
	}
}

func TestDatabaseSettingsRejectInvalidPageVerify(t *testing.T) {
	for _, v := range []string{"", "checksum", "CHECKSUM; DROP DATABASE x", "BOTH"} {
		v := v
		if _, err := (DatabaseSettings{PageVerify: &v}).Statements("app"); err == nil {
			t.Errorf("page verify %q must be rejected before a statement is built", v)
		}
	}
	for _, v := range PageVerifyOptions {
		v := v
		if _, err := (DatabaseSettings{PageVerify: &v}).Statements("app"); err != nil {
			t.Errorf("page verify %q must be accepted: %v", v, err)
		}
	}
}
