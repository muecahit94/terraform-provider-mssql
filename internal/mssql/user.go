// Copyright (c) 2024 muecahit94
// SPDX-License-Identifier: MIT

package mssql

import (
	"context"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
)

// userPrincipalTypes are the sys.database_principals types managed as users:
// S = SQL user, U = Windows user, G = Windows group, E = Entra ID user,
// X = Entra ID group.
const userPrincipalTypes = "'S', 'U', 'G', 'E', 'X'"

// guidToSID converts an Azure AD Object ID (GUID) to the binary SID format required by SQL Server.
// For example: "cbb9c7db-2777-47b7-8954-0269ae3dc553" -> "0xDBC7B9CB7727B74789540269AE3DC553"
func guidToSID(guid string) (string, error) {
	// Remove hyphens from GUID
	cleanGUID := strings.ReplaceAll(guid, "-", "")
	if len(cleanGUID) != 32 {
		return "", fmt.Errorf("invalid GUID format: %s", guid)
	}

	// Parse the GUID parts (GUIDs have a specific byte order)
	// Format: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
	// Parts:  time_low-time_mid-time_hi_and_version-clock_seq_and_reserved-node

	// Decode the hex string
	bytes, err := hex.DecodeString(cleanGUID)
	if err != nil {
		return "", fmt.Errorf("failed to decode GUID: %w", err)
	}

	// GUIDs in SQL Server SID format: the first three parts are little-endian
	// Swap bytes in parts 1, 2, and 3 (bytes 0-3, 4-5, 6-7)
	// Part 1: bytes 0-3 (4 bytes) - reverse
	bytes[0], bytes[1], bytes[2], bytes[3] = bytes[3], bytes[2], bytes[1], bytes[0]
	// Part 2: bytes 4-5 (2 bytes) - reverse
	bytes[4], bytes[5] = bytes[5], bytes[4]
	// Part 3: bytes 6-7 (2 bytes) - reverse
	bytes[6], bytes[7] = bytes[7], bytes[6]
	// Parts 4 and 5 (bytes 8-15) remain in big-endian order

	// Convert to hex string with 0x prefix
	return "0x" + strings.ToUpper(hex.EncodeToString(bytes)), nil
}

// User represents a database user.
type User struct {
	PrincipalID       int
	Name              string
	DatabaseID        int
	DefaultSchemaName string
	Type              string // S = SQL user, U = Windows user, E = External user (Azure AD)
	LoginName         string
	// AuthenticationType is INSTANCE (a login), WINDOWS, NONE (created WITHOUT LOGIN) or EXTERNAL.
	AuthenticationType string

	sid []byte
}

const userSelect = `
	SELECT
		dp.principal_id,
		dp.name,
		DB_ID() AS database_id,
		ISNULL(dp.default_schema_name, 'dbo'),
		dp.type,
		ISNULL(sp.name, ''),
		dp.authentication_type_desc,
		dp.sid
	FROM sys.database_principals dp
	LEFT JOIN sys.server_principals sp ON dp.sid = sp.sid
	WHERE dp.type IN (` + userPrincipalTypes + `)`

func scanUser(s interface{ Scan(dest ...any) error }) (*User, error) {
	var user User
	if err := s.Scan(&user.PrincipalID, &user.Name, &user.DatabaseID, &user.DefaultSchemaName, &user.Type,
		&user.LoginName, &user.AuthenticationType, &user.sid); err != nil {
		return nil, err
	}
	return &user, nil
}

// fillLoginName looks up the login of a user in the server's own context. In Azure SQL Database a user
// database does not see the logins, which live in master, so the join in userSelect finds none.
func (c *Client) fillLoginName(ctx context.Context, user *User) error {
	if user == nil || user.LoginName != "" || len(user.sid) == 0 || c.db == nil {
		return nil
	}
	err := c.QueryRowContext(ctx, "SELECT name FROM sys.sql_logins WHERE sid = @p1", user.sid).Scan(&user.LoginName)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to look up the login of user %q: %w", user.Name, err)
	}
	return nil
}

// GetUser retrieves a user from a specific database, or nil when it does not exist.
func (c *Client) GetUser(ctx context.Context, databaseName, userName string) (*User, error) {
	var user *User
	err := c.withDatabase(ctx, databaseName, func(db sqlRunner) error {
		var err error
		user, err = scanUser(db.QueryRowContext(ctx, userSelect+` AND dp.name = @p1`, userName))
		return err
	})
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if err := c.fillLoginName(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

// ListUsers retrieves all users from a specific database.
func (c *Client) ListUsers(ctx context.Context, databaseName string) ([]User, error) {
	var users []User
	err := c.withDatabase(ctx, databaseName, func(db sqlRunner) error {
		rows, err := db.QueryContext(ctx, userSelect+` ORDER BY dp.name`)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			user, err := scanUser(rows)
			if err != nil {
				return err
			}
			users = append(users, *user)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	for i := range users {
		if err := c.fillLoginName(ctx, &users[i]); err != nil {
			return nil, err
		}
	}
	return users, nil
}

// AuthenticationTypeNone is the authentication type of a user created WITHOUT LOGIN.
const AuthenticationTypeNone = "NONE"

// alterUserLoginStatement builds the statement that maps a user to another login.
func alterUserLoginStatement(userName, loginName string) string {
	return "ALTER USER " + quoteName(userName) + " WITH LOGIN = " + quoteName(loginName)
}

// RelinkUser maps a user to another login with ALTER USER ... WITH LOGIN: the user, its permissions and its
// role memberships stay. It also fixes an orphaned user, whose SID matches no login. SQL Server cannot
// do it for a user that was created WITHOUT LOGIN.
func (c *Client) RelinkUser(ctx context.Context, databaseName, userName, loginName string) error {
	err := c.withDatabase(ctx, databaseName, func(db sqlRunner) error {
		_, err := db.ExecContext(ctx, alterUserLoginStatement(userName, loginName))
		return err
	})
	if err != nil {
		return fmt.Errorf("failed to map the user to the login: %w", err)
	}
	return nil
}

// createUserStatement builds CREATE USER. An empty login creates a user without a login
// (it cannot be used to connect, but can own permissions and be impersonated).
func createUserStatement(userName, loginName, defaultSchema string) string {
	query := "CREATE USER " + quoteName(userName)
	if loginName == "" {
		query += " WITHOUT LOGIN"
	} else {
		query += " FOR LOGIN " + quoteName(loginName)
	}
	return query + " WITH DEFAULT_SCHEMA = " + quoteName(defaultSchema)
}

// CreateSQLUserOptions contains options for creating a SQL user.
type CreateSQLUserOptions struct {
	DatabaseName  string
	UserName      string
	LoginName     string
	DefaultSchema string
}

// CreateSQLUser creates a new SQL user mapped to a login.
func (c *Client) CreateSQLUser(ctx context.Context, opts CreateSQLUserOptions) (*User, error) {
	defaultSchema := opts.DefaultSchema
	if defaultSchema == "" {
		defaultSchema = "dbo"
	}

	query := createUserStatement(opts.UserName, opts.LoginName, defaultSchema)

	err := c.ExecInDatabaseContext(ctx, opts.DatabaseName, query)
	if err != nil {
		return nil, fmt.Errorf("failed to create SQL user: %w", err)
	}

	user, err := c.GetUser(ctx, opts.DatabaseName, opts.UserName)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, fmt.Errorf("user was created but could not be retrieved")
	}
	return user, nil
}

// UpdateSQLUserOptions contains options for updating a SQL user.
type UpdateSQLUserOptions struct {
	DatabaseName  string
	UserName      string
	DefaultSchema *string
}

// UpdateSQLUser updates an existing SQL user.
func (c *Client) UpdateSQLUser(ctx context.Context, opts UpdateSQLUserOptions) (*User, error) {
	if opts.DefaultSchema != nil {
		query := "ALTER USER " + quoteName(opts.UserName) + " WITH DEFAULT_SCHEMA = " + quoteName(*opts.DefaultSchema)

		// Try to get a direct connection to the database first (Azure SQL support)
		db, err := c.GetDatabaseConnection(ctx, opts.DatabaseName)
		if err == nil {
			defer db.Close()
			_, err = db.ExecContext(ctx, query)
			if err != nil {
				return nil, fmt.Errorf("failed to update SQL user: %w", err)
			}
		} else {
			// Fallback to existing logic
			err = c.ExecInDatabaseContext(ctx, opts.DatabaseName, query)
			if err != nil {
				return nil, fmt.Errorf("failed to update SQL user: %w", err)
			}
		}
	}

	return c.GetUser(ctx, opts.DatabaseName, opts.UserName)
}

// DropUser drops a user from a database.
func (c *Client) DropUser(ctx context.Context, databaseName, userName string) error {
	query := "DROP USER IF EXISTS " + quoteName(userName)

	// Try to get a direct connection to the database first (Azure SQL support)
	db, err := c.GetDatabaseConnection(ctx, databaseName)
	if err == nil {
		defer db.Close()
		_, err = db.ExecContext(ctx, query)
		return err
	}

	// Fallback to existing logic
	err = c.ExecInDatabaseContext(ctx, databaseName, query)
	if err != nil {
		return fmt.Errorf("failed to drop user: %w", err)
	}

	return nil
}

// CreateAzureADUserOptions contains options for creating an Azure AD user.
type CreateAzureADUserOptions struct {
	DatabaseName  string
	UserName      string
	ObjectID      string
	DefaultSchema string
}

// CreateAzureADUser creates a new Azure AD user.
func (c *Client) CreateAzureADUser(ctx context.Context, opts CreateAzureADUserOptions) (*User, error) {
	// Get a connection directly to the target database
	// This is required for Azure SQL Database which doesn't support USE statement
	db, err := c.GetDatabaseConnection(ctx, opts.DatabaseName)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database %s: %w", opts.DatabaseName, err)
	}
	defer db.Close()

	defaultSchema := opts.DefaultSchema
	if defaultSchema == "" {
		defaultSchema = "dbo"
	}

	var query string
	if opts.ObjectID != "" {
		// For managed identities: use SID-based creation
		// Convert Azure AD Object ID (GUID) to binary SID format
		sid, err := guidToSID(opts.ObjectID)
		if err != nil {
			return nil, fmt.Errorf("failed to convert object ID to SID: %w", err)
		}

		query = fmt.Sprintf(
			"CREATE USER %s WITH SID = %s, TYPE = E, DEFAULT_SCHEMA = %s",
			quoteName(opts.UserName),
			sid,
			quoteName(defaultSchema),
		)
	} else {
		// For email-based users: use FROM EXTERNAL PROVIDER
		query = fmt.Sprintf(
			"CREATE USER %s FROM EXTERNAL PROVIDER WITH DEFAULT_SCHEMA = %s",
			quoteName(opts.UserName),
			quoteName(defaultSchema),
		)
	}

	_, err = db.ExecContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to create Azure AD user: %w", err)
	}

	return c.GetUser(ctx, opts.DatabaseName, opts.UserName)
}

// CreateAzureADServicePrincipalOptions contains options for creating an Azure AD service principal.
type CreateAzureADServicePrincipalOptions struct {
	DatabaseName  string
	Name          string
	ClientID      string
	DefaultSchema string
}

// CreateAzureADServicePrincipal creates a new Azure AD service principal.
func (c *Client) CreateAzureADServicePrincipal(ctx context.Context, opts CreateAzureADServicePrincipalOptions) (*User, error) {
	// Get a connection directly to the target database
	// This is required for Azure SQL Database which doesn't support USE statement
	db, err := c.GetDatabaseConnection(ctx, opts.DatabaseName)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database %s: %w", opts.DatabaseName, err)
	}
	defer db.Close()

	defaultSchema := opts.DefaultSchema
	if defaultSchema == "" {
		defaultSchema = "dbo"
	}

	// Convert Azure AD Client ID (GUID) to binary SID format
	sid, err := guidToSID(opts.ClientID)
	if err != nil {
		return nil, fmt.Errorf("failed to convert client ID to SID: %w", err)
	}

	query := fmt.Sprintf(
		"CREATE USER %s WITH SID = %s, TYPE = E, DEFAULT_SCHEMA = %s",
		quoteName(opts.Name),
		sid,
		quoteName(defaultSchema),
	)

	_, err = db.ExecContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to create Azure AD service principal: %w", err)
	}

	return c.GetUser(ctx, opts.DatabaseName, opts.Name)
}
