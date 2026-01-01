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

var _ resource.Resource = &SchemaPermissionResource{}
var _ resource.ResourceWithImportState = &SchemaPermissionResource{}

func NewSchemaPermissionResource() resource.Resource {
	return &SchemaPermissionResource{}
}

type SchemaPermissionResource struct {
	client *mssql.Client
}

type SchemaPermissionResourceModel struct {
	ID              types.String `tfsdk:"id"`
	DatabaseName    types.String `tfsdk:"database_name"`
	SchemaName      types.String `tfsdk:"schema_name"`
	PrincipalName   types.String `tfsdk:"principal_name"`
	Permission      types.String `tfsdk:"permission"`
	WithGrantOption types.Bool   `tfsdk:"with_grant_option"`
}

func (r *SchemaPermissionResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_schema_permission"
}

func (r *SchemaPermissionResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a schema-level permission grant.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"database_name": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"schema_name": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
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

func (r *SchemaPermissionResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *SchemaPermissionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data SchemaPermissionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.GrantSchemaPermission(ctx, data.DatabaseName.ValueString(), data.SchemaName.ValueString(), data.PrincipalName.ValueString(), data.Permission.ValueString(), data.WithGrantOption.ValueBool())
	if err != nil {
		resp.Diagnostics.AddError("Failed to grant schema permission", err.Error())
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%s/%s/%s/%s", data.DatabaseName.ValueString(), data.SchemaName.ValueString(), data.PrincipalName.ValueString(), strings.ToUpper(data.Permission.ValueString())))
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SchemaPermissionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data SchemaPermissionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	perm, err := r.client.GetSchemaPermission(ctx, data.DatabaseName.ValueString(), data.SchemaName.ValueString(), data.PrincipalName.ValueString(), data.Permission.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read schema permission", err.Error())
		return
	}
	if perm == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	// Only update WithGrantOption if this is a real permission (DatabaseID > 0).
	// Virtual permissions for schema owners have DatabaseID = 0 and we should
	// preserve the Terraform-configured value to avoid drift.
	if perm.DatabaseID > 0 {
		data.WithGrantOption = types.BoolValue(perm.WithGrantOption)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}


func (r *SchemaPermissionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data, state SchemaPermissionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !data.WithGrantOption.Equal(state.WithGrantOption) {
		if err := r.client.RevokeSchemaPermission(ctx, data.DatabaseName.ValueString(), data.SchemaName.ValueString(), data.PrincipalName.ValueString(), data.Permission.ValueString()); err != nil {
			resp.Diagnostics.AddError("Failed to revoke schema permission", err.Error())
			return
		}
		if err := r.client.GrantSchemaPermission(ctx, data.DatabaseName.ValueString(), data.SchemaName.ValueString(), data.PrincipalName.ValueString(), data.Permission.ValueString(), data.WithGrantOption.ValueBool()); err != nil {
			resp.Diagnostics.AddError("Failed to grant schema permission", err.Error())
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SchemaPermissionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data SchemaPermissionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.RevokeSchemaPermission(ctx, data.DatabaseName.ValueString(), data.SchemaName.ValueString(), data.PrincipalName.ValueString(), data.Permission.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to revoke schema permission", err.Error())
		return
	}
}

func (r *SchemaPermissionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 4 {
		resp.Diagnostics.AddError("Invalid import ID", "Import ID must be in format 'database_name/schema_name/principal_name/permission'")
		return
	}

	perm, err := r.client.GetSchemaPermission(ctx, parts[0], parts[1], parts[2], parts[3])
	if err != nil {
		resp.Diagnostics.AddError("Failed to import schema permission", err.Error())
		return
	}
	if perm == nil {
		resp.Diagnostics.AddError("Schema permission not found", fmt.Sprintf("Permission '%s' not found", parts[3]))
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("database_name"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("schema_name"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("principal_name"), parts[2])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("permission"), perm.PermissionName)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("with_grant_option"), perm.WithGrantOption)...)
}
