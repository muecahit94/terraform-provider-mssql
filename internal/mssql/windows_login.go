// Copyright (c) 2024 muecahit94
// SPDX-License-Identifier: MIT

package mssql

import (
	"context"
	"database/sql"
	"fmt"
)

// WindowsLogin is a login of a Windows or Active Directory user or group.
type WindowsLogin struct {
	PrincipalID     int
	Name            string
	Type            string // WINDOWS_LOGIN or WINDOWS_GROUP
	DefaultDatabase string
	DefaultLanguage string
	IsDisabled      bool
}

// GetWindowsLogin retrieves a Windows login by name. It returns nil when it does not exist.
func (c *Client) GetWindowsLogin(ctx context.Context, name string) (*WindowsLogin, error) {
	query := `
		SELECT
			principal_id,
			name,
			type_desc,
			ISNULL(default_database_name, 'master'),
			ISNULL(default_language_name, ''),
			is_disabled
		FROM sys.server_principals
		WHERE name = @p1 AND type IN ('U', 'G')`

	var login WindowsLogin
	err := c.QueryRowContext(ctx, query, name).Scan(
		&login.PrincipalID, &login.Name, &login.Type, &login.DefaultDatabase, &login.DefaultLanguage, &login.IsDisabled,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get Windows login: %w", err)
	}
	return &login, nil
}

// CreateWindowsLoginOptions contains the options of a new Windows login.
type CreateWindowsLoginOptions struct {
	Name string
	// DefaultDatabase and DefaultLanguage are left to the server defaults when empty.
	DefaultDatabase string
	DefaultLanguage string
	Disabled        bool
}

// createWindowsLoginStatement builds CREATE LOGIN ... FROM WINDOWS.
func createWindowsLoginStatement(name, defaultDatabase, defaultLanguage string) string {
	statement := "CREATE LOGIN " + quoteName(name) + " FROM WINDOWS"
	var options []string
	if defaultDatabase != "" {
		options = append(options, "DEFAULT_DATABASE = "+quoteName(defaultDatabase))
	}
	if defaultLanguage != "" {
		options = append(options, "DEFAULT_LANGUAGE = "+quoteName(defaultLanguage))
	}
	for i, option := range options {
		if i == 0 {
			statement += " WITH " + option
		} else {
			statement += ", " + option
		}
	}
	return statement
}

// CreateWindowsLogin creates the login of a Windows user or group.
func (c *Client) CreateWindowsLogin(ctx context.Context, opts CreateWindowsLoginOptions) (*WindowsLogin, error) {
	if _, err := c.ExecContext(ctx, createWindowsLoginStatement(opts.Name, opts.DefaultDatabase, opts.DefaultLanguage)); err != nil {
		return nil, fmt.Errorf("failed to create Windows login: %w", err)
	}
	if opts.Disabled {
		if err := c.SetWindowsLoginDisabled(ctx, opts.Name, true); err != nil {
			// Do not leave a half-created login behind: Terraform has no state for it.
			_ = c.DropWindowsLogin(ctx, opts.Name)
			return nil, err
		}
	}
	return c.GetWindowsLogin(ctx, opts.Name)
}

// alterLoginStatement builds an ALTER LOGIN statement for one option.
func alterLoginStatement(name, option string) string {
	return "ALTER LOGIN " + quoteName(name) + " " + option
}

// UpdateWindowsLoginOptions contains the settings that can change; a nil field is left alone.
type UpdateWindowsLoginOptions struct {
	DefaultDatabase *string
	DefaultLanguage *string
	Disabled        *bool
}

// UpdateWindowsLogin changes the settings of a Windows login.
func (c *Client) UpdateWindowsLogin(ctx context.Context, name string, opts UpdateWindowsLoginOptions) error {
	if opts.DefaultDatabase != nil {
		if _, err := c.ExecContext(ctx, alterLoginStatement(name, "WITH DEFAULT_DATABASE = "+quoteName(*opts.DefaultDatabase))); err != nil {
			return fmt.Errorf("failed to change the default database of the login: %w", err)
		}
	}
	if opts.DefaultLanguage != nil {
		if _, err := c.ExecContext(ctx, alterLoginStatement(name, "WITH DEFAULT_LANGUAGE = "+quoteName(*opts.DefaultLanguage))); err != nil {
			return fmt.Errorf("failed to change the default language of the login: %w", err)
		}
	}
	if opts.Disabled != nil {
		return c.SetWindowsLoginDisabled(ctx, name, *opts.Disabled)
	}
	return nil
}

// SetWindowsLoginDisabled disables or enables a Windows login.
func (c *Client) SetWindowsLoginDisabled(ctx context.Context, name string, disabled bool) error {
	option := "ENABLE"
	if disabled {
		option = "DISABLE"
	}
	if _, err := c.ExecContext(ctx, alterLoginStatement(name, option)); err != nil {
		return fmt.Errorf("failed to %s the login: %w", map[bool]string{true: "disable", false: "enable"}[disabled], err)
	}
	return nil
}

// DropWindowsLogin drops a Windows login.
func (c *Client) DropWindowsLogin(ctx context.Context, name string) error {
	if _, err := c.ExecContext(ctx, "DROP LOGIN "+quoteName(name)); err != nil {
		return fmt.Errorf("failed to drop Windows login: %w", err)
	}
	return nil
}
