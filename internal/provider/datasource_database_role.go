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

var _ datasource.DataSource = &DatabaseRoleDataSource{}

func NewDatabaseRoleDataSource() datasource.DataSource {
	return &DatabaseRoleDataSource{}
}

type DatabaseRoleDataSource struct {
	client *mssql.Client
}

type DatabaseRoleDataSourceModel struct {
	ID           types.String `tfsdk:"id"`
	DatabaseName types.String `tfsdk:"database_name"`
	Name         types.String `tfsdk:"name"`
	OwnerName    types.String `tfsdk:"owner_name"`
}

func (d *DatabaseRoleDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_database_role"
}

func (d *DatabaseRoleDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Use this data source to get information about a database role.",
		Attributes: map[string]schema.Attribute{
			"id":            schema.StringAttribute{Computed: true},
			"database_name": schema.StringAttribute{Required: true},
			"name":          schema.StringAttribute{Required: true},
			"owner_name":    schema.StringAttribute{Computed: true},
		},
	}
}

func (d *DatabaseRoleDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *DatabaseRoleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data DatabaseRoleDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	role, err := d.client.GetDatabaseRole(ctx, data.DatabaseName.ValueString(), data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read database role", err.Error())
		return
	}
	if role == nil {
		resp.Diagnostics.AddError("Database role not found", fmt.Sprintf("Role '%s' not found", data.Name.ValueString()))
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%d/%d", role.DatabaseID, role.PrincipalID))
	data.OwnerName = types.StringValue(role.OwnerName)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// DatabaseRoles data source
var _ datasource.DataSource = &DatabaseRolesDataSource{}

func NewDatabaseRolesDataSource() datasource.DataSource {
	return &DatabaseRolesDataSource{}
}

type DatabaseRolesDataSource struct {
	client *mssql.Client
}

type DatabaseRolesDataSourceModel struct {
	DatabaseName types.String                  `tfsdk:"database_name"`
	Roles        []DatabaseRoleDataSourceModel `tfsdk:"roles"`
}

func (d *DatabaseRolesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_database_roles"
}

func (d *DatabaseRolesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Use this data source to get information about all roles in a database.",
		Attributes: map[string]schema.Attribute{
			"database_name": schema.StringAttribute{Required: true},
			"roles": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":            schema.StringAttribute{Computed: true},
						"database_name": schema.StringAttribute{Computed: true},
						"name":          schema.StringAttribute{Computed: true},
						"owner_name":    schema.StringAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *DatabaseRolesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *DatabaseRolesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data DatabaseRolesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	roles, err := d.client.ListDatabaseRoles(ctx, data.DatabaseName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to list database roles", err.Error())
		return
	}

	for _, role := range roles {
		data.Roles = append(data.Roles, DatabaseRoleDataSourceModel{
			ID:           types.StringValue(fmt.Sprintf("%d/%d", role.DatabaseID, role.PrincipalID)),
			DatabaseName: data.DatabaseName,
			Name:         types.StringValue(role.Name),
			OwnerName:    types.StringValue(role.OwnerName),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
