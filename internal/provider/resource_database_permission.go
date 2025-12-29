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

var _ resource.Resource = &DatabasePermissionResource{}
var _ resource.ResourceWithImportState = &DatabasePermissionResource{}

func NewDatabasePermissionResource() resource.Resource {
	return &DatabasePermissionResource{}
}

type DatabasePermissionResource struct {
	client *mssql.Client
}

type DatabasePermissionResourceModel struct {
	ID              types.String `tfsdk:"id"`
	DatabaseName    types.String `tfsdk:"database_name"`
	PrincipalName   types.String `tfsdk:"principal_name"`
	Permission      types.String `tfsdk:"permission"`
	WithGrantOption types.Bool   `tfsdk:"with_grant_option"`
}

func (r *DatabasePermissionResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_database_permission"
}

func (r *DatabasePermissionResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a database-level permission grant.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The permission ID in format 'database_name/principal_name/permission'.",
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
			"principal_name": schema.StringAttribute{
				Description: "The name of the principal (user or role).",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"permission": schema.StringAttribute{
				Description: "The permission to grant (e.g., SELECT, INSERT, UPDATE, DELETE, EXECUTE, etc.).",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"with_grant_option": schema.BoolAttribute{
				Description: "Whether the principal can grant this permission to others.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
		},
	}
}

func (r *DatabasePermissionResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *DatabasePermissionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data DatabasePermissionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.GrantDatabasePermission(ctx, data.DatabaseName.ValueString(), data.PrincipalName.ValueString(), data.Permission.ValueString(), data.WithGrantOption.ValueBool())
	if err != nil {
		resp.Diagnostics.AddError("Failed to grant database permission", err.Error())
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%s/%s/%s", data.DatabaseName.ValueString(), data.PrincipalName.ValueString(), strings.ToUpper(data.Permission.ValueString())))
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DatabasePermissionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data DatabasePermissionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	perm, err := r.client.GetDatabasePermission(ctx, data.DatabaseName.ValueString(), data.PrincipalName.ValueString(), data.Permission.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read database permission", err.Error())
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

func (r *DatabasePermissionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data, state DatabasePermissionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If with_grant_option changed, we need to revoke and re-grant
	if !data.WithGrantOption.Equal(state.WithGrantOption) {
		if err := r.client.RevokeDatabasePermission(ctx, data.DatabaseName.ValueString(), data.PrincipalName.ValueString(), data.Permission.ValueString()); err != nil {
			resp.Diagnostics.AddError("Failed to revoke database permission", err.Error())
			return
		}
		if err := r.client.GrantDatabasePermission(ctx, data.DatabaseName.ValueString(), data.PrincipalName.ValueString(), data.Permission.ValueString(), data.WithGrantOption.ValueBool()); err != nil {
			resp.Diagnostics.AddError("Failed to grant database permission", err.Error())
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DatabasePermissionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data DatabasePermissionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.RevokeDatabasePermission(ctx, data.DatabaseName.ValueString(), data.PrincipalName.ValueString(), data.Permission.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to revoke database permission", err.Error())
		return
	}
}

func (r *DatabasePermissionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 3 {
		resp.Diagnostics.AddError("Invalid import ID", "Import ID must be in format 'database_name/principal_name/permission'")
		return
	}

	perm, err := r.client.GetDatabasePermission(ctx, parts[0], parts[1], parts[2])
	if err != nil {
		resp.Diagnostics.AddError("Failed to import database permission", err.Error())
		return
	}
	if perm == nil {
		resp.Diagnostics.AddError("Database permission not found", fmt.Sprintf("Permission '%s' not found for '%s'", parts[2], parts[1]))
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("database_name"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("principal_name"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("permission"), perm.PermissionName)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("with_grant_option"), perm.WithGrantOption)...)
}
