// Copyright (c) 2024 muecahit94
// SPDX-License-Identifier: MIT

package mssql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// Permission represents a permission grant.
type Permission struct {
	PrincipalID    int
	PrincipalName  string
	PermissionName string
	StateDesc      string // GRANT, DENY, REVOKE
	ObjectType     string // DATABASE, SCHEMA, TABLE, etc.
	ObjectName     string
}

// DatabasePermission represents a database-level permission.
type DatabasePermission struct {
	PrincipalID     int
	PrincipalName   string
	PermissionName  string
	StateDesc       string
	DatabaseID      int
	WithGrantOption bool
}

// GetDatabasePermission retrieves a specific database permission.
func (c *Client) GetDatabasePermission(ctx context.Context, databaseName, principalName, permission string) (*DatabasePermission, error) {
	query := `
		SELECT 
			dp.principal_id,
			dp.name,
			perm.permission_name,
			perm.state_desc,
			DB_ID(),
			CASE WHEN perm.state = 'W' THEN 1 ELSE 0 END
		FROM sys.database_permissions perm
		INNER JOIN sys.database_principals dp ON perm.grantee_principal_id = dp.principal_id
		WHERE dp.name = @p1 
			AND perm.permission_name = @p2 
			AND perm.class = 0`

	row, err := c.QueryRowInDatabaseContext(ctx, databaseName, query, principalName, strings.ToUpper(permission))
	if err != nil {
		return nil, err
	}

	var perm DatabasePermission
	err = row.Scan(
		&perm.PrincipalID,
		&perm.PrincipalName,
		&perm.PermissionName,
		&perm.StateDesc,
		&perm.DatabaseID,
		&perm.WithGrantOption,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get database permission: %w", err)
	}

	return &perm, nil
}

// ListDatabasePermissions retrieves all database permissions for a principal.
func (c *Client) ListDatabasePermissions(ctx context.Context, databaseName, principalName string) ([]DatabasePermission, error) {
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
			perm.permission_name,
			perm.state_desc,
			DB_ID(),
			CASE WHEN perm.state = 'W' THEN 1 ELSE 0 END
		FROM sys.database_permissions perm
		INNER JOIN sys.database_principals dp ON perm.grantee_principal_id = dp.principal_id
		WHERE dp.name = @p1 AND perm.class = 0
		ORDER BY perm.permission_name`

	rows, err := conn.QueryContext(ctx, query, principalName)
	if err != nil {
		return nil, fmt.Errorf("failed to list database permissions: %w", err)
	}
	defer rows.Close()

	var perms []DatabasePermission
	for rows.Next() {
		var perm DatabasePermission
		if err := rows.Scan(
			&perm.PrincipalID,
			&perm.PrincipalName,
			&perm.PermissionName,
			&perm.StateDesc,
			&perm.DatabaseID,
			&perm.WithGrantOption,
		); err != nil {
			return nil, fmt.Errorf("failed to scan database permission: %w", err)
		}
		perms = append(perms, perm)
	}

	return perms, rows.Err()
}

// GrantDatabasePermission grants a database-level permission.
func (c *Client) GrantDatabasePermission(ctx context.Context, databaseName, principalName, permission string, withGrantOption bool) error {
	query := fmt.Sprintf("GRANT %s TO [%s]", strings.ToUpper(permission), principalName)
	if withGrantOption {
		query += " WITH GRANT OPTION"
	}

	err := c.ExecInDatabaseContext(ctx, databaseName, query)
	if err != nil {
		return fmt.Errorf("failed to grant database permission: %w", err)
	}

	return nil
}

// RevokeDatabasePermission revokes a database-level permission.
func (c *Client) RevokeDatabasePermission(ctx context.Context, databaseName, principalName, permission string) error {
	query := fmt.Sprintf("REVOKE %s FROM [%s]", strings.ToUpper(permission), principalName)
	err := c.ExecInDatabaseContext(ctx, databaseName, query)
	if err != nil {
		return fmt.Errorf("failed to revoke database permission: %w", err)
	}

	return nil
}

// SchemaPermission represents a schema-level permission.
type SchemaPermission struct {
	PrincipalID     int
	PrincipalName   string
	PermissionName  string
	StateDesc       string
	SchemaName      string
	DatabaseID      int
	WithGrantOption bool
}

// GetSchemaPermission retrieves a specific schema permission.
func (c *Client) GetSchemaPermission(ctx context.Context, databaseName, schemaName, principalName, permission string) (*SchemaPermission, error) {
	query := `
		SELECT 
			dp.principal_id,
			dp.name,
			perm.permission_name,
			perm.state_desc,
			s.name,
			DB_ID(),
			CASE WHEN perm.state = 'W' THEN 1 ELSE 0 END
		FROM sys.database_permissions perm
		INNER JOIN sys.database_principals dp ON perm.grantee_principal_id = dp.principal_id
		INNER JOIN sys.schemas s ON perm.major_id = s.schema_id
		WHERE dp.name = @p1 
			AND perm.permission_name = @p2 
			AND s.name = @p3
			AND perm.class = 3`

	row, err := c.QueryRowInDatabaseContext(ctx, databaseName, query, principalName, strings.ToUpper(permission), schemaName)
	if err != nil {
		return nil, err
	}

	var perm SchemaPermission
	err = row.Scan(
		&perm.PrincipalID,
		&perm.PrincipalName,
		&perm.PermissionName,
		&perm.StateDesc,
		&perm.SchemaName,
		&perm.DatabaseID,
		&perm.WithGrantOption,
	)
	if err == nil {
		return &perm, nil
	}
	if err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get schema permission: %w", err)
	}

	// Permission not found. Check if the principal is the owner of the schema.
	// If so, they implicitly have the permission.
	ownerQuery := `
		SELECT 
			dp.principal_id,
			dp.name
		FROM sys.schemas s
		INNER JOIN sys.database_principals dp ON s.principal_id = dp.principal_id
		WHERE s.name = @p1 AND dp.name = @p2`

	ownerRow, err := c.QueryRowInDatabaseContext(ctx, databaseName, ownerQuery, schemaName, principalName)
	if err != nil {
		return nil, fmt.Errorf("failed to check schema ownership: %w", err)
	}

	var ownerID int
	var ownerName string
	err = ownerRow.Scan(&ownerID, &ownerName)
	if err == sql.ErrNoRows {
		// Not the owner, and no explicit permission found
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan schema owner: %w", err)
	}

	// Principal is the owner, so they have the permission implicitly.
	// Return a virtual permission object.
	return &SchemaPermission{
		PrincipalID:     ownerID,
		PrincipalName:   ownerName,
		PermissionName:  strings.ToUpper(permission),
		StateDesc:       "GRANT", // Implicit grant
		SchemaName:      schemaName,
		DatabaseID:      0,    // Unknown/Irrelevant for virtual
		WithGrantOption: true, // Owners effectively have grant option (CONTROL)
	}, nil
}

// ListSchemaPermissions retrieves all schema permissions for a principal.
func (c *Client) ListSchemaPermissions(ctx context.Context, databaseName, schemaName, principalName string) ([]SchemaPermission, error) {
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
			perm.permission_name,
			perm.state_desc,
			s.name,
			DB_ID(),
			CASE WHEN perm.state = 'W' THEN 1 ELSE 0 END
		FROM sys.database_permissions perm
		INNER JOIN sys.database_principals dp ON perm.grantee_principal_id = dp.principal_id
		INNER JOIN sys.schemas s ON perm.major_id = s.schema_id
		WHERE dp.name = @p1 AND s.name = @p2 AND perm.class = 3
		ORDER BY perm.permission_name`

	rows, err := conn.QueryContext(ctx, query, principalName, schemaName)
	if err != nil {
		return nil, fmt.Errorf("failed to list schema permissions: %w", err)
	}
	defer rows.Close()

	var perms []SchemaPermission
	for rows.Next() {
		var perm SchemaPermission
		if err := rows.Scan(
			&perm.PrincipalID,
			&perm.PrincipalName,
			&perm.PermissionName,
			&perm.StateDesc,
			&perm.SchemaName,
			&perm.DatabaseID,
			&perm.WithGrantOption,
		); err != nil {
			return nil, fmt.Errorf("failed to scan schema permission: %w", err)
		}
		perms = append(perms, perm)
	}

	return perms, rows.Err()
}

// GrantSchemaPermission grants a schema-level permission.
func (c *Client) GrantSchemaPermission(ctx context.Context, databaseName, schemaName, principalName, permission string, withGrantOption bool) error {
	query := fmt.Sprintf("GRANT %s ON SCHEMA::[%s] TO [%s]", strings.ToUpper(permission), schemaName, principalName)
	if withGrantOption {
		query += " WITH GRANT OPTION"
	}

	err := c.ExecInDatabaseContext(ctx, databaseName, query)
	if err != nil {
		return fmt.Errorf("failed to grant schema permission: %w", err)
	}

	return nil
}

// RevokeSchemaPermission revokes a schema-level permission.
// CASCADE is used to also revoke any permissions that were granted by this principal.
func (c *Client) RevokeSchemaPermission(ctx context.Context, databaseName, schemaName, principalName, permission string) error {
	query := fmt.Sprintf("REVOKE %s ON SCHEMA::[%s] FROM [%s] CASCADE", strings.ToUpper(permission), schemaName, principalName)
	err := c.ExecInDatabaseContext(ctx, databaseName, query)
	if err != nil {
		return fmt.Errorf("failed to revoke schema permission: %w", err)
	}

	return nil
}

// ServerPermission represents a server-level permission.
type ServerPermission struct {
	PrincipalID     int
	PrincipalName   string
	PermissionName  string
	StateDesc       string
	WithGrantOption bool
}

// GetServerPermission retrieves a specific server permission.
func (c *Client) GetServerPermission(ctx context.Context, principalName, permission string) (*ServerPermission, error) {
	query := `
		SELECT 
			sp.principal_id,
			sp.name,
			perm.permission_name,
			perm.state_desc,
			CASE WHEN perm.state = 'W' THEN 1 ELSE 0 END
		FROM sys.server_permissions perm
		INNER JOIN sys.server_principals sp ON perm.grantee_principal_id = sp.principal_id
		WHERE sp.name = @p1 
			AND perm.permission_name = @p2 
			AND perm.class = 100`
	row := c.QueryRowContext(ctx, query, principalName, strings.ToUpper(permission))

	var perm ServerPermission
	err := row.Scan(
		&perm.PrincipalID,
		&perm.PrincipalName,
		&perm.PermissionName,
		&perm.StateDesc,
		&perm.WithGrantOption,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get server permission: %w", err)
	}

	return &perm, nil
}

// ListServerPermissions retrieves all server permissions for a principal.
func (c *Client) ListServerPermissions(ctx context.Context, principalName string) ([]ServerPermission, error) {
	query := `
		SELECT 
			sp.principal_id,
			sp.name,
			perm.permission_name,
			perm.state_desc,
			CASE WHEN perm.state = 'W' THEN 1 ELSE 0 END
		FROM sys.server_permissions perm
		INNER JOIN sys.server_principals sp ON perm.grantee_principal_id = sp.principal_id
		WHERE sp.name = @p1 AND perm.class = 100
		ORDER BY perm.permission_name`
	rows, err := c.QueryContext(ctx, query, principalName)
	if err != nil {
		return nil, fmt.Errorf("failed to list server permissions: %w", err)
	}
	defer rows.Close()

	var perms []ServerPermission
	for rows.Next() {
		var perm ServerPermission
		if err := rows.Scan(
			&perm.PrincipalID,
			&perm.PrincipalName,
			&perm.PermissionName,
			&perm.StateDesc,
			&perm.WithGrantOption,
		); err != nil {
			return nil, fmt.Errorf("failed to scan server permission: %w", err)
		}
		perms = append(perms, perm)
	}

	return perms, rows.Err()
}

// GrantServerPermission grants a server-level permission.
func (c *Client) GrantServerPermission(ctx context.Context, principalName, permission string, withGrantOption bool) error {
	query := fmt.Sprintf("GRANT %s TO [%s]", strings.ToUpper(permission), principalName)
	if withGrantOption {
		query += " WITH GRANT OPTION"
	}

	_, err := c.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to grant server permission: %w", err)
	}

	return nil
}

// RevokeServerPermission revokes a server-level permission.
func (c *Client) RevokeServerPermission(ctx context.Context, principalName, permission string) error {
	query := fmt.Sprintf("REVOKE %s FROM [%s]", strings.ToUpper(permission), principalName)
	_, err := c.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to revoke server permission: %w", err)
	}

	return nil
}
