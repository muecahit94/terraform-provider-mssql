// Copyright (c) 2024 muecahit94
// SPDX-License-Identifier: MIT

package mssql

import (
	"strings"
	"testing"
)

func TestUserPrincipalTypes(t *testing.T) {
	// G (Windows group) must be read like the other user types, otherwise a user created
	// for a group login cannot be found again.
	for _, typ := range []string{"'S'", "'U'", "'G'", "'E'", "'X'"} {
		if !strings.Contains(userPrincipalTypes, typ) {
			t.Errorf("userPrincipalTypes %q must contain %s", userPrincipalTypes, typ)
		}
	}
	// Roles and certificate users are not users in this sense.
	for _, typ := range []string{"'R'", "'C'", "'K'", "'A'"} {
		if strings.Contains(userPrincipalTypes, typ) {
			t.Errorf("userPrincipalTypes %q must not contain %s", userPrincipalTypes, typ)
		}
	}
}

func TestCreateUserStatement(t *testing.T) {
	tests := []struct {
		name                      string
		user, login, schema, want string
	}{
		{"for login", "app", "app_login", "dbo", "CREATE USER [app] FOR LOGIN [app_login] WITH DEFAULT_SCHEMA = [dbo]"},
		{"without login", "app", "", "dbo", "CREATE USER [app] WITHOUT LOGIN WITH DEFAULT_SCHEMA = [dbo]"},
		{"windows group", `DOMAIN\group`, `DOMAIN\group`, "dbo", `CREATE USER [DOMAIN\group] FOR LOGIN [DOMAIN\group] WITH DEFAULT_SCHEMA = [dbo]`},
		{"closing bracket is escaped", "a]b", "c]d", "s]t", "CREATE USER [a]]b] FOR LOGIN [c]]d] WITH DEFAULT_SCHEMA = [s]]t]"},
		{"injection attempt stays inside the identifier", "x]; DROP USER y; --", "", "dbo", "CREATE USER [x]]; DROP USER y; --] WITHOUT LOGIN WITH DEFAULT_SCHEMA = [dbo]"},
	}
	for _, tt := range tests {
		if got := createUserStatement(tt.user, tt.login, tt.schema); got != tt.want {
			t.Errorf("%s: createUserStatement() = %q, want %q", tt.name, got, tt.want)
		}
	}
}
