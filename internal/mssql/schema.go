// Copyright (c) 2024 muecahit94
// SPDX-License-Identifier: MIT

package mssql

import (
	"context"
	"database/sql"
	"fmt"
)

// Schema represents a database schema.
type Schema struct {
	SchemaID   int
	Name       string
	OwnerName  string
	DatabaseID int
}

// GetSchema retrieves a schema by name.
func (c *Client) GetSchema(ctx context.Context, databaseName, schemaName string) (*Schema, error) {
	if err := c.UseDatabase(ctx, databaseName); err != nil {
		return nil, err
	}

	query := `
		SELECT 
			s.schema_id, 
			s.name, 
			dp.name as owner_name,
			DB_ID()
		FROM sys.schemas s
		INNER JOIN sys.database_principals dp ON s.principal_id = dp.principal_id
		WHERE s.name = @p1`
	row := c.QueryRowContext(ctx, query, schemaName)

	var schema Schema
	err := row.Scan(
		&schema.SchemaID,
		&schema.Name,
		&schema.OwnerName,
		&schema.DatabaseID,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get schema: %w", err)
	}

	return &schema, nil
}

// GetSchemaByID retrieves a schema by ID.
func (c *Client) GetSchemaByID(ctx context.Context, databaseName string, schemaID int) (*Schema, error) {
	if err := c.UseDatabase(ctx, databaseName); err != nil {
		return nil, err
	}

	query := `
		SELECT 
			s.schema_id, 
			s.name, 
			dp.name as owner_name,
			DB_ID()
		FROM sys.schemas s
		INNER JOIN sys.database_principals dp ON s.principal_id = dp.principal_id
		WHERE s.schema_id = @p1`
	row := c.QueryRowContext(ctx, query, schemaID)

	var schema Schema
	err := row.Scan(
		&schema.SchemaID,
		&schema.Name,
		&schema.OwnerName,
		&schema.DatabaseID,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get schema: %w", err)
	}

	return &schema, nil
}

// ListSchemas retrieves all schemas from a database.
func (c *Client) ListSchemas(ctx context.Context, databaseName string) ([]Schema, error) {
	if err := c.UseDatabase(ctx, databaseName); err != nil {
		return nil, err
	}

	query := `
		SELECT 
			s.schema_id, 
			s.name, 
			dp.name as owner_name,
			DB_ID()
		FROM sys.schemas s
		INNER JOIN sys.database_principals dp ON s.principal_id = dp.principal_id
		ORDER BY s.name`
	rows, err := c.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list schemas: %w", err)
	}
	defer rows.Close()

	var schemas []Schema
	for rows.Next() {
		var schema Schema
		if err := rows.Scan(
			&schema.SchemaID,
			&schema.Name,
			&schema.OwnerName,
			&schema.DatabaseID,
		); err != nil {
			return nil, fmt.Errorf("failed to scan schema: %w", err)
		}
		schemas = append(schemas, schema)
	}

	return schemas, rows.Err()
}

// CreateSchemaOptions contains options for creating a schema.
type CreateSchemaOptions struct {
	DatabaseName string
	SchemaName   string
	OwnerName    string
}

// CreateSchema creates a new schema.
func (c *Client) CreateSchema(ctx context.Context, opts CreateSchemaOptions) (*Schema, error) {
	if err := c.UseDatabase(ctx, opts.DatabaseName); err != nil {
		return nil, err
	}

	query := fmt.Sprintf("CREATE SCHEMA [%s]", opts.SchemaName)
	if opts.OwnerName != "" {
		query += fmt.Sprintf(" AUTHORIZATION [%s]", opts.OwnerName)
	}

	_, err := c.ExecContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to create schema: %w", err)
	}

	return c.GetSchema(ctx, opts.DatabaseName, opts.SchemaName)
}

// UpdateSchemaOptions contains options for updating a schema.
type UpdateSchemaOptions struct {
	DatabaseName string
	SchemaName   string
	NewOwnerName *string
}

// UpdateSchema updates an existing schema.
func (c *Client) UpdateSchema(ctx context.Context, opts UpdateSchemaOptions) (*Schema, error) {
	if err := c.UseDatabase(ctx, opts.DatabaseName); err != nil {
		return nil, err
	}

	if opts.NewOwnerName != nil {
		query := fmt.Sprintf("ALTER AUTHORIZATION ON SCHEMA::[%s] TO [%s]", opts.SchemaName, *opts.NewOwnerName)
		if _, err := c.ExecContext(ctx, query); err != nil {
			return nil, fmt.Errorf("failed to update schema owner: %w", err)
		}
	}

	return c.GetSchema(ctx, opts.DatabaseName, opts.SchemaName)
}

// DropSchema drops a schema.
func (c *Client) DropSchema(ctx context.Context, databaseName, schemaName string) error {
	if err := c.UseDatabase(ctx, databaseName); err != nil {
		return err
	}

	query := fmt.Sprintf("DROP SCHEMA IF EXISTS [%s]", schemaName)
	_, err := c.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to drop schema: %w", err)
	}

	return nil
}
