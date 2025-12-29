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

var _ datasource.DataSource = &ServerRoleDataSource{}

func NewServerRoleDataSource() datasource.DataSource {
	return &ServerRoleDataSource{}
}

type ServerRoleDataSource struct {
	client *mssql.Client
}

type ServerRoleDataSourceModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	OwnerName types.String `tfsdk:"owner_name"`
}

func (d *ServerRoleDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_server_role"
}

func (d *ServerRoleDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Use this data source to get information about a server role.",
		Attributes: map[string]schema.Attribute{
			"id":         schema.StringAttribute{Computed: true},
			"name":       schema.StringAttribute{Required: true},
			"owner_name": schema.StringAttribute{Computed: true},
		},
	}
}

func (d *ServerRoleDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ServerRoleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ServerRoleDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	role, err := d.client.GetServerRole(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read server role", err.Error())
		return
	}
	if role == nil {
		resp.Diagnostics.AddError("Server role not found", fmt.Sprintf("Role '%s' not found", data.Name.ValueString()))
		return
	}

	data.ID = types.StringValue(strconv.Itoa(role.PrincipalID))
	data.OwnerName = types.StringValue(role.OwnerName)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// ServerRoles data source
var _ datasource.DataSource = &ServerRolesDataSource{}

func NewServerRolesDataSource() datasource.DataSource {
	return &ServerRolesDataSource{}
}

type ServerRolesDataSource struct {
	client *mssql.Client
}

type ServerRolesDataSourceModel struct {
	Roles []ServerRoleDataSourceModel `tfsdk:"roles"`
}

func (d *ServerRolesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_server_roles"
}

func (d *ServerRolesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Use this data source to get information about all server roles.",
		Attributes: map[string]schema.Attribute{
			"roles": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":         schema.StringAttribute{Computed: true},
						"name":       schema.StringAttribute{Computed: true},
						"owner_name": schema.StringAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *ServerRolesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ServerRolesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ServerRolesDataSourceModel

	roles, err := d.client.ListServerRoles(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to list server roles", err.Error())
		return
	}

	for _, role := range roles {
		data.Roles = append(data.Roles, ServerRoleDataSourceModel{
			ID:        types.StringValue(strconv.Itoa(role.PrincipalID)),
			Name:      types.StringValue(role.Name),
			OwnerName: types.StringValue(role.OwnerName),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// ServerPermissions data source
var _ datasource.DataSource = &ServerPermissionsDataSource{}

func NewServerPermissionsDataSource() datasource.DataSource {
	return &ServerPermissionsDataSource{}
}

type ServerPermissionsDataSource struct {
	client *mssql.Client
}

type ServerPermissionsDataSourceModel struct {
	PrincipalName types.String      `tfsdk:"principal_name"`
	Permissions   []PermissionModel `tfsdk:"permissions"`
}

func (d *ServerPermissionsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_server_permissions"
}

func (d *ServerPermissionsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Use this data source to get server permissions for a principal.",
		Attributes: map[string]schema.Attribute{
			"principal_name": schema.StringAttribute{Required: true},
			"permissions": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"permission":        schema.StringAttribute{Computed: true},
						"with_grant_option": schema.BoolAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *ServerPermissionsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ServerPermissionsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ServerPermissionsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	perms, err := d.client.ListServerPermissions(ctx, data.PrincipalName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to list server permissions", err.Error())
		return
	}

	for _, perm := range perms {
		data.Permissions = append(data.Permissions, PermissionModel{
			Permission:      types.StringValue(perm.PermissionName),
			WithGrantOption: types.BoolValue(perm.WithGrantOption),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
