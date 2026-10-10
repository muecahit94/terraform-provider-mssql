// Copyright (c) 2024 muecahit94
// SPDX-License-Identifier: MIT

package mssql

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"
)

// ObjectPermission is a permission granted on a securable that lives in a schema, such as a table,
// a view or a procedure, or on one column of it.
type ObjectPermission struct {
	PrincipalName   string
	PermissionName  string
	SchemaName      string
	ObjectName      string
	ColumnName      string
	State           string
	WithGrantOption bool
}

// permissionNamePattern matches permission names such as SELECT or VIEW DEFINITION. The name is part
// of the statement text and cannot be passed as a parameter, so it is validated first.
var permissionNamePattern = regexp.MustCompile(`^[A-Z]+( [A-Z]+)*$`)

// NormalizeObjectPermission upper-cases a permission name and rejects anything that is not a
// plain permission keyword sequence.
func NormalizeObjectPermission(permission string) (string, error) {
	normalized := strings.ToUpper(strings.TrimSpace(permission))
	if !permissionNamePattern.MatchString(normalized) {
		return "", fmt.Errorf("invalid permission %q: expected a permission name such as SELECT or VIEW DEFINITION", permission)
	}
	return normalized, nil
}

// objectSecurable returns OBJECT::[schema].[object], followed by ([column]) for a column-level permission.
func objectSecurable(schemaName, objectName, columnName string) string {
	securable := "OBJECT::" + quoteName(schemaName) + "." + quoteName(objectName)
	if columnName != "" {
		securable += " (" + quoteName(columnName) + ")"
	}
	return securable
}

// GrantObjectPermissionStatement builds the GRANT statement of an object or column permission.
func GrantObjectPermissionStatement(schemaName, objectName, columnName, principalName, permission string, withGrantOption bool) (string, error) {
	permission, err := NormalizeObjectPermission(permission)
	if err != nil {
		return "", err
	}
	statement := "GRANT " + permission + " ON " + objectSecurable(schemaName, objectName, columnName) + " TO " + quoteName(principalName)
	if withGrantOption {
		statement += " WITH GRANT OPTION"
	}
	return statement, nil
}

// RevokeObjectPermissionStatement builds the REVOKE statement of an object or column permission.
// CASCADE also revokes what the principal granted on to others.
func RevokeObjectPermissionStatement(schemaName, objectName, columnName, principalName, permission string) (string, error) {
	permission, err := NormalizeObjectPermission(permission)
	if err != nil {
		return "", err
	}
	return "REVOKE " + permission + " ON " + objectSecurable(schemaName, objectName, columnName) + " FROM " + quoteName(principalName) + " CASCADE", nil
}

// execInDatabase runs a statement in the context of a database without a USE statement, which
// would leave the pooled connection in another database.
func (c *Client) execInDatabase(ctx context.Context, database, statement string) error {
	_, err := c.ExecContext(ctx, "EXEC "+quoteName(database)+".sys.sp_executesql @p1", statement)
	return err
}

// objectPermissionQuery selects the permissions of class OBJECT_OR_COLUMN. The catalog views are
// qualified with the database name, so no USE is needed and the database collation applies.
func objectPermissionQuery(database string) string {
	db := quoteName(database)
	return `
		SELECT
			g.name,
			p.permission_name,
			s.name,
			o.name,
			ISNULL(col.name, ''),
			p.state
		FROM ` + db + `.sys.database_permissions p
		INNER JOIN ` + db + `.sys.database_principals g ON g.principal_id = p.grantee_principal_id
		INNER JOIN ` + db + `.sys.all_objects o ON o.object_id = p.major_id
		INNER JOIN ` + db + `.sys.schemas s ON s.schema_id = o.schema_id
		LEFT JOIN ` + db + `.sys.all_columns col ON col.object_id = p.major_id AND col.column_id = p.minor_id AND p.minor_id > 0
		WHERE p.class = 1`
}

// GetObjectPermission retrieves one permission, or nil when it does not exist. A DENY is returned
// with State "D": the caller decides what to do with it, because granting over a DENY removes it.
func (c *Client) GetObjectPermission(ctx context.Context, database, schemaName, objectName, columnName, principalName, permission string) (*ObjectPermission, error) {
	permission, err := NormalizeObjectPermission(permission)
	if err != nil {
		return nil, err
	}

	query := objectPermissionQuery(database) + `
			AND g.name = @p1
			AND p.permission_name = @p2
			AND s.name = @p3
			AND o.name = @p4
			AND ISNULL(col.name, '') = @p5`
	row := c.QueryRowContext(ctx, query, principalName, permission, schemaName, objectName, columnName)

	var perm ObjectPermission
	err = row.Scan(&perm.PrincipalName, &perm.PermissionName, &perm.SchemaName, &perm.ObjectName, &perm.ColumnName, &perm.State)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get object permission: %w", err)
	}
	perm.WithGrantOption = perm.State == "W"
	return &perm, nil
}

// GrantObjectPermission grants a permission on an object or a column.
func (c *Client) GrantObjectPermission(ctx context.Context, database, schemaName, objectName, columnName, principalName, permission string, withGrantOption bool) error {
	statement, err := GrantObjectPermissionStatement(schemaName, objectName, columnName, principalName, permission, withGrantOption)
	if err != nil {
		return err
	}
	if err := c.execInDatabase(ctx, database, statement); err != nil {
		return fmt.Errorf("failed to grant object permission: %w", err)
	}
	return nil
}

// RevokeObjectPermission revokes a permission on an object or a column.
func (c *Client) RevokeObjectPermission(ctx context.Context, database, schemaName, objectName, columnName, principalName, permission string) error {
	statement, err := RevokeObjectPermissionStatement(schemaName, objectName, columnName, principalName, permission)
	if err != nil {
		return err
	}
	if err := c.execInDatabase(ctx, database, statement); err != nil {
		return fmt.Errorf("failed to revoke object permission: %w", err)
	}
	return nil
}
