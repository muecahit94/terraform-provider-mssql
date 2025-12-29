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

var _ resource.Resource = &ServerRoleMemberResource{}
var _ resource.ResourceWithImportState = &ServerRoleMemberResource{}

func NewServerRoleMemberResource() resource.Resource {
	return &ServerRoleMemberResource{}
}

type ServerRoleMemberResource struct {
	client *mssql.Client
}

type ServerRoleMemberResourceModel struct {
	ID         types.String `tfsdk:"id"`
	RoleName   types.String `tfsdk:"role_name"`
	MemberName types.String `tfsdk:"member_name"`
}

func (r *ServerRoleMemberResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_server_role_member"
}

func (r *ServerRoleMemberResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages membership in a SQL Server server role.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"role_name": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"member_name": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *ServerRoleMemberResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ServerRoleMemberResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ServerRoleMemberResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.AddServerRoleMember(ctx, data.RoleName.ValueString(), data.MemberName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to add server role member", err.Error())
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%s/%s", data.RoleName.ValueString(), data.MemberName.ValueString()))
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ServerRoleMemberResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ServerRoleMemberResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	member, err := r.client.GetServerRoleMember(ctx, data.RoleName.ValueString(), data.MemberName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read server role member", err.Error())
		return
	}
	if member == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ServerRoleMemberResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Update Not Supported", "Server role membership does not support updates.")
}

func (r *ServerRoleMemberResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ServerRoleMemberResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.RemoveServerRoleMember(ctx, data.RoleName.ValueString(), data.MemberName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to remove server role member", err.Error())
		return
	}
}

func (r *ServerRoleMemberResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 {
		resp.Diagnostics.AddError("Invalid import ID", "Import ID must be in format 'role_name/member_name'")
		return
	}

	member, err := r.client.GetServerRoleMember(ctx, parts[0], parts[1])
	if err != nil {
		resp.Diagnostics.AddError("Failed to import server role member", err.Error())
		return
	}
	if member == nil {
		resp.Diagnostics.AddError("Server role member not found", fmt.Sprintf("Member '%s' not found in role '%s'", parts[1], parts[0]))
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("role_name"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("member_name"), parts[1])...)
}
