// Copyright (c) 2024 muecahit94
// SPDX-License-Identifier: MIT

package mssql

import (
	"context"
	"database/sql"
	"fmt"
)

// DatabaseRole represents a database role.
type DatabaseRole struct {
	PrincipalID    int
	Name           string
	DatabaseID     int
	OwnerName      string
	IsFixedRole    bool
	IsDatabaseRole bool
}

// GetDatabaseRole retrieves a database role by name.
func (c *Client) GetDatabaseRole(ctx context.Context, databaseName, roleName string) (*DatabaseRole, error) {
	query := `
		SELECT
			dp.principal_id,
			dp.name,
			DB_ID() as database_id,
			ISNULL(owner.name, ''),
			dp.is_fixed_role,
			CASE WHEN dp.type = 'R' THEN 1 ELSE 0 END
		FROM sys.database_principals dp
		LEFT JOIN sys.database_principals owner ON dp.owning_principal_id = owner.principal_id
		WHERE dp.name = @p1 AND dp.type = 'R'`

	row, err := c.QueryRowInDatabaseContext(ctx, databaseName, query, roleName)
	if err != nil {
		return nil, err
	}

	var role DatabaseRole
	err = row.Scan(
		&role.PrincipalID,
		&role.Name,
		&role.DatabaseID,
		&role.OwnerName,
		&role.IsFixedRole,
		&role.IsDatabaseRole,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get database role: %w", err)
	}

	return &role, nil
}

// GetDatabaseRoleByID retrieves a database role by principal ID.
func (c *Client) GetDatabaseRoleByID(ctx context.Context, databaseName string, principalID int) (*DatabaseRole, error) {
	query := `
		SELECT
			dp.principal_id,
			dp.name,
			DB_ID() as database_id,
			ISNULL(owner.name, ''),
			dp.is_fixed_role,
			CASE WHEN dp.type = 'R' THEN 1 ELSE 0 END
		FROM sys.database_principals dp
		LEFT JOIN sys.database_principals owner ON dp.owning_principal_id = owner.principal_id
		WHERE dp.principal_id = @p1 AND dp.type = 'R'`

	row, err := c.QueryRowInDatabaseContext(ctx, databaseName, query, principalID)
	if err != nil {
		return nil, err
	}

	var role DatabaseRole
	err = row.Scan(
		&role.PrincipalID,
		&role.Name,
		&role.DatabaseID,
		&role.OwnerName,
		&role.IsFixedRole,
		&role.IsDatabaseRole,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get database role: %w", err)
	}

	return &role, nil
}

// ListDatabaseRoles retrieves all database roles.
func (c *Client) ListDatabaseRoles(ctx context.Context, databaseName string) ([]DatabaseRole, error) {
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
			ISNULL(owner.name, ''),
			dp.is_fixed_role,
			CASE WHEN dp.type = 'R' THEN 1 ELSE 0 END
		FROM sys.database_principals dp
		LEFT JOIN sys.database_principals owner ON dp.owning_principal_id = owner.principal_id
		WHERE dp.type = 'R'
		ORDER BY dp.name`

	rows, err := conn.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list database roles: %w", err)
	}
	defer rows.Close()

	var roles []DatabaseRole
	for rows.Next() {
		var role DatabaseRole
		if err := rows.Scan(
			&role.PrincipalID,
			&role.Name,
			&role.DatabaseID,
			&role.OwnerName,
			&role.IsFixedRole,
			&role.IsDatabaseRole,
		); err != nil {
			return nil, fmt.Errorf("failed to scan database role: %w", err)
		}
		roles = append(roles, role)
	}

	return roles, rows.Err()
}

// CreateDatabaseRoleOptions contains options for creating a database role.
type CreateDatabaseRoleOptions struct {
	DatabaseName string
	RoleName     string
	OwnerName    string
}

// CreateDatabaseRole creates a new database role.
func (c *Client) CreateDatabaseRole(ctx context.Context, opts CreateDatabaseRoleOptions) (*DatabaseRole, error) {
	query := fmt.Sprintf("CREATE ROLE [%s]", opts.RoleName)
	if opts.OwnerName != "" {
		query += fmt.Sprintf(" AUTHORIZATION [%s]", opts.OwnerName)
	}

	err := c.ExecInDatabaseContext(ctx, opts.DatabaseName, query)
	if err != nil {
		return nil, fmt.Errorf("failed to create database role: %w", err)
	}

	role, err := c.GetDatabaseRole(ctx, opts.DatabaseName, opts.RoleName)
	if err != nil {
		return nil, err
	}
	if role == nil {
		return nil, fmt.Errorf("role was created but could not be retrieved")
	}
	return role, nil
}

// UpdateDatabaseRoleOptions contains options for updating a database role.
type UpdateDatabaseRoleOptions struct {
	DatabaseName string
	RoleName     string
	NewOwnerName *string
}

// UpdateDatabaseRole updates an existing database role.
func (c *Client) UpdateDatabaseRole(ctx context.Context, opts UpdateDatabaseRoleOptions) (*DatabaseRole, error) {
	if err := c.UseDatabase(ctx, opts.DatabaseName); err != nil {
		return nil, err
	}

	if opts.NewOwnerName != nil {
		query := fmt.Sprintf("ALTER AUTHORIZATION ON ROLE::[%s] TO [%s]", opts.RoleName, *opts.NewOwnerName)
		if _, err := c.ExecContext(ctx, query); err != nil {
			return nil, fmt.Errorf("failed to update database role owner: %w", err)
		}
	}

	return c.GetDatabaseRole(ctx, opts.DatabaseName, opts.RoleName)
}

// DropDatabaseRole drops a database role.
func (c *Client) DropDatabaseRole(ctx context.Context, databaseName, roleName string) error {
	if err := c.UseDatabase(ctx, databaseName); err != nil {
		return err
	}

	query := fmt.Sprintf("DROP ROLE IF EXISTS [%s]", roleName)
	_, err := c.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to drop database role: %w", err)
	}

	return nil
}

// DatabaseRoleMember represents a role membership.
type DatabaseRoleMember struct {
	RoleID     int
	RoleName   string
	MemberID   int
	MemberName string
	DatabaseID int
}

// GetDatabaseRoleMember retrieves a role membership.
func (c *Client) GetDatabaseRoleMember(ctx context.Context, databaseName, roleName, memberName string) (*DatabaseRoleMember, error) {
	query := `
		SELECT
			role_dp.principal_id,
			role_dp.name,
			member_dp.principal_id,
			member_dp.name,
			DB_ID()
		FROM sys.database_role_members drm
		INNER JOIN sys.database_principals role_dp ON drm.role_principal_id = role_dp.principal_id
		INNER JOIN sys.database_principals member_dp ON drm.member_principal_id = member_dp.principal_id
		WHERE role_dp.name = @p1 AND member_dp.name = @p2`

	row, err := c.QueryRowInDatabaseContext(ctx, databaseName, query, roleName, memberName)
	if err != nil {
		return nil, err
	}

	var member DatabaseRoleMember
	err = row.Scan(
		&member.RoleID,
		&member.RoleName,
		&member.MemberID,
		&member.MemberName,
		&member.DatabaseID,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get database role member: %w", err)
	}

	return &member, nil
}

// AddDatabaseRoleMember adds a member to a database role.
func (c *Client) AddDatabaseRoleMember(ctx context.Context, databaseName, roleName, memberName string) error {
	query := fmt.Sprintf("ALTER ROLE [%s] ADD MEMBER [%s]", roleName, memberName)
	err := c.ExecInDatabaseContext(ctx, databaseName, query)
	if err != nil {
		return fmt.Errorf("failed to add database role member: %w", err)
	}

	return nil
}

// RemoveDatabaseRoleMember removes a member from a database role.
func (c *Client) RemoveDatabaseRoleMember(ctx context.Context, databaseName, roleName, memberName string) error {
	query := fmt.Sprintf("ALTER ROLE [%s] DROP MEMBER [%s]", roleName, memberName)
	err := c.ExecInDatabaseContext(ctx, databaseName, query)
	if err != nil {
		return fmt.Errorf("failed to remove database role member: %w", err)
	}

	return nil
}

// ServerRole represents a server role.
type ServerRole struct {
	PrincipalID int
	Name        string
	OwnerName   string
	IsFixedRole bool
}

// GetServerRole retrieves a server role by name.
func (c *Client) GetServerRole(ctx context.Context, roleName string) (*ServerRole, error) {
	query := `
		SELECT
			sp.principal_id,
			sp.name,
			ISNULL(owner.name, ''),
			sp.is_fixed_role
		FROM sys.server_principals sp
		LEFT JOIN sys.server_principals owner ON sp.owning_principal_id = owner.principal_id
		WHERE sp.name = @p1 AND sp.type = 'R'`
	row := c.QueryRowContext(ctx, query, roleName)

	var role ServerRole
	err := row.Scan(
		&role.PrincipalID,
		&role.Name,
		&role.OwnerName,
		&role.IsFixedRole,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get server role: %w", err)
	}

	return &role, nil
}

// GetServerRoleByID retrieves a server role by principal ID.
func (c *Client) GetServerRoleByID(ctx context.Context, principalID int) (*ServerRole, error) {
	query := `
		SELECT
			sp.principal_id,
			sp.name,
			ISNULL(owner.name, ''),
			sp.is_fixed_role
		FROM sys.server_principals sp
		LEFT JOIN sys.server_principals owner ON sp.owning_principal_id = owner.principal_id
		WHERE sp.principal_id = @p1 AND sp.type = 'R'`
	row := c.QueryRowContext(ctx, query, principalID)

	var role ServerRole
	err := row.Scan(
		&role.PrincipalID,
		&role.Name,
		&role.OwnerName,
		&role.IsFixedRole,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get server role: %w", err)
	}

	return &role, nil
}

// ListServerRoles retrieves all server roles.
func (c *Client) ListServerRoles(ctx context.Context) ([]ServerRole, error) {
	query := `
		SELECT
			sp.principal_id,
			sp.name,
			ISNULL(owner.name, ''),
			sp.is_fixed_role
		FROM sys.server_principals sp
		LEFT JOIN sys.server_principals owner ON sp.owning_principal_id = owner.principal_id
		WHERE sp.type = 'R'
		ORDER BY sp.name`
	rows, err := c.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list server roles: %w", err)
	}
	defer rows.Close()

	var roles []ServerRole
	for rows.Next() {
		var role ServerRole
		if err := rows.Scan(
			&role.PrincipalID,
			&role.Name,
			&role.OwnerName,
			&role.IsFixedRole,
		); err != nil {
			return nil, fmt.Errorf("failed to scan server role: %w", err)
		}
		roles = append(roles, role)
	}

	return roles, rows.Err()
}

// CreateServerRoleOptions contains options for creating a server role.
type CreateServerRoleOptions struct {
	RoleName  string
	OwnerName string
}

// CreateServerRole creates a new server role.
func (c *Client) CreateServerRole(ctx context.Context, opts CreateServerRoleOptions) (*ServerRole, error) {
	query := fmt.Sprintf("CREATE SERVER ROLE [%s]", opts.RoleName)
	if opts.OwnerName != "" {
		query += fmt.Sprintf(" AUTHORIZATION [%s]", opts.OwnerName)
	}

	_, err := c.ExecContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to create server role: %w", err)
	}

	return c.GetServerRole(ctx, opts.RoleName)
}

// DropServerRole drops a server role.
func (c *Client) DropServerRole(ctx context.Context, roleName string) error {
	query := fmt.Sprintf("DROP SERVER ROLE [%s]", roleName)
	_, err := c.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to drop server role: %w", err)
	}

	return nil
}

// ServerRoleMember represents a server role membership.
type ServerRoleMember struct {
	RoleID     int
	RoleName   string
	MemberID   int
	MemberName string
}

// GetServerRoleMember retrieves a server role membership.
func (c *Client) GetServerRoleMember(ctx context.Context, roleName, memberName string) (*ServerRoleMember, error) {
	query := `
		SELECT
			role_sp.principal_id,
			role_sp.name,
			member_sp.principal_id,
			member_sp.name
		FROM sys.server_role_members srm
		INNER JOIN sys.server_principals role_sp ON srm.role_principal_id = role_sp.principal_id
		INNER JOIN sys.server_principals member_sp ON srm.member_principal_id = member_sp.principal_id
		WHERE role_sp.name = @p1 AND member_sp.name = @p2`
	row := c.QueryRowContext(ctx, query, roleName, memberName)

	var member ServerRoleMember
	err := row.Scan(
		&member.RoleID,
		&member.RoleName,
		&member.MemberID,
		&member.MemberName,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get server role member: %w", err)
	}

	return &member, nil
}

// AddServerRoleMember adds a member to a server role.
func (c *Client) AddServerRoleMember(ctx context.Context, roleName, memberName string) error {
	query := fmt.Sprintf("ALTER SERVER ROLE [%s] ADD MEMBER [%s]", roleName, memberName)
	_, err := c.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to add server role member: %w", err)
	}

	return nil
}

// RemoveServerRoleMember removes a member from a server role.
func (c *Client) RemoveServerRoleMember(ctx context.Context, roleName, memberName string) error {
	query := fmt.Sprintf("ALTER SERVER ROLE [%s] DROP MEMBER [%s]", roleName, memberName)
	_, err := c.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to remove server role member: %w", err)
	}

	return nil
}
