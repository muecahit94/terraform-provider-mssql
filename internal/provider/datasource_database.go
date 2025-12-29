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

var _ datasource.DataSource = &DatabaseDataSource{}

func NewDatabaseDataSource() datasource.DataSource {
	return &DatabaseDataSource{}
}

type DatabaseDataSource struct {
	client *mssql.Client
}

type DatabaseDataSourceModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

func (d *DatabaseDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_database"
}

func (d *DatabaseDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Use this data source to get information about a SQL Server database.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"name": schema.StringAttribute{
				Required: true,
			},
		},
	}
}

func (d *DatabaseDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *DatabaseDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data DatabaseDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	db, err := d.client.GetDatabase(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read database", err.Error())
		return
	}
	if db == nil {
		resp.Diagnostics.AddError("Database not found", fmt.Sprintf("Database '%s' not found", data.Name.ValueString()))
		return
	}

	data.ID = types.StringValue(strconv.Itoa(db.ID))
	data.Name = types.StringValue(db.Name)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Databases data source
var _ datasource.DataSource = &DatabasesDataSource{}

func NewDatabasesDataSource() datasource.DataSource {
	return &DatabasesDataSource{}
}

type DatabasesDataSource struct {
	client *mssql.Client
}

type DatabasesDataSourceModel struct {
	Databases []DatabaseDataSourceModel `tfsdk:"databases"`
}

func (d *DatabasesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_databases"
}

func (d *DatabasesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Use this data source to get information about all SQL Server databases.",
		Attributes: map[string]schema.Attribute{
			"databases": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":   schema.StringAttribute{Computed: true},
						"name": schema.StringAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *DatabasesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *DatabasesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data DatabasesDataSourceModel

	dbs, err := d.client.ListDatabases(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to list databases", err.Error())
		return
	}

	for _, db := range dbs {
		data.Databases = append(data.Databases, DatabaseDataSourceModel{
			ID:   types.StringValue(strconv.Itoa(db.ID)),
			Name: types.StringValue(db.Name),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
