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

func TestAlterUserLoginStatement(t *testing.T) {
	tests := []struct{ user, login, want string }{
		{"app", "app_login", "ALTER USER [app] WITH LOGIN = [app_login]"},
		{`DOMAIN\u`, `DOMAIN\u`, `ALTER USER [DOMAIN\u] WITH LOGIN = [DOMAIN\u]`},
		{"a]b", "c]d", "ALTER USER [a]]b] WITH LOGIN = [c]]d]"},
		{"x", "y]; DROP USER z; --", "ALTER USER [x] WITH LOGIN = [y]]; DROP USER z; --]"},
	}
	for _, tt := range tests {
		if got := alterUserLoginStatement(tt.user, tt.login); got != tt.want {
			t.Errorf("alterUserLoginStatement(%q, %q) = %q, want %q", tt.user, tt.login, got, tt.want)
		}
	}
}

func TestAzureADUserStatement(t *testing.T) {
	const objectID = "cbb9c7db-2777-47b7-8954-0269ae3dc553"
	const sid = "0xDBC7B9CB7727B74789540269AE3DC553"
	tests := []struct {
		name, user, objectID, principalType, want string
	}{
		{"object ID defaults to a user", "app-identity", objectID, "",
			"CREATE USER [app-identity] WITH SID = " + sid + ", TYPE = E, DEFAULT_SCHEMA = [dbo]"},
		{"object ID of a user", "app-identity", objectID, AzureADUserType,
			"CREATE USER [app-identity] WITH SID = " + sid + ", TYPE = E, DEFAULT_SCHEMA = [dbo]"},
		{"object ID of a group", "GRP_APPLICATIONS", objectID, AzureADGroupType,
			"CREATE USER [GRP_APPLICATIONS] WITH SID = " + sid + ", TYPE = X, DEFAULT_SCHEMA = [dbo]"},
		{"no object ID ignores the type", "alice@contoso.com", "", AzureADGroupType,
			"CREATE USER [alice@contoso.com] FROM EXTERNAL PROVIDER WITH DEFAULT_SCHEMA = [dbo]"},
		{"name is quoted", "a]b", objectID, AzureADGroupType,
			"CREATE USER [a]]b] WITH SID = " + sid + ", TYPE = X, DEFAULT_SCHEMA = [dbo]"},
	}
	for _, tt := range tests {
		got, err := azureADUserStatement(tt.user, tt.objectID, tt.principalType, "dbo")
		if err != nil {
			t.Errorf("%s: azureADUserStatement() error = %v", tt.name, err)
			continue
		}
		if got != tt.want {
			t.Errorf("%s: azureADUserStatement() = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestAzureADUserStatementRejectsInvalidInput(t *testing.T) {
	tests := []struct{ name, objectID, principalType string }{
		{"unknown type", "cbb9c7db-2777-47b7-8954-0269ae3dc553", "S"},
		{"injected type", "cbb9c7db-2777-47b7-8954-0269ae3dc553", "X; DROP USER [dbo]"},
		{"invalid object ID", "not-a-guid", AzureADGroupType},
	}
	for _, tt := range tests {
		if got, err := azureADUserStatement("user", tt.objectID, tt.principalType, "dbo"); err == nil {
			t.Errorf("%s: azureADUserStatement() = %q, want an error", tt.name, got)
		}
	}
}
