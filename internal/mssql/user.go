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
}

// Request a user from a specific database.
func (c *Client) GetUser(ctx context.Context, databaseName, userName string) (*User, error) {
	// Try to get a direct connection to the database first (Azure SQL support)
	db, err := c.GetDatabaseConnection(ctx, databaseName)
	if err == nil {
		defer db.Close()
		return c.getUserWithDB(ctx, db, userName)
	}

	// Fallback to USE statement for on-premises SQL Server
	query := `
		SELECT
			dp.principal_id,
			dp.name,
			DB_ID() as database_id,
			ISNULL(dp.default_schema_name, 'dbo'),
			dp.type,
			ISNULL(sp.name, '')
		FROM sys.database_principals dp
		LEFT JOIN sys.server_principals sp ON dp.sid = sp.sid
		WHERE dp.name = @p1 AND dp.type IN ('S', 'U', 'E', 'X')` // X = EXTERNAL_GROUP

	row, err := c.QueryRowInDatabaseContext(ctx, databaseName, query, userName)
	if err != nil {
		return nil, err
	}

	return scanUser(row)
}

func (c *Client) getUserWithDB(ctx context.Context, db *sql.DB, userName string) (*User, error) {
	query := `
		SELECT
			dp.principal_id,
			dp.name,
			DB_ID() as database_id,
			ISNULL(dp.default_schema_name, 'dbo'),
			dp.type,
			ISNULL(sp.name, '')
		FROM sys.database_principals dp
		LEFT JOIN sys.server_principals sp ON dp.sid = sp.sid
		WHERE dp.name = @p1 AND dp.type IN ('S', 'U', 'E', 'X')` // X = EXTERNAL_GROUP

	row := db.QueryRowContext(ctx, query, userName)
	return scanUser(row)
}

func scanUser(row *sql.Row) (*User, error) {
	var user User
	err := row.Scan(
		&user.PrincipalID,
		&user.Name,
		&user.DatabaseID,
		&user.DefaultSchemaName,
		&user.Type,
		&user.LoginName,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &user, nil
}

// GetUserByID retrieves a user by principal ID from a specific database.
func (c *Client) GetUserByID(ctx context.Context, databaseName string, principalID int) (*User, error) {
	query := `
		SELECT
			dp.principal_id,
			dp.name,
			DB_ID() as database_id,
			ISNULL(dp.default_schema_name, 'dbo'),
			dp.type,
			ISNULL(sp.name, '')
		FROM sys.database_principals dp
		LEFT JOIN sys.server_principals sp ON dp.sid = sp.sid
		WHERE dp.principal_id = @p1 AND dp.type IN ('S', 'U', 'E', 'X')` // X = EXTERNAL_GROUP

	row, err := c.QueryRowInDatabaseContext(ctx, databaseName, query, principalID)
	if err != nil {
		return nil, err
	}

	var user User
	err = row.Scan(
		&user.PrincipalID,
		&user.Name,
		&user.DatabaseID,
		&user.DefaultSchemaName,
		&user.Type,
		&user.LoginName,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

// ListUsers retrieves all users from a specific database.
func (c *Client) ListUsers(ctx context.Context, databaseName string) ([]User, error) {
	// Get a dedicated connection from the pool
	conn, err := c.db.Conn(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get database connection: %w", err)
	}
	defer conn.Close()

	// Switch to the target database
	if _, err := conn.ExecContext(ctx, fmt.Sprintf("USE [%s]", databaseName)); err != nil {
		return nil, fmt.Errorf("failed to switch database context: %w", err)
	}

	query := `
		SELECT
			dp.principal_id,
			dp.name,
			DB_ID() as database_id,
			ISNULL(dp.default_schema_name, 'dbo'),
			dp.type,
			ISNULL(sp.name, '')
		FROM sys.database_principals dp
		LEFT JOIN sys.server_principals sp ON dp.sid = sp.sid
		WHERE dp.type IN ('S', 'U', 'E', 'X') // X = EXTERNAL_GROUP
		ORDER BY dp.name`

	rows, err := conn.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		if err := rows.Scan(
			&user.PrincipalID,
			&user.Name,
			&user.DatabaseID,
			&user.DefaultSchemaName,
			&user.Type,
			&user.LoginName,
		); err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}

	return users, rows.Err()
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

	query := fmt.Sprintf(
		"CREATE USER [%s] FOR LOGIN [%s] WITH DEFAULT_SCHEMA = [%s]",
		opts.UserName,
		opts.LoginName,
		defaultSchema,
	)

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
		query := fmt.Sprintf("ALTER USER [%s] WITH DEFAULT_SCHEMA = [%s]", opts.UserName, *opts.DefaultSchema)

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
	query := fmt.Sprintf("DROP USER IF EXISTS [%s]", userName)

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
			"CREATE USER [%s] WITH SID = %s, TYPE = E, DEFAULT_SCHEMA = [%s]",
			opts.UserName,
			sid,
			defaultSchema,
		)
	} else {
		// For email-based users: use FROM EXTERNAL PROVIDER
		query = fmt.Sprintf(
			"CREATE USER [%s] FROM EXTERNAL PROVIDER WITH DEFAULT_SCHEMA = [%s]",
			opts.UserName,
			defaultSchema,
		)
	}

	_, err = db.ExecContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to create Azure AD user: %w", err)
	}

	return c.getUserWithDB(ctx, db, opts.UserName)
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
		"CREATE USER [%s] WITH SID = %s, TYPE = E, DEFAULT_SCHEMA = [%s]",
		opts.Name,
		sid,
		defaultSchema,
	)

	_, err = db.ExecContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to create Azure AD service principal: %w", err)
	}

	return c.getUserWithDB(ctx, db, opts.Name)
}
