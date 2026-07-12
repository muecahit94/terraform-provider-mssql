// Copyright (c) 2024 muecahit94
// SPDX-License-Identifier: MIT

package mssql

import (
	"context"
	"crypto/sha256"
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

// ExecuteScript executes a SQL script and returns the results as a map.
func (c *Client) ExecuteScript(ctx context.Context, databaseName, script string) (map[string]string, error) {
	// Use a dedicated connection so that the USE statement and the script
	// execution are guaranteed to run on the same connection. This prevents
	// the database context from being lost when the pool hands out a
	// different connection (e.g. after a server-side connection reset caused
	// by ALTER DATABASE … WITH ROLLBACK IMMEDIATE).
	conn, err := c.db.Conn(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get database connection: %w", err)
	}
	defer conn.Close()

	if databaseName != "" {
		if _, err := conn.ExecContext(ctx, fmt.Sprintf("USE [%s]", databaseName)); err != nil {
			return nil, fmt.Errorf("failed to switch database context: %w", err)
		}
	}

	rows, err := conn.QueryContext(ctx, script)
	if err != nil {
		return nil, fmt.Errorf("failed to execute script: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	result := make(map[string]string)

	if rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		for i, col := range columns {
			if values[i] != nil {
				switch v := values[i].(type) {
				case []byte:
					result[col] = string(v)
				default:
					result[col] = fmt.Sprintf("%v", v)
				}
			} else {
				result[col] = ""
			}
		}
	}

	return result, rows.Err()
}

// ExecuteScriptNoResult executes a SQL script without returning results.
func (c *Client) ExecuteScriptNoResult(ctx context.Context, databaseName, script string) error {
	// Use a dedicated connection so that the USE statement and the script
	// execution are guaranteed to run on the same connection. This prevents
	// the database context from being lost when the pool hands out a
	// different connection (e.g. after a server-side connection reset caused
	// by ALTER DATABASE … WITH ROLLBACK IMMEDIATE).
	conn, err := c.db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("failed to get database connection: %w", err)
	}
	defer conn.Close()

	if databaseName != "" {
		if _, err := conn.ExecContext(ctx, fmt.Sprintf("USE [%s]", databaseName)); err != nil {
			return fmt.Errorf("failed to switch database context: %w", err)
		}
	}

	if _, err := conn.ExecContext(ctx, script); err != nil {
		return fmt.Errorf("failed to execute script: %w", err)
	}

	return nil
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
	if databaseName != "" {
		if err := c.UseDatabase(ctx, databaseName); err != nil {
			return nil, err
		}
	}

	rows, err := c.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	result := &QueryResult{
		Columns: columns,
		Rows:    []map[string]string{},
	}

	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		row := make(map[string]string)
		for i, col := range columns {
			if values[i] != nil {
				switch v := values[i].(type) {
				case []byte:
					row[col] = string(v)
				default:
					row[col] = fmt.Sprintf("%v", v)
				}
			} else {
				row[col] = ""
			}
		}
		result.Rows = append(result.Rows, row)
	}

	return result, rows.Err()
}
