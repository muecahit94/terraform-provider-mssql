// Copyright (c) 2024 muecahit94
// SPDX-License-Identifier: MIT

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/muecahit94/terraform-provider-mssql/internal/mssql"
)

var _ datasource.DataSource = &QueryDataSource{}

func NewQueryDataSource() datasource.DataSource {
	return &QueryDataSource{}
}

type QueryDataSource struct {
	client *mssql.Client
}

type QueryRowModel struct {
	Values types.Map `tfsdk:"values"`
}

type QueryDataSourceModel struct {
	DatabaseName types.String `tfsdk:"database_name"`
	Query        types.String `tfsdk:"query"`
	Result       types.List   `tfsdk:"result"`
}

func (d *QueryDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_query"
}

func (d *QueryDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Execute a custom SQL query and return the results.",
		Attributes: map[string]schema.Attribute{
			"database_name": schema.StringAttribute{
				Description: "The database to execute the query in. Empty for server-level queries.",
				Optional:    true,
			},
			"query": schema.StringAttribute{
				Description: "The SQL query to execute. Must be a SELECT statement.",
				Required:    true,
			},
			"result": schema.ListNestedAttribute{
				Description: "The query results.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"values": schema.MapAttribute{
							Description: "The column values for this row.",
							Computed:    true,
							ElementType: types.StringType,
						},
					},
				},
			},
		},
	}
}

func (d *QueryDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *QueryDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data QueryDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := d.client.ExecuteQuery(ctx, data.DatabaseName.ValueString(), data.Query.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to execute query", err.Error())
		return
	}

	var rows []QueryRowModel
	for _, row := range result.Rows {
		mapValue, diags := types.MapValueFrom(ctx, types.StringType, row)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		rows = append(rows, QueryRowModel{Values: mapValue})
	}

	resultList, diags := types.ListValueFrom(ctx, types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"values": types.MapType{ElemType: types.StringType},
		},
	}, rows)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.Result = resultList
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
