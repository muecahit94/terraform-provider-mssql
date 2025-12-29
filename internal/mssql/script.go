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
	if databaseName != "" {
		if err := c.UseDatabase(ctx, databaseName); err != nil {
			return nil, err
		}
	}

	rows, err := c.QueryContext(ctx, script)
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
	if databaseName != "" {
		if err := c.UseDatabase(ctx, databaseName); err != nil {
			return err
		}
	}

	_, err := c.ExecContext(ctx, script)
	if err != nil {
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
