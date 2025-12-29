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

// Azure AD User data source
var _ datasource.DataSource = &AzureADUserDataSource{}

func NewAzureADUserDataSource() datasource.DataSource {
	return &AzureADUserDataSource{}
}

type AzureADUserDataSource struct {
	client *mssql.Client
}

type AzureADUserDataSourceModel struct {
	ID            types.String `tfsdk:"id"`
	DatabaseName  types.String `tfsdk:"database_name"`
	Name          types.String `tfsdk:"name"`
	DefaultSchema types.String `tfsdk:"default_schema"`
}

func (d *AzureADUserDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_azuread_user"
}

func (d *AzureADUserDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Use this data source to get information about an Azure AD user.",
		Attributes: map[string]schema.Attribute{
			"id":             schema.StringAttribute{Computed: true},
			"database_name":  schema.StringAttribute{Required: true},
			"name":           schema.StringAttribute{Required: true},
			"default_schema": schema.StringAttribute{Computed: true},
		},
	}
}

func (d *AzureADUserDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *AzureADUserDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data AzureADUserDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	user, err := d.client.GetUser(ctx, data.DatabaseName.ValueString(), data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read Azure AD user", err.Error())
		return
	}
	if user == nil {
		resp.Diagnostics.AddError("Azure AD user not found", fmt.Sprintf("User '%s' not found", data.Name.ValueString()))
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%d/%d", user.DatabaseID, user.PrincipalID))
	data.DefaultSchema = types.StringValue(user.DefaultSchemaName)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Azure AD Service Principal data source
var _ datasource.DataSource = &AzureADServicePrincipalDataSource{}

func NewAzureADServicePrincipalDataSource() datasource.DataSource {
	return &AzureADServicePrincipalDataSource{}
}

type AzureADServicePrincipalDataSource struct {
	client *mssql.Client
}

type AzureADServicePrincipalDataSourceModel struct {
	ID            types.String `tfsdk:"id"`
	DatabaseName  types.String `tfsdk:"database_name"`
	Name          types.String `tfsdk:"name"`
	DefaultSchema types.String `tfsdk:"default_schema"`
}

func (d *AzureADServicePrincipalDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_azuread_service_principal"
}

func (d *AzureADServicePrincipalDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Use this data source to get information about an Azure AD service principal.",
		Attributes: map[string]schema.Attribute{
			"id":             schema.StringAttribute{Computed: true},
			"database_name":  schema.StringAttribute{Required: true},
			"name":           schema.StringAttribute{Required: true},
			"default_schema": schema.StringAttribute{Computed: true},
		},
	}
}

func (d *AzureADServicePrincipalDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *AzureADServicePrincipalDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data AzureADServicePrincipalDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	user, err := d.client.GetUser(ctx, data.DatabaseName.ValueString(), data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read Azure AD service principal", err.Error())
		return
	}
	if user == nil {
		resp.Diagnostics.AddError("Azure AD service principal not found", fmt.Sprintf("Principal '%s' not found", data.Name.ValueString()))
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%d/%d", user.DatabaseID, user.PrincipalID))
	data.DefaultSchema = types.StringValue(user.DefaultSchemaName)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
