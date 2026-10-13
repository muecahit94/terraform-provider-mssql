// Copyright (c) 2024 muecahit94
// SPDX-License-Identifier: MIT

package mssql

import (
	"context"
	"regexp"
	"strings"
	"testing"
)

// Azure SQL Database rejects SUSER_SNAME/SUSER_SID with an argument (issue #49) and three-part names
// that point into another database.
var azureIncompatible = regexp.MustCompile(`(?i)SUSER_S(NAME|ID)\s*\(\s*[^)\s]|\]\.sys\.`)

func TestQueriesRunOnAzureSQLDatabase(t *testing.T) {
	queries := map[string]string{
		"databaseSelectAzure":   databaseSelectAzure,
		"objectPermissionQuery": objectPermissionQuery,
		"userSelect":            userSelect,
	}
	for name, query := range queries {
		if m := azureIncompatible.FindString(query); m != "" {
			t.Errorf("%s uses %q, which Azure SQL Database rejects", name, m)
		}
	}
	// Azure SQL Database lists SQL logins only in sys.sql_logins.
	if !strings.Contains(databaseSelectAzure, "sys.sql_logins") {
		t.Error("databaseSelectAzure must resolve the owner through sys.sql_logins")
	}
}

// The owner on SQL Server is read as since 1.8.0, so that upgrades show no diff.
func TestDatabaseSelectSQLServerKeepsSUSERSNAME(t *testing.T) {
	if !strings.Contains(databaseSelectSQLServer, "SUSER_SNAME(d.owner_sid)") {
		t.Errorf("databaseSelectSQLServer = %q", databaseSelectSQLServer)
	}
	if strings.Contains(databaseSelectSQLServer, "%") || strings.Contains(databaseSelectAzure, "%") {
		t.Error("the owner expression was not substituted")
	}
}

func TestAzureIncompatiblePattern(t *testing.T) {
	for _, bad := range []string{"SUSER_SNAME(owner_sid)", "suser_sname( d.owner_sid )", "SUSER_SID('x')", "FROM [db].sys.objects"} {
		if !azureIncompatible.MatchString(bad) {
			t.Errorf("pattern must catch %q", bad)
		}
	}
	for _, good := range []string{"SUSER_SNAME()", "FROM sys.databases d", "LEFT JOIN sys.server_principals sp"} {
		if azureIncompatible.MatchString(good) {
			t.Errorf("pattern must not flag %q", good)
		}
	}
}

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

func TestQuoteString(t *testing.T) {
	tests := []struct{ in, want string }{
		{"P@ssw0rd123!", "N'P@ssw0rd123!'"},
		{"", "N''"},
		{"it's", "N'it''s'"},
		{"x', CHECK_POLICY = OFF; DROP LOGIN sa; --", "N'x'', CHECK_POLICY = OFF; DROP LOGIN sa; --'"},
		{"ünïcødé", "N'ünïcødé'"},
	}
	for _, tt := range tests {
		if got := quoteString(tt.in); got != tt.want {
			t.Errorf("quoteString(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestPermissionStatementsRejectInjectedPermissions(t *testing.T) {
	c := &Client{}
	ctx := context.Background()
	const injected = "SELECT TO public; DROP TABLE t; --"
	checks := map[string]error{
		"GrantDatabasePermission":  c.GrantDatabasePermission(ctx, "db", "u", injected, false),
		"RevokeDatabasePermission": c.RevokeDatabasePermission(ctx, "db", "u", injected),
		"GrantSchemaPermission":    c.GrantSchemaPermission(ctx, "db", "s", "u", injected, false),
		"RevokeSchemaPermission":   c.RevokeSchemaPermission(ctx, "db", "s", "u", injected),
		"GrantServerPermission":    c.GrantServerPermission(ctx, "u", injected, false),
		"RevokeServerPermission":   c.RevokeServerPermission(ctx, "u", injected),
	}
	for name, err := range checks {
		if err == nil {
			t.Errorf("%s must reject %q before reaching the server", name, injected)
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
