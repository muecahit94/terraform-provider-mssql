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

// Database represents a SQL Server database.
type Database struct {
	ID                 int
	Name               string
	Collation          string
	CompatibilityLevel int
	RecoveryModel      string
}

// RecoveryModels are the values SQL Server accepts for a database recovery model.
var RecoveryModels = []string{"FULL", "SIMPLE", "BULK_LOGGED"}

// collationPattern matches SQL Server collation names, e.g. SQL_Latin1_General_CP1_CI_AS.
// A collation cannot be passed as a parameter, so it is validated before it is
// interpolated into CREATE DATABASE.
var collationPattern = regexp.MustCompile(`^[A-Za-z0-9_]+$`)

// quoteName quotes a SQL Server identifier, doubling any closing bracket.
func quoteName(name string) string {
	return "[" + strings.ReplaceAll(name, "]", "]]") + "]"
}

const databaseSelect = `
	SELECT
		database_id,
		name,
		ISNULL(collation_name, ''),
		compatibility_level,
		recovery_model_desc
	FROM sys.databases`

func scanDatabase(s interface{ Scan(dest ...any) error }) (*Database, error) {
	var db Database
	if err := s.Scan(&db.ID, &db.Name, &db.Collation, &db.CompatibilityLevel, &db.RecoveryModel); err != nil {
		return nil, err
	}
	return &db, nil
}

// GetDatabase retrieves a database by name.
func (c *Client) GetDatabase(ctx context.Context, name string) (*Database, error) {
	db, err := scanDatabase(c.QueryRowContext(ctx, databaseSelect+` WHERE name = @p1`, name))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get database: %w", err)
	}

	return db, nil
}

// GetDatabaseByID retrieves a database by ID.
func (c *Client) GetDatabaseByID(ctx context.Context, id int) (*Database, error) {
	db, err := scanDatabase(c.QueryRowContext(ctx, databaseSelect+` WHERE database_id = @p1`, id))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get database: %w", err)
	}

	return db, nil
}

// ListDatabases retrieves all databases.
func (c *Client) ListDatabases(ctx context.Context) ([]Database, error) {
	rows, err := c.QueryContext(ctx, databaseSelect+` ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("failed to list databases: %w", err)
	}
	defer rows.Close()

	var databases []Database
	for rows.Next() {
		db, err := scanDatabase(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan database: %w", err)
		}
		databases = append(databases, *db)
	}

	return databases, rows.Err()
}

// CreateDatabaseOptions contains options for creating a database.
type CreateDatabaseOptions struct {
	Name string
	// Collation is the collation of the new database. Empty uses the server default.
	Collation string
	// CompatibilityLevel is applied after creation. Zero keeps the server default.
	CompatibilityLevel int
	// RecoveryModel is applied after creation. Empty keeps the server default.
	RecoveryModel string
}

// CreateDatabase creates a new database.
func (c *Client) CreateDatabase(ctx context.Context, opts CreateDatabaseOptions) (*Database, error) {
	// Database names and collations cannot use parameterized queries.
	query := "CREATE DATABASE " + quoteName(opts.Name)
	if opts.Collation != "" {
		if !collationPattern.MatchString(opts.Collation) {
			return nil, fmt.Errorf("invalid collation name %q", opts.Collation)
		}
		query += " COLLATE " + opts.Collation
	}
	if _, err := c.ExecContext(ctx, query); err != nil {
		return nil, fmt.Errorf("failed to create database: %w", err)
	}

	if opts.CompatibilityLevel != 0 {
		if err := c.SetDatabaseCompatibilityLevel(ctx, opts.Name, opts.CompatibilityLevel); err != nil {
			return nil, err
		}
	}
	if opts.RecoveryModel != "" {
		if err := c.SetDatabaseRecoveryModel(ctx, opts.Name, opts.RecoveryModel); err != nil {
			return nil, err
		}
	}

	return c.GetDatabase(ctx, opts.Name)
}

// SetDatabaseCompatibilityLevel changes the compatibility level of a database.
func (c *Client) SetDatabaseCompatibilityLevel(ctx context.Context, name string, level int) error {
	// The level is an integer, so formatting it with %d cannot inject SQL.
	query := fmt.Sprintf("ALTER DATABASE %s SET COMPATIBILITY_LEVEL = %d", quoteName(name), level)
	if _, err := c.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("failed to set compatibility level: %w", err)
	}
	return nil
}

// SetDatabaseRecoveryModel changes the recovery model of a database.
func (c *Client) SetDatabaseRecoveryModel(ctx context.Context, name, model string) error {
	valid := false
	for _, m := range RecoveryModels {
		if model == m {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("invalid recovery model %q: must be one of %s", model, strings.Join(RecoveryModels, ", "))
	}

	query := fmt.Sprintf("ALTER DATABASE %s SET RECOVERY %s", quoteName(name), model)
	if _, err := c.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("failed to set recovery model: %w", err)
	}
	return nil
}

// DropDatabase drops a database.
func (c *Client) DropDatabase(ctx context.Context, name string) error {
	// Set to single user mode to force close all connections
	alterQuery := fmt.Sprintf("ALTER DATABASE %s SET SINGLE_USER WITH ROLLBACK IMMEDIATE", quoteName(name))
	_, _ = c.ExecContext(ctx, alterQuery) // Ignore error if database doesn't exist or is already in single user mode

	query := fmt.Sprintf("DROP DATABASE IF EXISTS %s", quoteName(name))
	_, err := c.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to drop database: %w", err)
	}

	return nil
}
