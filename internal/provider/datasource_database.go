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
	ID                 types.String `tfsdk:"id"`
	Name               types.String `tfsdk:"name"`
	Collation          types.String `tfsdk:"collation"`
	CompatibilityLevel types.Int64  `tfsdk:"compatibility_level"`
	RecoveryModel      types.String `tfsdk:"recovery_model"`

	AutoClose             types.Bool   `tfsdk:"auto_close"`
	AutoShrink            types.Bool   `tfsdk:"auto_shrink"`
	PageVerify            types.String `tfsdk:"page_verify"`
	SnapshotIsolation     types.Bool   `tfsdk:"snapshot_isolation"`
	ReadCommittedSnapshot types.Bool   `tfsdk:"read_committed_snapshot"`
	QueryStore            types.Bool   `tfsdk:"query_store"`
	Trustworthy           types.Bool   `tfsdk:"trustworthy"`
}

func databaseDataSourceModel(db mssql.Database) DatabaseDataSourceModel {
	return DatabaseDataSourceModel{
		ID:                 types.StringValue(strconv.Itoa(db.ID)),
		Name:               types.StringValue(db.Name),
		Collation:          types.StringValue(db.Collation),
		CompatibilityLevel: types.Int64Value(int64(db.CompatibilityLevel)),
		RecoveryModel:      types.StringValue(db.RecoveryModel),

		AutoClose:             types.BoolValue(db.AutoClose),
		AutoShrink:            types.BoolValue(db.AutoShrink),
		PageVerify:            types.StringValue(db.PageVerify),
		SnapshotIsolation:     types.BoolValue(db.SnapshotIsolation),
		ReadCommittedSnapshot: types.BoolValue(db.ReadCommittedSnapshot),
		QueryStore:            types.BoolValue(db.QueryStore),
		Trustworthy:           types.BoolValue(db.Trustworthy),
	}
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
			"collation": schema.StringAttribute{
				Description: "The collation of the database.",
				Computed:    true,
			},
			"compatibility_level": schema.Int64Attribute{
				Description: "The compatibility level of the database.",
				Computed:    true,
			},
			"recovery_model": schema.StringAttribute{
				Description: "The recovery model of the database: `FULL`, `SIMPLE` or `BULK_LOGGED`.",
				Computed:    true,
			},
			"auto_close":              schema.BoolAttribute{Description: "Whether `AUTO_CLOSE` is on.", Computed: true},
			"auto_shrink":             schema.BoolAttribute{Description: "Whether `AUTO_SHRINK` is on.", Computed: true},
			"page_verify":             schema.StringAttribute{Description: "`CHECKSUM`, `TORN_PAGE_DETECTION` or `NONE`.", Computed: true},
			"snapshot_isolation":      schema.BoolAttribute{Description: "Whether snapshot isolation is allowed.", Computed: true},
			"read_committed_snapshot": schema.BoolAttribute{Description: "Whether `READ_COMMITTED_SNAPSHOT` is on.", Computed: true},
			"query_store":             schema.BoolAttribute{Description: "Whether Query Store is on.", Computed: true},
			"trustworthy":             schema.BoolAttribute{Description: "Whether `TRUSTWORTHY` is on.", Computed: true},
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

	data = databaseDataSourceModel(*db)
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
						"id":                      schema.StringAttribute{Computed: true},
						"name":                    schema.StringAttribute{Computed: true},
						"collation":               schema.StringAttribute{Computed: true},
						"compatibility_level":     schema.Int64Attribute{Computed: true},
						"recovery_model":          schema.StringAttribute{Computed: true},
						"auto_close":              schema.BoolAttribute{Computed: true},
						"auto_shrink":             schema.BoolAttribute{Computed: true},
						"page_verify":             schema.StringAttribute{Computed: true},
						"snapshot_isolation":      schema.BoolAttribute{Computed: true},
						"read_committed_snapshot": schema.BoolAttribute{Computed: true},
						"query_store":             schema.BoolAttribute{Computed: true},
						"trustworthy":             schema.BoolAttribute{Computed: true},
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
		data.Databases = append(data.Databases, databaseDataSourceModel(db))
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
