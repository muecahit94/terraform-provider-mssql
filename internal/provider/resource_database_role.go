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
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/muecahit94/terraform-provider-mssql/internal/mssql"
)

var _ resource.Resource = &DatabaseRoleResource{}
var _ resource.ResourceWithImportState = &DatabaseRoleResource{}

func NewDatabaseRoleResource() resource.Resource {
	return &DatabaseRoleResource{}
}

type DatabaseRoleResource struct {
	client *mssql.Client
}

type DatabaseRoleResourceModel struct {
	ID           types.String `tfsdk:"id"`
	DatabaseName types.String `tfsdk:"database_name"`
	Name         types.String `tfsdk:"name"`
	OwnerName    types.String `tfsdk:"owner_name"`
}

func (r *DatabaseRoleResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_database_role"
}

func (r *DatabaseRoleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a SQL Server database role.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The role ID in format 'database_id/principal_id'.",
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
				Description: "The name of the role.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"owner_name": schema.StringAttribute{
				Description: "The owner of the role.",
				Optional:    true,
				Computed:    true,
			},
		},
	}
}

func (r *DatabaseRoleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *DatabaseRoleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data DatabaseRoleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating database role", map[string]interface{}{"database": data.DatabaseName.ValueString(), "name": data.Name.ValueString()})

	opts := mssql.CreateDatabaseRoleOptions{
		DatabaseName: data.DatabaseName.ValueString(),
		RoleName:     data.Name.ValueString(),
		OwnerName:    data.OwnerName.ValueString(),
	}

	role, err := r.client.CreateDatabaseRole(ctx, opts)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create database role", err.Error())
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%d/%d", role.DatabaseID, role.PrincipalID))
	data.OwnerName = types.StringValue(role.OwnerName)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DatabaseRoleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data DatabaseRoleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	role, err := r.client.GetDatabaseRole(ctx, data.DatabaseName.ValueString(), data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read database role", err.Error())
		return
	}
	if role == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	data.OwnerName = types.StringValue(role.OwnerName)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DatabaseRoleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data, state DatabaseRoleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !data.OwnerName.Equal(state.OwnerName) {
		owner := data.OwnerName.ValueString()
		_, err := r.client.UpdateDatabaseRole(ctx, mssql.UpdateDatabaseRoleOptions{
			DatabaseName: data.DatabaseName.ValueString(),
			RoleName:     data.Name.ValueString(),
			NewOwnerName: &owner,
		})
		if err != nil {
			resp.Diagnostics.AddError("Failed to update database role", err.Error())
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DatabaseRoleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data DatabaseRoleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DropDatabaseRole(ctx, data.DatabaseName.ValueString(), data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete database role", err.Error())
		return
	}
}

func (r *DatabaseRoleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 {
		resp.Diagnostics.AddError("Invalid import ID", "Import ID must be in format 'database_name/role_name'")
		return
	}

	role, err := r.client.GetDatabaseRole(ctx, parts[0], parts[1])
	if err != nil {
		resp.Diagnostics.AddError("Failed to import database role", err.Error())
		return
	}
	if role == nil {
		resp.Diagnostics.AddError("Database role not found", fmt.Sprintf("Role '%s' not found in database '%s'", parts[1], parts[0]))
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), fmt.Sprintf("%d/%d", role.DatabaseID, role.PrincipalID))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("database_name"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), role.Name)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("owner_name"), role.OwnerName)...)
}
