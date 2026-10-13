// Copyright (c) 2024 muecahit94
// SPDX-License-Identifier: MIT

package mssql

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
)

// Script represents a SQL script execution.
type Script struct {
	ID           string
	DatabaseName string
	CreateScript string
	ReadScript   string
	UpdateScript string
	DeleteScript string
	State        map[string]string
}

// ScriptState stores the state read from a script execution.
type ScriptState struct {
	ID    string
	State map[string]string
}

// rowValues converts the current row to strings; NULL becomes an empty string.
func rowValues(rows *sql.Rows, columns []string) (map[string]string, error) {
	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}
	if err := rows.Scan(valuePtrs...); err != nil {
		return nil, fmt.Errorf("failed to scan row: %w", err)
	}

	row := make(map[string]string, len(columns))
	for i, col := range columns {
		switch v := values[i].(type) {
		case nil:
			row[col] = ""
		case []byte:
			row[col] = string(v)
		default:
			row[col] = fmt.Sprintf("%v", v)
		}
	}
	return row, nil
}

// ExecuteScript executes a SQL script and returns the first row of its result as a map.
// The script runs on one connection, so a database context set inside it lasts for the whole script.
func (c *Client) ExecuteScript(ctx context.Context, databaseName, script string) (map[string]string, error) {
	result := make(map[string]string)
	err := c.withDatabase(ctx, databaseName, func(db sqlRunner) error {
		rows, err := db.QueryContext(ctx, script)
		if err != nil {
			return fmt.Errorf("failed to execute script: %w", err)
		}
		defer rows.Close()

		columns, err := rows.Columns()
		if err != nil {
			return fmt.Errorf("failed to get columns: %w", err)
		}
		if rows.Next() {
			if result, err = rowValues(rows, columns); err != nil {
				return err
			}
		}
		return rows.Err()
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// ExecuteScriptNoResult executes a SQL script without returning results.
func (c *Client) ExecuteScriptNoResult(ctx context.Context, databaseName, script string) error {
	return c.withDatabase(ctx, databaseName, func(db sqlRunner) error {
		if _, err := db.ExecContext(ctx, script); err != nil {
			return fmt.Errorf("failed to execute script: %w", err)
		}
		return nil
	})
}

// GenerateScriptID generates a unique ID for a script based on its content.
func GenerateScriptID(createScript, databaseName string) string {
	hash := sha256.Sum256([]byte(createScript + databaseName))
	return hex.EncodeToString(hash[:16])
}

// Query represents a custom query result.
type QueryResult struct {
	Columns []string
	Rows    []map[string]string
}

// ExecuteQuery executes a query and returns all results.
func (c *Client) ExecuteQuery(ctx context.Context, databaseName, query string) (*QueryResult, error) {
	result := &QueryResult{Rows: []map[string]string{}}
	err := c.withDatabase(ctx, databaseName, func(db sqlRunner) error {
		rows, err := db.QueryContext(ctx, query)
		if err != nil {
			return fmt.Errorf("failed to execute query: %w", err)
		}
		defer rows.Close()

		if result.Columns, err = rows.Columns(); err != nil {
			return fmt.Errorf("failed to get columns: %w", err)
		}
		for rows.Next() {
			row, err := rowValues(rows, result.Columns)
			if err != nil {
				return err
			}
			result.Rows = append(result.Rows, row)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}
