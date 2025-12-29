// Copyright (c) 2024 muecahit94
// SPDX-License-Identifier: MIT

package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
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
	DefaultDatabase        types.String `tfsdk:"default_database"`
	DefaultLanguage        types.String `tfsdk:"default_language"`
	CheckExpirationEnabled types.Bool   `tfsdk:"check_expiration_enabled"`
	CheckPolicyEnabled     types.Bool   `tfsdk:"check_policy_enabled"`
	IsDisabled             types.Bool   `tfsdk:"is_disabled"`
}

func (d *SQLLoginDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sql_login"
}

func (d *SQLLoginDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Use this data source to get information about a SQL Server login.",
		Attributes: map[string]schema.Attribute{
			"id":                       schema.StringAttribute{Computed: true},
			"name":                     schema.StringAttribute{Required: true},
			"default_database":         schema.StringAttribute{Computed: true},
			"default_language":         schema.StringAttribute{Computed: true},
			"check_expiration_enabled": schema.BoolAttribute{Computed: true},
			"check_policy_enabled":     schema.BoolAttribute{Computed: true},
			"is_disabled":              schema.BoolAttribute{Computed: true},
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

func (d *SQLLoginDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data SQLLoginDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	login, err := d.client.GetSQLLogin(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read SQL login", err.Error())
		return
	}
	if login == nil {
		resp.Diagnostics.AddError("SQL login not found", fmt.Sprintf("Login '%s' not found", data.Name.ValueString()))
		return
	}

	data.ID = types.StringValue(strconv.Itoa(login.PrincipalID))
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

type SQLLoginsDataSourceModel struct {
	Logins []SQLLoginDataSourceModel `tfsdk:"logins"`
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
		data.Logins = append(data.Logins, SQLLoginDataSourceModel{
			ID:                     types.StringValue(strconv.Itoa(login.PrincipalID)),
			Name:                   types.StringValue(login.Name),
			DefaultDatabase:        types.StringValue(login.DefaultDatabaseName),
			DefaultLanguage:        types.StringValue(login.DefaultLanguageName),
			CheckExpirationEnabled: types.BoolValue(login.CheckExpirationEnabled),
			CheckPolicyEnabled:     types.BoolValue(login.CheckPolicyEnabled),
			IsDisabled:             types.BoolValue(login.IsDisabled),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
