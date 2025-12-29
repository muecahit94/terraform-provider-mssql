// Copyright (c) 2024 muecahit94
// SPDX-License-Identifier: MIT

package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/muecahit94/terraform-provider-mssql/internal/mssql"
)

var _ resource.Resource = &SchemaResource{}
var _ resource.ResourceWithImportState = &SchemaResource{}

func NewSchemaResource() resource.Resource {
	return &SchemaResource{}
}

type SchemaResource struct {
	client *mssql.Client
}

type SchemaResourceModel struct {
	ID           types.String `tfsdk:"id"`
	DatabaseName types.String `tfsdk:"database_name"`
	Name         types.String `tfsdk:"name"`
	OwnerName    types.String `tfsdk:"owner_name"`
}

func (r *SchemaResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_schema"
}

func (r *SchemaResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a SQL Server database schema.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The schema ID in format 'database_id/schema_id'.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"database_name": schema.StringAttribute{
				Description: "The name of the database.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the schema.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"owner_name": schema.StringAttribute{
				Description: "The owner of the schema.",
				Optional:    true,
				Computed:    true,
			},
		},
	}
}

func (r *SchemaResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*mssql.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("Expected *mssql.Client, got: %T.", req.ProviderData))
		return
	}
	r.client = client
}

func (r *SchemaResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data SchemaResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	schema, err := r.client.CreateSchema(ctx, mssql.CreateSchemaOptions{
		DatabaseName: data.DatabaseName.ValueString(),
		SchemaName:   data.Name.ValueString(),
		OwnerName:    data.OwnerName.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create schema", err.Error())
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%d/%d", schema.DatabaseID, schema.SchemaID))
	data.OwnerName = types.StringValue(schema.OwnerName)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SchemaResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data SchemaResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	schema, err := r.client.GetSchema(ctx, data.DatabaseName.ValueString(), data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read schema", err.Error())
		return
	}
	if schema == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	data.OwnerName = types.StringValue(schema.OwnerName)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SchemaResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data, state SchemaResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !data.OwnerName.Equal(state.OwnerName) {
		owner := data.OwnerName.ValueString()
		_, err := r.client.UpdateSchema(ctx, mssql.UpdateSchemaOptions{
			DatabaseName: data.DatabaseName.ValueString(),
			SchemaName:   data.Name.ValueString(),
			NewOwnerName: &owner,
		})
		if err != nil {
			resp.Diagnostics.AddError("Failed to update schema", err.Error())
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SchemaResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data SchemaResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DropSchema(ctx, data.DatabaseName.ValueString(), data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete schema", err.Error())
		return
	}
}

func (r *SchemaResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 {
		resp.Diagnostics.AddError("Invalid import ID", "Import ID must be in format 'database_name/schema_name'")
		return
	}

	schema, err := r.client.GetSchema(ctx, parts[0], parts[1])
	if err != nil {
		resp.Diagnostics.AddError("Failed to import schema", err.Error())
		return
	}
	if schema == nil {
		resp.Diagnostics.AddError("Schema not found", fmt.Sprintf("Schema '%s' not found in database '%s'", parts[1], parts[0]))
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), fmt.Sprintf("%d/%d", schema.DatabaseID, schema.SchemaID))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("database_name"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), schema.Name)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("owner_name"), schema.OwnerName)...)
}
