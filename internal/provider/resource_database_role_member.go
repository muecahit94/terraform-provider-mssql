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

var _ resource.Resource = &DatabaseRoleMemberResource{}
var _ resource.ResourceWithImportState = &DatabaseRoleMemberResource{}

func NewDatabaseRoleMemberResource() resource.Resource {
	return &DatabaseRoleMemberResource{}
}

type DatabaseRoleMemberResource struct {
	client *mssql.Client
}

type DatabaseRoleMemberResourceModel struct {
	ID           types.String `tfsdk:"id"`
	DatabaseName types.String `tfsdk:"database_name"`
	RoleName     types.String `tfsdk:"role_name"`
	MemberName   types.String `tfsdk:"member_name"`
}

func (r *DatabaseRoleMemberResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_database_role_member"
}

func (r *DatabaseRoleMemberResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages membership in a SQL Server database role.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The membership ID in format 'database_name/role_name/member_name'.",
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
			"role_name": schema.StringAttribute{
				Description: "The name of the role.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"member_name": schema.StringAttribute{
				Description: "The name of the member (user or role).",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *DatabaseRoleMemberResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *DatabaseRoleMemberResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data DatabaseRoleMemberResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.AddDatabaseRoleMember(ctx, data.DatabaseName.ValueString(), data.RoleName.ValueString(), data.MemberName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to add database role member", err.Error())
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%s/%s/%s", data.DatabaseName.ValueString(), data.RoleName.ValueString(), data.MemberName.ValueString()))
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DatabaseRoleMemberResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data DatabaseRoleMemberResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	member, err := r.client.GetDatabaseRoleMember(ctx, data.DatabaseName.ValueString(), data.RoleName.ValueString(), data.MemberName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read database role member", err.Error())
		return
	}
	if member == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DatabaseRoleMemberResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Update Not Supported", "Role membership does not support updates. Changes require replacement.")
}

func (r *DatabaseRoleMemberResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data DatabaseRoleMemberResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.RemoveDatabaseRoleMember(ctx, data.DatabaseName.ValueString(), data.RoleName.ValueString(), data.MemberName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to remove database role member", err.Error())
		return
	}
}

func (r *DatabaseRoleMemberResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 3 {
		resp.Diagnostics.AddError("Invalid import ID", "Import ID must be in format 'database_name/role_name/member_name'")
		return
	}

	member, err := r.client.GetDatabaseRoleMember(ctx, parts[0], parts[1], parts[2])
	if err != nil {
		resp.Diagnostics.AddError("Failed to import database role member", err.Error())
		return
	}
	if member == nil {
		resp.Diagnostics.AddError("Database role member not found", fmt.Sprintf("Member '%s' not found in role '%s'", parts[2], parts[1]))
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("database_name"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("role_name"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("member_name"), parts[2])...)
}
