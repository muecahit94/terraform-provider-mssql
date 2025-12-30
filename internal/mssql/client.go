// Copyright (c) 2024 muecahit94
// SPDX-License-Identifier: MIT

// Package mssql provides a client for interacting with Microsoft SQL Server.
package mssql

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"strconv"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	mssqldb "github.com/microsoft/go-mssqldb"
)

// Client represents a connection to a SQL Server instance.
type Client struct {
	db       *sql.DB
	hostname string
	port     int
}

// Config holds the configuration for connecting to SQL Server.
type Config struct {
	Hostname string
	Port     int

	// SQL Authentication
	SQLAuth *SQLAuthConfig

	// Azure AD Authentication
	AzureAuth *AzureAuthConfig
}

// SQLAuthConfig holds SQL authentication credentials.
type SQLAuthConfig struct {
	Username string
	Password string
}

// AzureAuthConfig holds Azure AD authentication configuration.
type AzureAuthConfig struct {
	ClientID     string
	ClientSecret string
	TenantID     string
}

// NewClient creates a new SQL Server client with the given configuration.
func NewClient(ctx context.Context, cfg *Config) (*Client, error) {
	if cfg.Hostname == "" {
		cfg.Hostname = os.Getenv("MSSQL_HOSTNAME")
	}
	if cfg.Port == 0 {
		if portStr := os.Getenv("MSSQL_PORT"); portStr != "" {
			port, err := strconv.Atoi(portStr)
			if err == nil {
				cfg.Port = port
			}
		}
		if cfg.Port == 0 {
			cfg.Port = 1433
		}
	}

	var db *sql.DB
	var err error

	if cfg.AzureAuth != nil {
		db, err = connectWithAzureAuth(ctx, cfg)
	} else if cfg.SQLAuth != nil {
		db, err = connectWithSQLAuth(cfg)
	} else {
		return nil, fmt.Errorf("no authentication method configured")
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to SQL Server: %w", err)
	}

	// Verify connection
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping SQL Server: %w", err)
	}

	return &Client{
		db:       db,
		hostname: cfg.Hostname,
		port:     cfg.Port,
	}, nil
}

// connectWithSQLAuth establishes a connection using SQL authentication.
func connectWithSQLAuth(cfg *Config) (*sql.DB, error) {
	query := url.Values{}
	query.Add("app name", "terraform-provider-mssql")

	u := &url.URL{
		Scheme:   "sqlserver",
		User:     url.UserPassword(cfg.SQLAuth.Username, cfg.SQLAuth.Password),
		Host:     fmt.Sprintf("%s:%d", cfg.Hostname, cfg.Port),
		RawQuery: query.Encode(),
	}

	db, err := sql.Open("sqlserver", u.String())
	if err != nil {
		return nil, err
	}

	return db, nil
}

// connectWithAzureAuth establishes a connection using Azure AD authentication.
func connectWithAzureAuth(ctx context.Context, cfg *Config) (*sql.DB, error) {
	var cred azcore.TokenCredential
	var err error

	// Check for environment variable override
	clientID := cfg.AzureAuth.ClientID
	clientSecret := cfg.AzureAuth.ClientSecret
	tenantID := cfg.AzureAuth.TenantID

	if clientID == "" {
		clientID = os.Getenv("ARM_CLIENT_ID")
	}
	if clientSecret == "" {
		clientSecret = os.Getenv("ARM_CLIENT_SECRET")
	}
	if tenantID == "" {
		tenantID = os.Getenv("ARM_TENANT_ID")
	}

	if clientID != "" && clientSecret != "" && tenantID != "" {
		// Use Service Principal authentication
		cred, err = azidentity.NewClientSecretCredential(tenantID, clientID, clientSecret, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create client secret credential: %w", err)
		}
	} else {
		// Use default Azure credential chain
		cred, err = azidentity.NewDefaultAzureCredential(nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create default Azure credential: %w", err)
		}
	}

	// Get token for Azure SQL
	token, err := cred.GetToken(ctx, policy.TokenRequestOptions{
		Scopes: []string{"https://database.windows.net/.default"},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get Azure AD token: %w", err)
	}

	connector, err := mssqldb.NewAccessTokenConnector(
		fmt.Sprintf("sqlserver://%s:%d?database=master&app+name=terraform-provider-mssql", cfg.Hostname, cfg.Port),
		func() (string, error) {
			return token.Token, nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create access token connector: %w", err)
	}

	db := sql.OpenDB(connector)
	return db, nil
}

// Close closes the database connection.
func (c *Client) Close() error {
	if c.db != nil {
		return c.db.Close()
	}
	return nil
}

// DB returns the underlying database connection for advanced queries.
func (c *Client) DB() *sql.DB {
	return c.db
}

// Hostname returns the connected server hostname.
func (c *Client) Hostname() string {
	return c.hostname
}

// Port returns the connected server port.
func (c *Client) Port() int {
	return c.port
}

// ExecContext executes a query without returning any rows.
func (c *Client) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return c.db.ExecContext(ctx, query, args...)
}

// QueryContext executes a query that returns rows.
func (c *Client) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return c.db.QueryContext(ctx, query, args...)
}

// QueryRowContext executes a query that is expected to return at most one row.
func (c *Client) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return c.db.QueryRowContext(ctx, query, args...)
}

// UseDatabase switches the connection to use the specified database.
func (c *Client) UseDatabase(ctx context.Context, databaseName string) error {
	_, err := c.db.ExecContext(ctx, fmt.Sprintf("USE [%s]", databaseName))
	return err
}

// ExecInDatabaseContext executes a query in the context of a specific database.
// This uses a dedicated connection to ensure the USE statement persists for the query.
func (c *Client) ExecInDatabaseContext(ctx context.Context, databaseName, query string) error {
	// Get a dedicated connection from the pool
	conn, err := c.db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("failed to get database connection: %w", err)
	}
	defer conn.Close()

	// Switch to the target database
	if _, err := conn.ExecContext(ctx, fmt.Sprintf("USE [%s]", databaseName)); err != nil {
		return fmt.Errorf("failed to switch database context: %w", err)
	}

	// Execute the query in the correct context
	if _, err := conn.ExecContext(ctx, query); err != nil {
		return err
	}

	return nil
}

// QueryRowInDatabaseContext executes a query in the context of a specific database and returns a row.
// This uses a dedicated connection to ensure the USE statement persists for the query.
func (c *Client) QueryRowInDatabaseContext(ctx context.Context, databaseName, query string, args ...interface{}) (*sql.Row, error) {
	// Get a dedicated connection from the pool
	conn, err := c.db.Conn(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get database connection: %w", err)
	}
	// Note: We don't close the connection here because the Row needs it for scanning
	// The connection will be returned to the pool when the row is scanned or closed

	// Switch to the target database
	if _, err := conn.ExecContext(ctx, fmt.Sprintf("USE [%s]", databaseName)); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to switch database context: %w", err)
	}

	// Execute the query in the correct context
	row := conn.QueryRowContext(ctx, query, args...)
	return row, nil
}
