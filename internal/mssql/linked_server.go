// Copyright (c) 2024 muecahit94
// SPDX-License-Identifier: MIT

package mssql

import (
	"context"
	"database/sql"
	"fmt"
)

// LinkedServer represents a linked server definition from sys.servers.
type LinkedServer struct {
	ServerID                int
	Name                    string
	Product                 string
	Provider                string
	DataSource              string
	Location                string
	ProviderString          string
	Catalog                 string
	ConnectTimeout          int
	QueryTimeout            int
	CollationName           string
	IsRemoteLoginEnabled    bool // RPC
	IsRPCOutEnabled         bool
	IsDataAccessEnabled     bool
	IsCollationCompatible   bool
	UsesRemoteCollation     bool
	LazySchemaValidation    bool
	RemoteProcTransPromoted bool
}

// LinkedServerLogin represents a login mapping from sys.linked_logins.
// An empty LocalLogin stands for the mapping that applies to all local logins.
type LinkedServerLogin struct {
	ServerName     string
	LocalLogin     string
	UsesSelf       bool
	RemoteUser     string
	ServerID       int
	LocalPrincipal int
}

const linkedServerSelect = `
	SELECT
		server_id,
		name,
		ISNULL(product, ''),
		ISNULL(provider, ''),
		ISNULL(data_source, ''),
		ISNULL(location, ''),
		ISNULL(provider_string, ''),
		ISNULL(catalog, ''),
		ISNULL(connect_timeout, 0),
		ISNULL(query_timeout, 0),
		ISNULL(collation_name, ''),
		is_remote_login_enabled,
		is_rpc_out_enabled,
		is_data_access_enabled,
		is_collation_compatible,
		uses_remote_collation,
		lazy_schema_validation,
		is_remote_proc_transaction_promotion_enabled
	FROM sys.servers
	WHERE is_linked = 1 AND `

func scanLinkedServer(row *sql.Row) (*LinkedServer, error) {
	var s LinkedServer
	err := row.Scan(
		&s.ServerID,
		&s.Name,
		&s.Product,
		&s.Provider,
		&s.DataSource,
		&s.Location,
		&s.ProviderString,
		&s.Catalog,
		&s.ConnectTimeout,
		&s.QueryTimeout,
		&s.CollationName,
		&s.IsRemoteLoginEnabled,
		&s.IsRPCOutEnabled,
		&s.IsDataAccessEnabled,
		&s.IsCollationCompatible,
		&s.UsesRemoteCollation,
		&s.LazySchemaValidation,
		&s.RemoteProcTransPromoted,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get linked server: %w", err)
	}
	return &s, nil
}

// GetLinkedServer retrieves a linked server by name. It returns nil when the server does not exist.
func (c *Client) GetLinkedServer(ctx context.Context, name string) (*LinkedServer, error) {
	return scanLinkedServer(c.QueryRowContext(ctx, linkedServerSelect+"name = @p1", name))
}

// GetLinkedServerByID retrieves a linked server by its server ID. It returns nil when the server does not exist.
func (c *Client) GetLinkedServerByID(ctx context.Context, id int) (*LinkedServer, error) {
	return scanLinkedServer(c.QueryRowContext(ctx, linkedServerSelect+"server_id = @p1", id))
}

// LinkedServerOptions are the sp_serveroption settings of a linked server.
type LinkedServerOptions struct {
	RPC                  bool
	RPCOut               bool
	DataAccess           bool
	CollationCompatible  bool
	UseRemoteCollation   bool
	CollationName        string
	ConnectTimeout       int
	QueryTimeout         int
	LazySchemaValidation bool
	RemoteProcTransPromo bool
}

// CreateLinkedServerOptions contains options for creating a linked server.
type CreateLinkedServerOptions struct {
	Name           string
	Product        string
	Provider       string
	DataSource     string
	Location       string
	ProviderString string
	Catalog        string
	Options        LinkedServerOptions
}

// nullIfEmpty maps an empty string to SQL NULL so optional sp_addlinkedserver arguments are left unset.
func nullIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func onOff(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// CreateLinkedServer creates a linked server and applies its options.
func (c *Client) CreateLinkedServer(ctx context.Context, opts CreateLinkedServerOptions) (*LinkedServer, error) {
	// sp_addlinkedserver and sp_serveroption take their arguments as bound parameters,
	// so no identifier or value is interpolated into the statement.
	_, err := c.ExecContext(ctx, `
		EXEC master.dbo.sp_addlinkedserver
			@server = @p1,
			@srvproduct = @p2,
			@provider = @p3,
			@datasrc = @p4,
			@location = @p5,
			@provstr = @p6,
			@catalog = @p7`,
		opts.Name,
		opts.Product,
		nullIfEmpty(opts.Provider),
		nullIfEmpty(opts.DataSource),
		nullIfEmpty(opts.Location),
		nullIfEmpty(opts.ProviderString),
		nullIfEmpty(opts.Catalog),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create linked server: %w", err)
	}

	if err := c.SetLinkedServerOptions(ctx, opts.Name, opts.Options); err != nil {
		// Do not leave a half-configured linked server behind.
		_ = c.DropLinkedServer(ctx, opts.Name)
		return nil, err
	}

	return c.GetLinkedServer(ctx, opts.Name)
}

// SetLinkedServerOptions applies all sp_serveroption settings.
func (c *Client) SetLinkedServerOptions(ctx context.Context, name string, o LinkedServerOptions) error {
	collation := interface{}(nil)
	if o.CollationName != "" {
		collation = o.CollationName
	}

	settings := []struct {
		option string
		value  interface{}
	}{
		{"rpc", onOff(o.RPC)},
		{"rpc out", onOff(o.RPCOut)},
		{"data access", onOff(o.DataAccess)},
		{"collation compatible", onOff(o.CollationCompatible)},
		{"use remote collation", onOff(o.UseRemoteCollation)},
		{"lazy schema validation", onOff(o.LazySchemaValidation)},
		{"remote proc transaction promotion", onOff(o.RemoteProcTransPromo)},
		{"connect timeout", fmt.Sprintf("%d", o.ConnectTimeout)},
		{"query timeout", fmt.Sprintf("%d", o.QueryTimeout)},
		{"collation name", collation},
	}

	for _, s := range settings {
		// collation name may only be set while use remote collation is false.
		if s.option == "collation name" && (o.UseRemoteCollation || o.CollationName == "") {
			continue
		}
		if _, err := c.ExecContext(ctx,
			"EXEC master.dbo.sp_serveroption @server = @p1, @optname = @p2, @optvalue = @p3",
			name, s.option, s.value,
		); err != nil {
			return fmt.Errorf("failed to set linked server option '%s': %w", s.option, err)
		}
	}
	return nil
}

// DropLinkedServer drops a linked server together with its login mappings.
func (c *Client) DropLinkedServer(ctx context.Context, name string) error {
	if _, err := c.ExecContext(ctx,
		"EXEC master.dbo.sp_dropserver @server = @p1, @droplogins = 'droplogins'", name,
	); err != nil {
		return fmt.Errorf("failed to drop linked server: %w", err)
	}
	return nil
}

// GetLinkedServerLogin retrieves a login mapping. An empty localLogin selects the mapping for all local logins.
// It returns nil when the mapping does not exist.
func (c *Client) GetLinkedServerLogin(ctx context.Context, serverName, localLogin string) (*LinkedServerLogin, error) {
	query := `
		SELECT
			s.server_id,
			s.name,
			ll.local_principal_id,
			ISNULL(p.name, ''),
			ll.uses_self_credential,
			ISNULL(ll.remote_name, '')
		FROM sys.linked_logins ll
		JOIN sys.servers s ON s.server_id = ll.server_id
		LEFT JOIN sys.server_principals p ON p.principal_id = ll.local_principal_id
		WHERE s.is_linked = 1 AND s.name = @p1
		  AND (
			(@p2 = '' AND ll.local_principal_id = 0)
			OR (@p2 <> '' AND p.name = @p2)
		  )`
	row := c.QueryRowContext(ctx, query, serverName, localLogin)

	var l LinkedServerLogin
	err := row.Scan(&l.ServerID, &l.ServerName, &l.LocalPrincipal, &l.LocalLogin, &l.UsesSelf, &l.RemoteUser)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get linked server login: %w", err)
	}
	return &l, nil
}

// SetLinkedServerLoginOptions contains options for mapping a local login to a remote login.
type SetLinkedServerLoginOptions struct {
	ServerName string
	LocalLogin string // empty maps all local logins
	UseSelf    bool
	RemoteUser string
	Password   string
}

// SetLinkedServerLogin creates or replaces a login mapping.
// sp_addlinkedsrvlogin overwrites an existing mapping for the same local login.
func (c *Client) SetLinkedServerLogin(ctx context.Context, opts SetLinkedServerLoginOptions) (*LinkedServerLogin, error) {
	var err error
	if opts.UseSelf {
		_, err = c.ExecContext(ctx, `
			EXEC master.dbo.sp_addlinkedsrvlogin
				@rmtsrvname = @p1,
				@useself = 'true',
				@locallogin = @p2`,
			opts.ServerName, nullIfEmpty(opts.LocalLogin),
		)
	} else {
		_, err = c.ExecContext(ctx, `
			EXEC master.dbo.sp_addlinkedsrvlogin
				@rmtsrvname = @p1,
				@useself = 'false',
				@locallogin = @p2,
				@rmtuser = @p3,
				@rmtpassword = @p4`,
			opts.ServerName, nullIfEmpty(opts.LocalLogin), opts.RemoteUser, opts.Password,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to set linked server login: %w", err)
	}

	return c.GetLinkedServerLogin(ctx, opts.ServerName, opts.LocalLogin)
}

// DropLinkedServerLogin removes a login mapping. An empty localLogin removes the mapping for all local logins.
func (c *Client) DropLinkedServerLogin(ctx context.Context, serverName, localLogin string) error {
	if _, err := c.ExecContext(ctx,
		"EXEC master.dbo.sp_droplinkedsrvlogin @rmtsrvname = @p1, @locallogin = @p2",
		serverName, nullIfEmpty(localLogin),
	); err != nil {
		return fmt.Errorf("failed to drop linked server login: %w", err)
	}
	return nil
}
