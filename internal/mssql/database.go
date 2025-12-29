// Copyright (c) 2024 muecahit94
// SPDX-License-Identifier: MIT

package mssql

import (
	"context"
	"database/sql"
	"fmt"
)

// Database represents a SQL Server database.
type Database struct {
	ID   int
	Name string
}

// GetDatabase retrieves a database by name.
func (c *Client) GetDatabase(ctx context.Context, name string) (*Database, error) {
	query := `SELECT database_id, name FROM sys.databases WHERE name = @p1`
	row := c.QueryRowContext(ctx, query, name)

	var db Database
	err := row.Scan(&db.ID, &db.Name)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get database: %w", err)
	}

	return &db, nil
}

// GetDatabaseByID retrieves a database by ID.
func (c *Client) GetDatabaseByID(ctx context.Context, id int) (*Database, error) {
	query := `SELECT database_id, name FROM sys.databases WHERE database_id = @p1`
	row := c.QueryRowContext(ctx, query, id)

	var db Database
	err := row.Scan(&db.ID, &db.Name)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get database: %w", err)
	}

	return &db, nil
}

// ListDatabases retrieves all databases.
func (c *Client) ListDatabases(ctx context.Context) ([]Database, error) {
	query := `SELECT database_id, name FROM sys.databases ORDER BY name`
	rows, err := c.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list databases: %w", err)
	}
	defer rows.Close()

	var databases []Database
	for rows.Next() {
		var db Database
		if err := rows.Scan(&db.ID, &db.Name); err != nil {
			return nil, fmt.Errorf("failed to scan database: %w", err)
		}
		databases = append(databases, db)
	}

	return databases, rows.Err()
}

// CreateDatabase creates a new database.
func (c *Client) CreateDatabase(ctx context.Context, name string) (*Database, error) {
	// Database names cannot use parameterized queries
	query := fmt.Sprintf("CREATE DATABASE [%s]", name)
	_, err := c.ExecContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to create database: %w", err)
	}

	return c.GetDatabase(ctx, name)
}

// DropDatabase drops a database.
func (c *Client) DropDatabase(ctx context.Context, name string) error {
	// Set to single user mode to force close all connections
	alterQuery := fmt.Sprintf("ALTER DATABASE [%s] SET SINGLE_USER WITH ROLLBACK IMMEDIATE", name)
	_, _ = c.ExecContext(ctx, alterQuery) // Ignore error if database doesn't exist or is already in single user mode

	query := fmt.Sprintf("DROP DATABASE IF EXISTS [%s]", name)
	_, err := c.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to drop database: %w", err)
	}

	return nil
}
