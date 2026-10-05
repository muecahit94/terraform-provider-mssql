// Copyright (c) 2024 muecahit94
// SPDX-License-Identifier: MIT

package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/muecahit94/terraform-provider-mssql/internal/mssql"
)

var _ datasource.DataSource = &SQLLoginDataSource{}

func NewSQLLoginDataSource() datasource.DataSource {
	return &SQLLoginDataSource{}
}

type SQLLoginDataSource struct {
	client *mssql.Client
}

type SQLLoginDataSourceModel struct {
	ID                     types.String `tfsdk:"id"`
	Name                   types.String `tfsdk:"name"`
	LoginName              types.String `tfsdk:"login_name"`
	SID                    types.String `tfsdk:"sid"`
	DefaultDatabase        types.String `tfsdk:"default_database"`
	DefaultLanguage        types.String `tfsdk:"default_language"`
	CheckExpirationEnabled types.Bool   `tfsdk:"check_expiration_enabled"`
	CheckPolicyEnabled     types.Bool   `tfsdk:"check_policy_enabled"`
	IsDisabled             types.Bool   `tfsdk:"is_disabled"`
	Server                 *ServerModel `tfsdk:"server"`
}

func (d *SQLLoginDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sql_login"
}

func (d *SQLLoginDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Use this data source to get information about a SQL Server login.",
		Attributes: map[string]schema.Attribute{
			"id":                       schema.StringAttribute{Computed: true},
			"name":                     schema.StringAttribute{Description: "The name of the SQL login.", Optional: true, Computed: true},
			"login_name":               schema.StringAttribute{Description: "Alias for `name`. The name of the SQL login.", Optional: true},
			"sid":                      schema.StringAttribute{Description: "The SID (Security Identifier) of the SQL login in hexadecimal format.", Computed: true},
			"default_database":         schema.StringAttribute{Computed: true},
			"default_language":         schema.StringAttribute{Computed: true},
			"check_expiration_enabled": schema.BoolAttribute{Computed: true},
			"check_policy_enabled":     schema.BoolAttribute{Computed: true},
			"is_disabled":              schema.BoolAttribute{Computed: true},
		},
		Blocks: map[string]schema.Block{
			"server": schema.SingleNestedBlock{
				Description: "SQL Server instance configuration. If omitted, the default provider connection settings are used.",
				Attributes: map[string]schema.Attribute{
					"hostname": schema.StringAttribute{
						Description: "FQDN or IP address of the SQL endpoint.",
						Optional:    true,
					},
					"host": schema.StringAttribute{
						Description: "Alias for `hostname`. FQDN or IP address of the SQL endpoint.",
						Optional:    true,
					},
					"port": schema.Int64Attribute{
						Description: "TCP port of SQL endpoint. Defaults to 1433.",
						Optional:    true,
					},
				},
				Blocks: map[string]schema.Block{
					"sql_auth": schema.SingleNestedBlock{
						Description: "SQL authentication credentials. Either sql_auth, login, or azure_auth must be provided.",
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
					"login": schema.SingleNestedBlock{
						Description: "Alias for `sql_auth`. SQL authentication credentials.",
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
						Description: "Azure AD authentication configuration.",
						Attributes: map[string]schema.Attribute{
							"client_id": schema.StringAttribute{
								Description: "Service Principal client (application) ID.",
								Optional:    true,
							},
							"client_secret": schema.StringAttribute{
								Description: "Service Principal secret.",
								Optional:    true,
								Sensitive:   true,
							},
							"tenant_id": schema.StringAttribute{
								Description: "Azure AD tenant ID.",
								Optional:    true,
							},
						},
					},
				},
			},
		},
	}
}

func (d *SQLLoginDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*mssql.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Data Source Configure Type", fmt.Sprintf("Expected *mssql.Client, got: %T.", req.ProviderData))
		return
	}
	d.client = client
}

func (d *SQLLoginDataSource) getClient(ctx context.Context, server *ServerModel) (*mssql.Client, diag.Diagnostics) {
	var diags diag.Diagnostics

	if server != nil && (!server.Hostname.IsNull() || !server.Host.IsNull()) {
		cfg, sDiags := serverToConfig(server)
		diags.Append(sDiags...)
		if diags.HasError() {
			return nil, diags
		}

		if d.client == nil {
			diags.AddError(
				"MSSQL Provider Not Configured",
				"The MSSQL provider has not been initialized.",
			)
			return nil, diags
		}

		client, err := d.client.GetClientForServer(ctx, cfg)
		if err != nil {
			diags.AddError(
				"Failed to Connect to SQL Server",
				fmt.Sprintf("Failed to establish connection to server %s:%d: %s", cfg.Hostname, cfg.Port, err.Error()),
			)
			return nil, diags
		}
		return client, diags
	}

	if d.client == nil || d.client.DB() == nil {
		diags.AddError(
			"Default MSSQL Connection Not Configured",
			"No connection configuration was found. You must either configure connection settings in the provider block (hostname, auth) or provide a 'server' block on the data source.",
		)
		return nil, diags
	}

	return d.client, diags
}

func validateDataSourceLoginName(data SQLLoginDataSourceModel) (string, diag.Diagnostics) {
	var diags diag.Diagnostics
	hasName := !data.Name.IsNull() && !data.Name.IsUnknown() && data.Name.ValueString() != ""
	hasLoginName := !data.LoginName.IsNull() && !data.LoginName.IsUnknown() && data.LoginName.ValueString() != ""

	if hasName && hasLoginName && data.Name.ValueString() != data.LoginName.ValueString() {
		diags.AddAttributeError(
			path.Root("login_name"),
			"Conflicting login name attributes",
			"Only one of `name` and `login_name` can be set, or both must have identical values.",
		)
		return "", diags
	} else if !hasName && !hasLoginName {
		diags.AddAttributeError(
			path.Root("name"),
			"Missing login name",
			"One of `name` or `login_name` must be set.",
		)
		return "", diags
	}

	if hasName {
		return data.Name.ValueString(), diags
	}
	return data.LoginName.ValueString(), diags
}

func (d *SQLLoginDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data SQLLoginDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	loginName, nameDiags := validateDataSourceLoginName(data)
	resp.Diagnostics.Append(nameDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.Server != nil {
		resp.Diagnostics.Append(validateServerConfig(data.Server)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	client, clientDiags := d.getClient(ctx, data.Server)
	resp.Diagnostics.Append(clientDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	login, err := client.GetSQLLogin(ctx, loginName)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read SQL login", err.Error())
		return
	}
	if login == nil {
		resp.Diagnostics.AddError("SQL login not found", fmt.Sprintf("Login '%s' not found", loginName))
		return
	}

	data.ID = types.StringValue(strconv.Itoa(login.PrincipalID))
	data.Name = types.StringValue(login.Name)
	if !data.LoginName.IsNull() {
		data.LoginName = types.StringValue(login.Name)
	}
	data.SID = types.StringValue(login.SID)
	data.DefaultDatabase = types.StringValue(login.DefaultDatabaseName)
	data.DefaultLanguage = types.StringValue(login.DefaultLanguageName)
	data.CheckExpirationEnabled = types.BoolValue(login.CheckExpirationEnabled)
	data.CheckPolicyEnabled = types.BoolValue(login.CheckPolicyEnabled)
	data.IsDisabled = types.BoolValue(login.IsDisabled)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// SQLLogins data source
var _ datasource.DataSource = &SQLLoginsDataSource{}

func NewSQLLoginsDataSource() datasource.DataSource {
	return &SQLLoginsDataSource{}
}

type SQLLoginsDataSource struct {
	client *mssql.Client
}

type SQLLoginsItemModel struct {
	ID                     types.String `tfsdk:"id"`
	Name                   types.String `tfsdk:"name"`
	SID                    types.String `tfsdk:"sid"`
	DefaultDatabase        types.String `tfsdk:"default_database"`
	DefaultLanguage        types.String `tfsdk:"default_language"`
	CheckExpirationEnabled types.Bool   `tfsdk:"check_expiration_enabled"`
	CheckPolicyEnabled     types.Bool   `tfsdk:"check_policy_enabled"`
	IsDisabled             types.Bool   `tfsdk:"is_disabled"`
}

type SQLLoginsDataSourceModel struct {
	Logins []SQLLoginsItemModel `tfsdk:"logins"`
}

func (d *SQLLoginsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sql_logins"
}

func (d *SQLLoginsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Use this data source to get information about all SQL Server logins.",
		Attributes: map[string]schema.Attribute{
			"logins": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":                       schema.StringAttribute{Computed: true},
						"name":                     schema.StringAttribute{Computed: true},
						"sid":                      schema.StringAttribute{Description: "The SID (Security Identifier) of the SQL login in hexadecimal format.", Computed: true},
						"default_database":         schema.StringAttribute{Computed: true},
						"default_language":         schema.StringAttribute{Computed: true},
						"check_expiration_enabled": schema.BoolAttribute{Computed: true},
						"check_policy_enabled":     schema.BoolAttribute{Computed: true},
						"is_disabled":              schema.BoolAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *SQLLoginsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*mssql.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Data Source Configure Type", fmt.Sprintf("Expected *mssql.Client, got: %T.", req.ProviderData))
		return
	}
	d.client = client
}

func (d *SQLLoginsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data SQLLoginsDataSourceModel

	logins, err := d.client.ListSQLLogins(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to list SQL logins", err.Error())
		return
	}

	for _, login := range logins {
		data.Logins = append(data.Logins, SQLLoginsItemModel{
			ID:                     types.StringValue(strconv.Itoa(login.PrincipalID)),
			Name:                   types.StringValue(login.Name),
			SID:                    types.StringValue(login.SID),
			DefaultDatabase:        types.StringValue(login.DefaultDatabaseName),
			DefaultLanguage:        types.StringValue(login.DefaultLanguageName),
			CheckExpirationEnabled: types.BoolValue(login.CheckExpirationEnabled),
			CheckPolicyEnabled:     types.BoolValue(login.CheckPolicyEnabled),
			IsDisabled:             types.BoolValue(login.IsDisabled),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
