// Copyright (c) 2024 muecahit94
// SPDX-License-Identifier: MIT

// Package provider implements the MSSQL Terraform provider.
package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
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
		},
		Blocks: map[string]schema.Block{
			"sql_auth": schema.SingleNestedBlock{
				Description: "SQL authentication credentials.",
				Attributes: map[string]schema.Attribute{
					"username": schema.StringAttribute{
						Description: "Username for SQL authentication.",
						Required:    true,
					},
					"password": schema.StringAttribute{
						Description: "Password for SQL authentication.",
						Required:    true,
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
	client, err := mssql.NewClient(ctx, cfg)
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

	// Make the client available during DataSource and Resource type Configure methods.
	resp.DataSourceData = client
	resp.ResourceData = client
}

// Resources defines the resources implemented in the provider.
func (p *MSSQLProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewDatabaseResource,
		NewSQLLoginResource,
		NewSQLUserResource,
		NewDatabaseRoleResource,
		NewDatabaseRoleMemberResource,
		NewDatabasePermissionResource,
		NewSchemaResource,
		NewSchemaPermissionResource,
		NewServerRoleResource,
		NewServerRoleMemberResource,
		NewServerPermissionResource,
		NewScriptResource,
		NewAzureADUserResource,
		NewAzureADServicePrincipalResource,
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
