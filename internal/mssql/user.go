// Copyright (c) 2024 muecahit94
// SPDX-License-Identifier: MIT

package mssql

import (
	"context"
	"database/sql"
	"fmt"
)

// User represents a database user.
type User struct {
	PrincipalID       int
	Name              string
	DatabaseID        int
	DefaultSchemaName string
	Type              string // S = SQL user, U = Windows user, E = External user (Azure AD)
	LoginName         string
}

// GetUser retrieves a user from a specific database.
func (c *Client) GetUser(ctx context.Context, databaseName, userName string) (*User, error) {
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
		WHERE dp.name = @p1 AND dp.type IN ('S', 'U', 'E')`

	row, err := c.QueryRowInDatabaseContext(ctx, databaseName, query, userName)
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
		WHERE dp.principal_id = @p1 AND dp.type IN ('S', 'U', 'E')`

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
		WHERE dp.type IN ('S', 'U', 'E')
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
		err := c.ExecInDatabaseContext(ctx, opts.DatabaseName, query)
		if err != nil {
			return nil, fmt.Errorf("failed to update SQL user: %w", err)
		}
	}

	return c.GetUser(ctx, opts.DatabaseName, opts.UserName)
}

// DropUser drops a user from a database.
func (c *Client) DropUser(ctx context.Context, databaseName, userName string) error {
	query := fmt.Sprintf("DROP USER IF EXISTS [%s]", userName)
	err := c.ExecInDatabaseContext(ctx, databaseName, query)
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
	if err := c.UseDatabase(ctx, opts.DatabaseName); err != nil {
		return nil, err
	}

	defaultSchema := opts.DefaultSchema
	if defaultSchema == "" {
		defaultSchema = "dbo"
	}

	query := fmt.Sprintf(
		"CREATE USER [%s] WITH SID = %s, TYPE = E, DEFAULT_SCHEMA = [%s]",
		opts.UserName,
		opts.ObjectID,
		defaultSchema,
	)

	_, err := c.ExecContext(ctx, query)
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
	if err := c.UseDatabase(ctx, opts.DatabaseName); err != nil {
		return nil, err
	}

	defaultSchema := opts.DefaultSchema
	if defaultSchema == "" {
		defaultSchema = "dbo"
	}

	query := fmt.Sprintf(
		"CREATE USER [%s] WITH SID = %s, TYPE = E, DEFAULT_SCHEMA = [%s]",
		opts.Name,
		opts.ClientID,
		defaultSchema,
	)

	_, err := c.ExecContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to create Azure AD service principal: %w", err)
	}

	return c.GetUser(ctx, opts.DatabaseName, opts.Name)
}
