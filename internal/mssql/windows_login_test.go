// Copyright (c) 2024 muecahit94
// SPDX-License-Identifier: MIT

package mssql

import "testing"

func TestCreateWindowsLoginStatement(t *testing.T) {
	tests := []struct{ name, db, lang, want string }{
		{`CORP\alice`, "", "", `CREATE LOGIN [CORP\alice] FROM WINDOWS`},
		{`CORP\alice`, "app", "", `CREATE LOGIN [CORP\alice] FROM WINDOWS WITH DEFAULT_DATABASE = [app]`},
		{`CORP\alice`, "", "us_english", `CREATE LOGIN [CORP\alice] FROM WINDOWS WITH DEFAULT_LANGUAGE = [us_english]`},
		{`CORP\ops`, "app", "us_english", `CREATE LOGIN [CORP\ops] FROM WINDOWS WITH DEFAULT_DATABASE = [app], DEFAULT_LANGUAGE = [us_english]`},
		{"alice@corp.example", "", "", "CREATE LOGIN [alice@corp.example] FROM WINDOWS"},
		{"a]b", "c]d", "", "CREATE LOGIN [a]]b] FROM WINDOWS WITH DEFAULT_DATABASE = [c]]d]"},
		{"x]; DROP LOGIN y; --", "", "", "CREATE LOGIN [x]]; DROP LOGIN y; --] FROM WINDOWS"},
	}
	for _, tt := range tests {
		if got := createWindowsLoginStatement(tt.name, tt.db, tt.lang); got != tt.want {
			t.Errorf("createWindowsLoginStatement(%q, %q, %q) = %q, want %q", tt.name, tt.db, tt.lang, got, tt.want)
		}
	}
}

func TestAlterLoginStatement(t *testing.T) {
	if got := alterLoginStatement(`CORP\alice`, "DISABLE"); got != `ALTER LOGIN [CORP\alice] DISABLE` {
		t.Errorf("got %q", got)
	}
	if got := alterLoginStatement("a]b", "WITH DEFAULT_DATABASE = [x]"); got != "ALTER LOGIN [a]]b] WITH DEFAULT_DATABASE = [x]" {
		t.Errorf("got %q", got)
	}
}
