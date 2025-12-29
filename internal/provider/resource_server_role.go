// Copyright (c) 2024 muecahit94
// SPDX-License-Identifier: MIT

package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/muecahit94/terraform-provider-mssql/internal/mssql"
)

var _ resource.Resource = &ServerRoleResource{}
var _ resource.ResourceWithImportState = &ServerRoleResource{}

func NewServerRoleResource() resource.Resource {
	return &ServerRoleResource{}
}

type ServerRoleResource struct {
	client *mssql.Client
}

type ServerRoleResourceModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	OwnerName types.String `tfsdk:"owner_name"`
}

func (r *ServerRoleResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_server_role"
}

func (r *ServerRoleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a SQL Server server role.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"owner_name": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
		},
	}
}

func (r *ServerRoleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ServerRoleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ServerRoleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	role, err := r.client.CreateServerRole(ctx, mssql.CreateServerRoleOptions{
		RoleName:  data.Name.ValueString(),
		OwnerName: data.OwnerName.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create server role", err.Error())
		return
	}

	data.ID = types.StringValue(strconv.Itoa(role.PrincipalID))
	data.OwnerName = types.StringValue(role.OwnerName)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ServerRoleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ServerRoleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, _ := strconv.Atoi(data.ID.ValueString())
	role, err := r.client.GetServerRoleByID(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read server role", err.Error())
		return
	}
	if role == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	data.Name = types.StringValue(role.Name)
	data.OwnerName = types.StringValue(role.OwnerName)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ServerRoleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Update Not Supported", "Server role does not support in-place updates.")
}

func (r *ServerRoleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ServerRoleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DropServerRole(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete server role", err.Error())
		return
	}
}

func (r *ServerRoleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	role, err := r.client.GetServerRole(ctx, req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to import server role", err.Error())
		return
	}
	if role == nil {
		resp.Diagnostics.AddError("Server role not found", fmt.Sprintf("Role '%s' not found", req.ID))
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), strconv.Itoa(role.PrincipalID))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), role.Name)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("owner_name"), role.OwnerName)...)
}
