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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/muecahit94/terraform-provider-mssql/internal/mssql"
)

var _ resource.Resource = &ServerPermissionResource{}
var _ resource.ResourceWithImportState = &ServerPermissionResource{}

func NewServerPermissionResource() resource.Resource {
	return &ServerPermissionResource{}
}

type ServerPermissionResource struct {
	client *mssql.Client
}

type ServerPermissionResourceModel struct {
	ID              types.String `tfsdk:"id"`
	PrincipalName   types.String `tfsdk:"principal_name"`
	Permission      types.String `tfsdk:"permission"`
	WithGrantOption types.Bool   `tfsdk:"with_grant_option"`
}

func (r *ServerPermissionResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_server_permission"
}

func (r *ServerPermissionResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a server-level permission grant.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"principal_name": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"permission": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"with_grant_option": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},
		},
	}
}

func (r *ServerPermissionResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ServerPermissionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ServerPermissionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.GrantServerPermission(ctx, data.PrincipalName.ValueString(), data.Permission.ValueString(), data.WithGrantOption.ValueBool())
	if err != nil {
		resp.Diagnostics.AddError("Failed to grant server permission", err.Error())
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%s/%s", data.PrincipalName.ValueString(), strings.ToUpper(data.Permission.ValueString())))
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ServerPermissionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ServerPermissionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	perm, err := r.client.GetServerPermission(ctx, data.PrincipalName.ValueString(), data.Permission.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read server permission", err.Error())
		return
	}
	if perm == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	data.Permission = types.StringValue(perm.PermissionName)
	data.WithGrantOption = types.BoolValue(perm.WithGrantOption)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ServerPermissionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data, state ServerPermissionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !data.WithGrantOption.Equal(state.WithGrantOption) {
		if err := r.client.RevokeServerPermission(ctx, data.PrincipalName.ValueString(), data.Permission.ValueString()); err != nil {
			resp.Diagnostics.AddError("Failed to revoke server permission", err.Error())
			return
		}
		if err := r.client.GrantServerPermission(ctx, data.PrincipalName.ValueString(), data.Permission.ValueString(), data.WithGrantOption.ValueBool()); err != nil {
			resp.Diagnostics.AddError("Failed to grant server permission", err.Error())
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ServerPermissionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ServerPermissionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.RevokeServerPermission(ctx, data.PrincipalName.ValueString(), data.Permission.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to revoke server permission", err.Error())
		return
	}
}

func (r *ServerPermissionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 {
		resp.Diagnostics.AddError("Invalid import ID", "Import ID must be in format 'principal_name/permission'")
		return
	}

	perm, err := r.client.GetServerPermission(ctx, parts[0], parts[1])
	if err != nil {
		resp.Diagnostics.AddError("Failed to import server permission", err.Error())
		return
	}
	if perm == nil {
		resp.Diagnostics.AddError("Server permission not found", fmt.Sprintf("Permission '%s' not found for '%s'", parts[1], parts[0]))
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("principal_name"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("permission"), perm.PermissionName)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("with_grant_option"), perm.WithGrantOption)...)
}
