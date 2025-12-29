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

var _ datasource.DataSource = &SchemaDataSource{}

func NewSchemaDataSource() datasource.DataSource {
	return &SchemaDataSource{}
}

type SchemaDataSource struct {
	client *mssql.Client
}

type SchemaDataSourceModel struct {
	ID           types.String `tfsdk:"id"`
	DatabaseName types.String `tfsdk:"database_name"`
	Name         types.String `tfsdk:"name"`
	OwnerName    types.String `tfsdk:"owner_name"`
}

func (d *SchemaDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_schema"
}

func (d *SchemaDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Use this data source to get information about a database schema.",
		Attributes: map[string]schema.Attribute{
			"id":            schema.StringAttribute{Computed: true},
			"database_name": schema.StringAttribute{Required: true},
			"name":          schema.StringAttribute{Required: true},
			"owner_name":    schema.StringAttribute{Computed: true},
		},
	}
}

func (d *SchemaDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *SchemaDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data SchemaDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	schema, err := d.client.GetSchema(ctx, data.DatabaseName.ValueString(), data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read schema", err.Error())
		return
	}
	if schema == nil {
		resp.Diagnostics.AddError("Schema not found", fmt.Sprintf("Schema '%s' not found", data.Name.ValueString()))
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%d/%d", schema.DatabaseID, schema.SchemaID))
	data.OwnerName = types.StringValue(schema.OwnerName)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Schemas data source
var _ datasource.DataSource = &SchemasDataSource{}

func NewSchemasDataSource() datasource.DataSource {
	return &SchemasDataSource{}
}

type SchemasDataSource struct {
	client *mssql.Client
}

type SchemasDataSourceModel struct {
	DatabaseName types.String            `tfsdk:"database_name"`
	Schemas      []SchemaDataSourceModel `tfsdk:"schemas"`
}

func (d *SchemasDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_schemas"
}

func (d *SchemasDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Use this data source to get information about all schemas in a database.",
		Attributes: map[string]schema.Attribute{
			"database_name": schema.StringAttribute{Required: true},
			"schemas": schema.ListNestedAttribute{
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

func (d *SchemasDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *SchemasDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data SchemasDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	schemas, err := d.client.ListSchemas(ctx, data.DatabaseName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to list schemas", err.Error())
		return
	}

	for _, schema := range schemas {
		data.Schemas = append(data.Schemas, SchemaDataSourceModel{
			ID:           types.StringValue(fmt.Sprintf("%d/%d", schema.DatabaseID, schema.SchemaID)),
			DatabaseName: data.DatabaseName,
			Name:         types.StringValue(schema.Name),
			OwnerName:    types.StringValue(schema.OwnerName),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// SchemaPermissions data source
var _ datasource.DataSource = &SchemaPermissionsDataSource{}

func NewSchemaPermissionsDataSource() datasource.DataSource {
	return &SchemaPermissionsDataSource{}
}

type SchemaPermissionsDataSource struct {
	client *mssql.Client
}

type SchemaPermissionsDataSourceModel struct {
	DatabaseName  types.String      `tfsdk:"database_name"`
	SchemaName    types.String      `tfsdk:"schema_name"`
	PrincipalName types.String      `tfsdk:"principal_name"`
	Permissions   []PermissionModel `tfsdk:"permissions"`
}

func (d *SchemaPermissionsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_schema_permissions"
}

func (d *SchemaPermissionsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Use this data source to get schema permissions for a principal.",
		Attributes: map[string]schema.Attribute{
			"database_name":  schema.StringAttribute{Required: true},
			"schema_name":    schema.StringAttribute{Required: true},
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

func (d *SchemaPermissionsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *SchemaPermissionsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data SchemaPermissionsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	perms, err := d.client.ListSchemaPermissions(ctx, data.DatabaseName.ValueString(), data.SchemaName.ValueString(), data.PrincipalName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to list schema permissions", err.Error())
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
