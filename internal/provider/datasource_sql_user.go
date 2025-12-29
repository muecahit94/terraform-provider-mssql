// Copyright (c) 2024 muecahit94
// SPDX-License-Identifier: MIT

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/muecahit94/terraform-provider-mssql/internal/mssql"
)

var _ datasource.DataSource = &SQLUserDataSource{}

func NewSQLUserDataSource() datasource.DataSource {
	return &SQLUserDataSource{}
}

type SQLUserDataSource struct {
	client *mssql.Client
}

type SQLUserDataSourceModel struct {
	ID            types.String `tfsdk:"id"`
	DatabaseName  types.String `tfsdk:"database_name"`
	Name          types.String `tfsdk:"name"`
	LoginName     types.String `tfsdk:"login_name"`
	DefaultSchema types.String `tfsdk:"default_schema"`
}

func (d *SQLUserDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sql_user"
}

func (d *SQLUserDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Use this data source to get information about a SQL Server database user.",
		Attributes: map[string]schema.Attribute{
			"id":             schema.StringAttribute{Computed: true},
			"database_name":  schema.StringAttribute{Required: true},
			"name":           schema.StringAttribute{Required: true},
			"login_name":     schema.StringAttribute{Computed: true},
			"default_schema": schema.StringAttribute{Computed: true},
		},
	}
}

func (d *SQLUserDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *SQLUserDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data SQLUserDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	user, err := d.client.GetUser(ctx, data.DatabaseName.ValueString(), data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read SQL user", err.Error())
		return
	}
	if user == nil {
		resp.Diagnostics.AddError("SQL user not found", fmt.Sprintf("User '%s' not found in database '%s'", data.Name.ValueString(), data.DatabaseName.ValueString()))
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%d/%d", user.DatabaseID, user.PrincipalID))
	data.LoginName = types.StringValue(user.LoginName)
	data.DefaultSchema = types.StringValue(user.DefaultSchemaName)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// SQLUsers data source
var _ datasource.DataSource = &SQLUsersDataSource{}

func NewSQLUsersDataSource() datasource.DataSource {
	return &SQLUsersDataSource{}
}

type SQLUsersDataSource struct {
	client *mssql.Client
}

type SQLUsersDataSourceModel struct {
	DatabaseName types.String             `tfsdk:"database_name"`
	Users        []SQLUserDataSourceModel `tfsdk:"users"`
}

func (d *SQLUsersDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sql_users"
}

func (d *SQLUsersDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Use this data source to get information about all users in a database.",
		Attributes: map[string]schema.Attribute{
			"database_name": schema.StringAttribute{Required: true},
			"users": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":             schema.StringAttribute{Computed: true},
						"database_name":  schema.StringAttribute{Computed: true},
						"name":           schema.StringAttribute{Computed: true},
						"login_name":     schema.StringAttribute{Computed: true},
						"default_schema": schema.StringAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *SQLUsersDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *SQLUsersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data SQLUsersDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	users, err := d.client.ListUsers(ctx, data.DatabaseName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to list SQL users", err.Error())
		return
	}

	for _, user := range users {
		data.Users = append(data.Users, SQLUserDataSourceModel{
			ID:            types.StringValue(fmt.Sprintf("%d/%d", user.DatabaseID, user.PrincipalID)),
			DatabaseName:  data.DatabaseName,
			Name:          types.StringValue(user.Name),
			LoginName:     types.StringValue(user.LoginName),
			DefaultSchema: types.StringValue(user.DefaultSchemaName),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
