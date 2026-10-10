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
	// Owner is the login that owns the database; empty when the owner login no longer exists.
	Owner string

	AutoClose             bool
	AutoShrink            bool
	PageVerify            string
	SnapshotIsolation     bool
	ReadCommittedSnapshot bool
	QueryStore            bool
	Trustworthy           bool
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
		recovery_model_desc,
		ISNULL(SUSER_SNAME(owner_sid), ''),
		is_auto_close_on,
		is_auto_shrink_on,
		page_verify_option_desc,
		CASE WHEN snapshot_isolation_state IN (1, 2) THEN 1 ELSE 0 END,
		is_read_committed_snapshot_on,
		is_query_store_on,
		is_trustworthy_on
	FROM sys.databases`

func scanDatabase(s interface{ Scan(dest ...any) error }) (*Database, error) {
	var db Database
	if err := s.Scan(&db.ID, &db.Name, &db.Collation, &db.CompatibilityLevel, &db.RecoveryModel, &db.Owner,
		&db.AutoClose, &db.AutoShrink, &db.PageVerify, &db.SnapshotIsolation, &db.ReadCommittedSnapshot, &db.QueryStore, &db.Trustworthy); err != nil {
		return nil, err
	}
	return &db, nil
}

// collationInContextQuery reads the collation from inside the database. sys.databases returns
// NULL for it while a database with AUTO_CLOSE ON is closed (nobody is connected), and
// DATABASEPROPERTYEX does too; running in the context of the database opens it for the call.
const collationInContextQuery = `SELECT CONVERT(nvarchar(128), DATABASEPROPERTYEX(DB_NAME(), N'Collation'))`

// fillCollation completes the collation of a database that sys.databases reported as NULL.
// A database that cannot be opened (offline, restoring) keeps an empty collation.
func (c *Client) fillCollation(ctx context.Context, db *Database) {
	if db == nil || db.Collation != "" {
		return
	}
	var collation sql.NullString
	if err := c.QueryRowContext(ctx, "EXEC "+quoteName(db.Name)+".sys.sp_executesql @p1", collationInContextQuery).Scan(&collation); err == nil && collation.Valid {
		db.Collation = collation.String
	}
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

	c.fillCollation(ctx, db)
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

	c.fillCollation(ctx, db)
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
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()

	for i := range databases {
		c.fillCollation(ctx, &databases[i])
	}
	return databases, nil
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
	// Owner is the login that becomes the owner after creation. Empty keeps the creator.
	Owner string
	// Settings are applied after creation; unset ones keep the server defaults.
	Settings DatabaseSettings
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

	if opts.Owner != "" {
		if err := c.SetDatabaseOwner(ctx, opts.Name, opts.Owner); err != nil {
			return nil, err
		}
	}
	if err := c.ApplyDatabaseSettings(ctx, opts.Name, opts.Settings); err != nil {
		return nil, err
	}

	return c.GetDatabase(ctx, opts.Name)
}

// alterAuthorizationStatement builds the statement that changes the owner of a database.
func alterAuthorizationStatement(database, login string) string {
	return "ALTER AUTHORIZATION ON DATABASE::" + quoteName(database) + " TO " + quoteName(login)
}

// SetDatabaseOwner makes a login the owner of a database.
func (c *Client) SetDatabaseOwner(ctx context.Context, name, login string) error {
	if _, err := c.ExecContext(ctx, alterAuthorizationStatement(name, login)); err != nil {
		return fmt.Errorf("failed to set database owner: %w", err)
	}
	return nil
}

// PageVerifyOptions are the values SQL Server accepts for PAGE_VERIFY.
var PageVerifyOptions = []string{"CHECKSUM", "TORN_PAGE_DETECTION", "NONE"}

// DatabaseSettings are the ALTER DATABASE ... SET options that can be changed in place.
// A nil field is left as it is.
type DatabaseSettings struct {
	AutoClose             *bool
	AutoShrink            *bool
	PageVerify            *string
	SnapshotIsolation     *bool
	ReadCommittedSnapshot *bool
	QueryStore            *bool
	Trustworthy           *bool
}

func onOffKeyword(b bool) string {
	if b {
		return "ON"
	}
	return "OFF"
}

// Statements returns the ALTER DATABASE statements that apply the settings.
// An invalid PAGE_VERIFY value is reported before any statement is built, because the
// value is part of the statement text.
func (s DatabaseSettings) Statements(database string) ([]string, error) {
	db := quoteName(database)
	var out []string
	if s.AutoClose != nil {
		out = append(out, "ALTER DATABASE "+db+" SET AUTO_CLOSE "+onOffKeyword(*s.AutoClose))
	}
	if s.AutoShrink != nil {
		out = append(out, "ALTER DATABASE "+db+" SET AUTO_SHRINK "+onOffKeyword(*s.AutoShrink))
	}
	if s.PageVerify != nil {
		valid := false
		for _, v := range PageVerifyOptions {
			if *s.PageVerify == v {
				valid = true
			}
		}
		if !valid {
			return nil, fmt.Errorf("invalid page verify option %q: must be one of %s", *s.PageVerify, strings.Join(PageVerifyOptions, ", "))
		}
		out = append(out, "ALTER DATABASE "+db+" SET PAGE_VERIFY "+*s.PageVerify)
	}
	if s.SnapshotIsolation != nil {
		out = append(out, "ALTER DATABASE "+db+" SET ALLOW_SNAPSHOT_ISOLATION "+onOffKeyword(*s.SnapshotIsolation))
	}
	if s.ReadCommittedSnapshot != nil {
		// Needs exclusive access to the database; fail at once instead of waiting for the connections to end.
		out = append(out, "ALTER DATABASE "+db+" SET READ_COMMITTED_SNAPSHOT "+onOffKeyword(*s.ReadCommittedSnapshot)+" WITH NO_WAIT")
	}
	if s.QueryStore != nil {
		out = append(out, "ALTER DATABASE "+db+" SET QUERY_STORE = "+onOffKeyword(*s.QueryStore))
	}
	if s.Trustworthy != nil {
		out = append(out, "ALTER DATABASE "+db+" SET TRUSTWORTHY "+onOffKeyword(*s.Trustworthy))
	}
	return out, nil
}

// ApplyDatabaseSettings changes the settings of a database.
func (c *Client) ApplyDatabaseSettings(ctx context.Context, name string, settings DatabaseSettings) error {
	statements, err := settings.Statements(name)
	if err != nil {
		return err
	}
	for _, statement := range statements {
		if _, err := c.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("failed to change database settings (%s): %w", statement, err)
		}
	}
	return nil
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
