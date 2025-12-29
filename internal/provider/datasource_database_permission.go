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

var _ datasource.DataSource = &DatabasePermissionsDataSource{}

func NewDatabasePermissionsDataSource() datasource.DataSource {
	return &DatabasePermissionsDataSource{}
}

type DatabasePermissionsDataSource struct {
	client *mssql.Client
}

type PermissionModel struct {
	Permission      types.String `tfsdk:"permission"`
	WithGrantOption types.Bool   `tfsdk:"with_grant_option"`
}

type DatabasePermissionsDataSourceModel struct {
	DatabaseName  types.String      `tfsdk:"database_name"`
	PrincipalName types.String      `tfsdk:"principal_name"`
	Permissions   []PermissionModel `tfsdk:"permissions"`
}

func (d *DatabasePermissionsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_database_permissions"
}

func (d *DatabasePermissionsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Use this data source to get database permissions for a principal.",
		Attributes: map[string]schema.Attribute{
			"database_name":  schema.StringAttribute{Required: true},
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

func (d *DatabasePermissionsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *DatabasePermissionsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data DatabasePermissionsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	perms, err := d.client.ListDatabasePermissions(ctx, data.DatabaseName.ValueString(), data.PrincipalName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to list database permissions", err.Error())
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
