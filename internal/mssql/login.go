// Copyright (c) 2024 muecahit94
// SPDX-License-Identifier: MIT

package mssql

import (
	"context"
	"database/sql"
	"fmt"
)

// SQLLogin represents a SQL Server login.
type SQLLogin struct {
	PrincipalID            int
	Name                   string
	DefaultDatabaseName    string
	DefaultLanguageName    string
	CheckExpirationEnabled bool
	CheckPolicyEnabled     bool
	IsDisabled             bool
}

// GetSQLLogin retrieves a SQL login by name.
func (c *Client) GetSQLLogin(ctx context.Context, name string) (*SQLLogin, error) {
	query := `
		SELECT
			principal_id,
			name,
			ISNULL(default_database_name, 'master'),
			ISNULL(default_language_name, ''),
			ISNULL(is_expiration_checked, 0),
			ISNULL(is_policy_checked, 0),
			is_disabled
		FROM sys.sql_logins
		WHERE name = @p1`
	row := c.QueryRowContext(ctx, query, name)

	var login SQLLogin
	err := row.Scan(
		&login.PrincipalID,
		&login.Name,
		&login.DefaultDatabaseName,
		&login.DefaultLanguageName,
		&login.CheckExpirationEnabled,
		&login.CheckPolicyEnabled,
		&login.IsDisabled,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get SQL login: %w", err)
	}

	return &login, nil
}

// GetSQLLoginByID retrieves a SQL login by principal ID.
func (c *Client) GetSQLLoginByID(ctx context.Context, id int) (*SQLLogin, error) {
	query := `
		SELECT
			principal_id,
			name,
			ISNULL(default_database_name, 'master'),
			ISNULL(default_language_name, ''),
			ISNULL(is_expiration_checked, 0),
			ISNULL(is_policy_checked, 0),
			is_disabled
		FROM sys.sql_logins
		WHERE principal_id = @p1`
	row := c.QueryRowContext(ctx, query, id)

	var login SQLLogin
	err := row.Scan(
		&login.PrincipalID,
		&login.Name,
		&login.DefaultDatabaseName,
		&login.DefaultLanguageName,
		&login.CheckExpirationEnabled,
		&login.CheckPolicyEnabled,
		&login.IsDisabled,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get SQL login: %w", err)
	}

	return &login, nil
}

// ListSQLLogins retrieves all SQL logins.
func (c *Client) ListSQLLogins(ctx context.Context) ([]SQLLogin, error) {
	query := `
		SELECT
			principal_id,
			name,
			ISNULL(default_database_name, 'master'),
			ISNULL(default_language_name, ''),
			ISNULL(is_expiration_checked, 0),
			ISNULL(is_policy_checked, 0),
			is_disabled
		FROM sys.sql_logins
		ORDER BY name`
	rows, err := c.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list SQL logins: %w", err)
	}
	defer rows.Close()

	var logins []SQLLogin
	for rows.Next() {
		var login SQLLogin
		if err := rows.Scan(
			&login.PrincipalID,
			&login.Name,
			&login.DefaultDatabaseName,
			&login.DefaultLanguageName,
			&login.CheckExpirationEnabled,
			&login.CheckPolicyEnabled,
			&login.IsDisabled,
		); err != nil {
			return nil, fmt.Errorf("failed to scan SQL login: %w", err)
		}
		logins = append(logins, login)
	}

	return logins, rows.Err()
}

// CreateSQLLoginOptions contains options for creating a SQL login.
type CreateSQLLoginOptions struct {
	Name                   string
	Password               string
	DefaultDatabase        string
	DefaultLanguage        string
	CheckExpirationEnabled bool
	CheckPolicyEnabled     bool
}

// CreateSQLLogin creates a new SQL login.
func (c *Client) CreateSQLLogin(ctx context.Context, opts CreateSQLLoginOptions) (*SQLLogin, error) {
	defaultDB := opts.DefaultDatabase
	if defaultDB == "" {
		defaultDB = "master"
	}

	query := fmt.Sprintf(`
		CREATE LOGIN [%s] WITH PASSWORD = '%s',
		DEFAULT_DATABASE = [%s],
		CHECK_EXPIRATION = %s,
		CHECK_POLICY = %s`,
		opts.Name,
		opts.Password,
		defaultDB,
		boolToOnOff(opts.CheckExpirationEnabled),
		boolToOnOff(opts.CheckPolicyEnabled),
	)

	if opts.DefaultLanguage != "" {
		query += fmt.Sprintf(", DEFAULT_LANGUAGE = [%s]", opts.DefaultLanguage)
	}

	_, err := c.ExecContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to create SQL login: %w", err)
	}

	return c.GetSQLLogin(ctx, opts.Name)
}

// UpdateSQLLoginOptions contains options for updating a SQL login.
type UpdateSQLLoginOptions struct {
	Name                   string
	Password               *string
	DefaultDatabase        *string
	DefaultLanguage        *string
	CheckExpirationEnabled *bool
	CheckPolicyEnabled     *bool
	IsDisabled             *bool
}

// UpdateSQLLogin updates an existing SQL login.
func (c *Client) UpdateSQLLogin(ctx context.Context, opts UpdateSQLLoginOptions) (*SQLLogin, error) {
	if opts.Password != nil {
		query := fmt.Sprintf("ALTER LOGIN [%s] WITH PASSWORD = '%s'", opts.Name, *opts.Password)
		if _, err := c.ExecContext(ctx, query); err != nil {
			return nil, fmt.Errorf("failed to update SQL login password: %w", err)
		}
	}

	var alterParts []string

	if opts.DefaultDatabase != nil {
		alterParts = append(alterParts, fmt.Sprintf("DEFAULT_DATABASE = [%s]", *opts.DefaultDatabase))
	}
	if opts.DefaultLanguage != nil {
		alterParts = append(alterParts, fmt.Sprintf("DEFAULT_LANGUAGE = [%s]", *opts.DefaultLanguage))
	}
	if opts.CheckExpirationEnabled != nil {
		alterParts = append(alterParts, fmt.Sprintf("CHECK_EXPIRATION = %s", boolToOnOff(*opts.CheckExpirationEnabled)))
	}
	if opts.CheckPolicyEnabled != nil {
		alterParts = append(alterParts, fmt.Sprintf("CHECK_POLICY = %s", boolToOnOff(*opts.CheckPolicyEnabled)))
	}

	if len(alterParts) > 0 {
		query := fmt.Sprintf("ALTER LOGIN [%s] WITH ", opts.Name)
		for i, part := range alterParts {
			if i > 0 {
				query += ", "
			}
			query += part
		}
		if _, err := c.ExecContext(ctx, query); err != nil {
			return nil, fmt.Errorf("failed to update SQL login: %w", err)
		}
	}

	if opts.IsDisabled != nil {
		var query string
		if *opts.IsDisabled {
			query = fmt.Sprintf("ALTER LOGIN [%s] DISABLE", opts.Name)
		} else {
			query = fmt.Sprintf("ALTER LOGIN [%s] ENABLE", opts.Name)
		}
		if _, err := c.ExecContext(ctx, query); err != nil {
			return nil, fmt.Errorf("failed to update SQL login disabled state: %w", err)
		}
	}

	return c.GetSQLLogin(ctx, opts.Name)
}

// DropSQLLogin drops a SQL login.
func (c *Client) DropSQLLogin(ctx context.Context, name string) error {
	query := fmt.Sprintf("DROP LOGIN [%s]", name)
	_, err := c.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to drop SQL login: %w", err)
	}

	return nil
}

func boolToOnOff(b bool) string {
	if b {
		return "ON"
	}
	return "OFF"
}
