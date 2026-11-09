// Copyright (c) 2024 muecahit94
// SPDX-License-Identifier: MIT

// Package provider implements the MSSQL Terraform provider.
package provider

import (
	"context"
	"fmt"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/muecahit94/terraform-provider-mssql/internal/mssql"
)

// Ensure MSSQLProvider satisfies various provider interfaces.
var _ provider.Provider = &MSSQLProvider{}

// MSSQLProvider defines the provider implementation.
type MSSQLProvider struct {
	version string
}

// MSSQLProviderModel describes the provider data model.
type MSSQLProviderModel struct {
	Hostname  types.String    `tfsdk:"hostname"`
	Port      types.Int64     `tfsdk:"port"`
	SQLAuth   *SQLAuthModel   `tfsdk:"sql_auth"`
	AzureAuth *AzureAuthModel `tfsdk:"azure_auth"`
	Servers   types.Map       `tfsdk:"servers"`
}

type registeredServerModel struct {
	Hostname types.String `tfsdk:"hostname"`
	Port     types.Int64  `tfsdk:"port"`
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
}

func registeredServerConfigs(ctx context.Context, servers types.Map) (map[string]*mssql.Config, diag.Diagnostics) {
	var diags diag.Diagnostics
	if servers.IsNull() {
		return nil, diags
	}
	if servers.IsUnknown() {
		diags.AddAttributeError(path.Root("servers"), "Unknown server registry", "The server registry must be known when the provider is configured.")
		return nil, diags
	}

	var entries map[string]registeredServerModel
	diags.Append(servers.ElementsAs(ctx, &entries, false)...)
	if diags.HasError() {
		return nil, diags
	}

	configs := make(map[string]*mssql.Config, len(entries))
	endpoints := make(map[string]string, len(entries))
	for key, entry := range entries {
		if key == "" || entry.Hostname.IsUnknown() || entry.Hostname.IsNull() || entry.Hostname.ValueString() == "" ||
			entry.Username.IsUnknown() || entry.Username.IsNull() || entry.Username.ValueString() == "" ||
			entry.Password.IsUnknown() || entry.Password.IsNull() || entry.Password.ValueString() == "" ||
			entry.Port.IsUnknown() || (!entry.Port.IsNull() && (entry.Port.ValueInt64() < 1 || entry.Port.ValueInt64() > 65535)) {
			diags.AddAttributeError(path.Root("servers").AtMapKey(key), "Invalid server registry entry",
				"Each server needs a non-empty key, hostname, username, and password, and a known port between 1 and 65535.")
			continue
		}
		port := 1433
		if !entry.Port.IsNull() {
			port = int(entry.Port.ValueInt64())
		}
		endpoint := mssql.ServerEndpoint(entry.Hostname.ValueString(), port)
		if previous, ok := endpoints[endpoint]; ok {
			diags.AddAttributeError(path.Root("servers").AtMapKey(key), "Duplicate server endpoint",
				fmt.Sprintf("Server keys %q and %q refer to the same endpoint. Each endpoint must be unique.", previous, key))
			continue
		}
		endpoints[endpoint] = key
		configs[key] = &mssql.Config{
			Hostname: entry.Hostname.ValueString(),
			Port:     port,
			SQLAuth: &mssql.SQLAuthConfig{
				Username: entry.Username.ValueString(),
				Password: entry.Password.ValueString(),
			},
		}
	}
	return configs, diags
}

// SQLAuthModel describes SQL authentication configuration.
type SQLAuthModel struct {
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
}

// AzureAuthModel describes Azure AD authentication configuration.
type AzureAuthModel struct {
	ClientID     types.String `tfsdk:"client_id"`
	ClientSecret types.String `tfsdk:"client_secret"`
	TenantID     types.String `tfsdk:"tenant_id"`
}

// New creates a new provider instance.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &MSSQLProvider{
			version: version,
		}
	}
}

// Metadata returns the provider type name.
func (p *MSSQLProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "mssql"
	resp.Version = p.version
}

// Schema defines the provider-level schema for configuration data.
func (p *MSSQLProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The MSSQL provider allows you to manage Microsoft SQL Server and Azure SQL resources.",
		Attributes: map[string]schema.Attribute{
			"hostname": schema.StringAttribute{
				Description: "FQDN or IP address of the SQL endpoint. Can also be set using MSSQL_HOSTNAME environment variable.",
				Optional:    true,
			},
			"port": schema.Int64Attribute{
				Description: "TCP port of SQL endpoint. Defaults to 1433. Can also be set using MSSQL_PORT environment variable.",
				Optional:    true,
			},
			"servers": schema.MapNestedAttribute{
				Description: "Named SQL Server connections for mssql_sql_login resources. Credentials are taken from the current provider configuration.",
				Optional:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"hostname": schema.StringAttribute{Required: true, Description: "SQL Server hostname."},
						"port":     schema.Int64Attribute{Optional: true, Description: "SQL Server port. Defaults to 1433."},
						"username": schema.StringAttribute{Required: true, Description: "SQL authentication username."},
						"password": schema.StringAttribute{Required: true, Sensitive: true, Description: "SQL authentication password."},
					},
				},
			},
		},
		Blocks: map[string]schema.Block{
			"sql_auth": schema.SingleNestedBlock{
				Description: "SQL authentication credentials. Either sql_auth or azure_auth must be provided.",
				Attributes: map[string]schema.Attribute{
					"username": schema.StringAttribute{
						Description: "Username for SQL authentication.",
						Optional:    true,
					},
					"password": schema.StringAttribute{
						Description: "Password for SQL authentication.",
						Optional:    true,
						Sensitive:   true,
					},
				},
			},
			"azure_auth": schema.SingleNestedBlock{
				Description: "Azure AD authentication configuration. When set to empty block, uses default Azure credential chain.",
				Attributes: map[string]schema.Attribute{
					"client_id": schema.StringAttribute{
						Description: "Service Principal client (application) ID. When omitted, default credential chain will be used.",
						Optional:    true,
					},
					"client_secret": schema.StringAttribute{
						Description: "Service Principal secret. When omitted, default credential chain will be used.",
						Optional:    true,
						Sensitive:   true,
					},
					"tenant_id": schema.StringAttribute{
						Description: "Azure AD tenant ID. Required only if Azure SQL Server's tenant is different than Service Principal's.",
						Optional:    true,
					},
				},
			},
		},
	}
}

// Configure prepares a SQL Server client for data sources and resources.
func (p *MSSQLProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	tflog.Info(ctx, "Configuring MSSQL provider")

	var config MSSQLProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)

	if resp.Diagnostics.HasError() {
		return
	}

	servers, serverDiags := registeredServerConfigs(ctx, config.Servers)
	resp.Diagnostics.Append(serverDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build client configuration
	cfg := &mssql.Config{
		Hostname: config.Hostname.ValueString(),
		Port:     int(config.Port.ValueInt64()),
	}

	// Configure authentication
	if config.SQLAuth != nil {
		cfg.SQLAuth = &mssql.SQLAuthConfig{
			Username: config.SQLAuth.Username.ValueString(),
			Password: config.SQLAuth.Password.ValueString(),
		}
	} else if config.AzureAuth != nil {
		cfg.AzureAuth = &mssql.AzureAuthConfig{
			ClientID:     config.AzureAuth.ClientID.ValueString(),
			ClientSecret: config.AzureAuth.ClientSecret.ValueString(),
			TenantID:     config.AzureAuth.TenantID.ValueString(),
		}
	}

	// Create client
	var client *mssql.Client
	var err error

	hasHost := cfg.Hostname != "" || os.Getenv("MSSQL_HOSTNAME") != ""
	hasAuth := cfg.SQLAuth != nil || cfg.AzureAuth != nil || os.Getenv("ARM_CLIENT_ID") != ""

	if !hasHost && !hasAuth {
		client = mssql.NewMultiServerClient()
		tflog.Info(ctx, "MSSQL provider configured in multi-server mode (no default server)")
	} else {
		client, err = mssql.NewClient(ctx, cfg)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Create MSSQL Client",
				"An unexpected error occurred when creating the MSSQL client. "+
					"If the error is not clear, please contact the provider developers.\n\n"+
					"Error: "+err.Error(),
			)
			return
		}

		tflog.Info(ctx, "MSSQL provider configured successfully", map[string]interface{}{
			"hostname": cfg.Hostname,
			"port":     cfg.Port,
		})
	}

	client.SetServers(servers)

	// Make the client available during DataSource and Resource type Configure methods.
	resp.DataSourceData = client
	resp.ResourceData = client
}

// Resources defines the resources implemented in the provider.
func (p *MSSQLProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewDatabaseResource,
		NewSQLLoginResource,
		NewWindowsLoginResource,
		NewServerConfigurationResource,
		NewAgentJobResource,
		NewSQLUserResource,
		NewDatabaseRoleResource,
		NewDatabaseRoleMemberResource,
		NewDatabasePermissionResource,
		NewDatabaseObjectPermissionResource,
		NewSchemaResource,
		NewSchemaPermissionResource,
		NewServerRoleResource,
		NewServerRoleMemberResource,
		NewServerPermissionResource,
		NewScriptResource,
		NewAzureADUserResource,
		NewAzureADServicePrincipalResource,
		NewLinkedServerResource,
		NewLinkedServerLoginResource,
	}
}

// DataSources defines the data sources implemented in the provider.
func (p *MSSQLProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewDatabaseDataSource,
		NewDatabasesDataSource,
		NewSQLLoginDataSource,
		NewSQLLoginsDataSource,
		NewSQLUserDataSource,
		NewSQLUsersDataSource,
		NewDatabaseRoleDataSource,
		NewDatabaseRolesDataSource,
		NewDatabasePermissionsDataSource,
		NewSchemaDataSource,
		NewSchemasDataSource,
		NewSchemaPermissionsDataSource,
		NewServerRoleDataSource,
		NewServerRolesDataSource,
		NewServerPermissionsDataSource,
		NewAzureADUserDataSource,
		NewAzureADServicePrincipalDataSource,
		NewQueryDataSource,
	}
}
