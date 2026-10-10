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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/muecahit94/terraform-provider-mssql/internal/mssql"
)

var _ resource.Resource = &DatabaseObjectPermissionResource{}
var _ resource.ResourceWithImportState = &DatabaseObjectPermissionResource{}
var _ resource.ResourceWithValidateConfig = &DatabaseObjectPermissionResource{}

// NewDatabaseObjectPermissionResource creates a new object permission resource.
func NewDatabaseObjectPermissionResource() resource.Resource {
	return &DatabaseObjectPermissionResource{}
}

// DatabaseObjectPermissionResource grants a permission on an object (table, view, procedure, ...) or one of its columns.
type DatabaseObjectPermissionResource struct {
	client *mssql.Client
}

// DatabaseObjectPermissionResourceModel describes the resource data model.
type DatabaseObjectPermissionResourceModel struct {
	ID              types.String `tfsdk:"id"`
	DatabaseName    types.String `tfsdk:"database_name"`
	SchemaName      types.String `tfsdk:"schema_name"`
	ObjectName      types.String `tfsdk:"object_name"`
	ColumnName      types.String `tfsdk:"column_name"`
	PrincipalName   types.String `tfsdk:"principal_name"`
	Permission      types.String `tfsdk:"permission"`
	WithGrantOption types.Bool   `tfsdk:"with_grant_option"`
}

// objectPermissionID builds the ID: database/schema/object[/column]/principal/permission.
func objectPermissionID(database, schemaName, objectName, columnName, principal, permission string) string {
	parts := []string{database, schemaName, objectName}
	if columnName != "" {
		parts = append(parts, columnName)
	}
	return strings.Join(append(parts, principal, strings.ToUpper(permission)), "/")
}

// parseObjectPermissionID splits an ID built by objectPermissionID. A permission such as
// VIEW DEFINITION contains a space but no slash, so the parts can be told apart by their number.
func parseObjectPermissionID(id string) (database, schemaName, objectName, columnName, principal, permission string, err error) {
	parts := strings.Split(id, "/")
	switch len(parts) {
	case 5:
		return parts[0], parts[1], parts[2], "", parts[3], parts[4], nil
	case 6:
		return parts[0], parts[1], parts[2], parts[3], parts[4], parts[5], nil
	}
	return "", "", "", "", "", "", fmt.Errorf("expected database/schema/object/principal/permission or database/schema/object/column/principal/permission, got %q", id)
}

func (r *DatabaseObjectPermissionResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_database_object_permission"
}

func requiresReplaceString(description string, required bool) schema.StringAttribute {
	return schema.StringAttribute{
		Description: description + " Changing this forces a new resource to be created.",
		Required:    required,
		Optional:    !required,
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.RequiresReplace(),
		},
	}
}

func (r *DatabaseObjectPermissionResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a permission granted on an object of a database (a table, view, procedure or function) or on one of its columns.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID: `database/schema/object[/column]/principal/permission`.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"database_name":  requiresReplaceString("The name of the database.", true),
			"schema_name":    requiresReplaceString("The schema of the object.", true),
			"object_name":    requiresReplaceString("The name of the table, view, procedure or function.", true),
			"column_name":    requiresReplaceString("The column for a column-level permission, for example `SELECT` or `UPDATE` on one column. Omit it for a permission on the whole object.", false),
			"principal_name": requiresReplaceString("The user or role that gets the permission.", true),
			"permission":     requiresReplaceString("The permission to grant, for example `SELECT`, `INSERT`, `UPDATE`, `DELETE`, `EXECUTE`, `REFERENCES`, `ALTER`, `CONTROL` or `VIEW DEFINITION`.", true),
			"with_grant_option": schema.BoolAttribute{
				Description: "Whether the principal can grant this permission to others. Changing this forces a new resource to be created. Defaults to `false`.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *DatabaseObjectPermissionResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// ValidateConfig rejects a permission name that is not a plain keyword sequence, before it can reach a statement.
func (r *DatabaseObjectPermissionResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data DatabaseObjectPermissionResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() || data.Permission.IsNull() || data.Permission.IsUnknown() {
		return
	}
	if _, err := mssql.NormalizePermission(data.Permission.ValueString()); err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("permission"), "Invalid permission", err.Error())
		return
	}
	// The state holds what is configured, an import holds the upper-case spelling: allowing both would
	// make an imported permission look different from the configuration and replace it.
	if data.Permission.ValueString() != strings.ToUpper(data.Permission.ValueString()) {
		resp.Diagnostics.AddAttributeError(path.Root("permission"), "Invalid permission",
			fmt.Sprintf("Write the permission in upper case: %q.", strings.ToUpper(data.Permission.ValueString())))
	}
}

func (r *DatabaseObjectPermissionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data DatabaseObjectPermissionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	existing, err := r.client.GetObjectPermission(ctx, data.DatabaseName.ValueString(), data.SchemaName.ValueString(), data.ObjectName.ValueString(),
		data.ColumnName.ValueString(), data.PrincipalName.ValueString(), data.Permission.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to check the existing permission", err.Error())
		return
	}
	// GRANT over a DENY removes the DENY. That is never what a grant resource should do silently.
	if existing != nil && existing.State == "D" {
		resp.Diagnostics.AddError(
			"A DENY exists for this permission",
			fmt.Sprintf("%q is denied %s on %s.%s. Granting it would remove the DENY; remove the DENY first if that is intended.",
				data.PrincipalName.ValueString(), strings.ToUpper(data.Permission.ValueString()), data.SchemaName.ValueString(), data.ObjectName.ValueString()),
		)
		return
	}

	if err := r.client.GrantObjectPermission(ctx, data.DatabaseName.ValueString(), data.SchemaName.ValueString(), data.ObjectName.ValueString(),
		data.ColumnName.ValueString(), data.PrincipalName.ValueString(), data.Permission.ValueString(), data.WithGrantOption.ValueBool()); err != nil {
		resp.Diagnostics.AddError("Failed to grant the object permission", err.Error())
		return
	}

	data.ID = types.StringValue(objectPermissionID(data.DatabaseName.ValueString(), data.SchemaName.ValueString(), data.ObjectName.ValueString(),
		data.ColumnName.ValueString(), data.PrincipalName.ValueString(), data.Permission.ValueString()))
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DatabaseObjectPermissionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data DatabaseObjectPermissionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	perm, err := r.client.GetObjectPermission(ctx, data.DatabaseName.ValueString(), data.SchemaName.ValueString(), data.ObjectName.ValueString(),
		data.ColumnName.ValueString(), data.PrincipalName.ValueString(), data.Permission.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read the object permission", err.Error())
		return
	}
	// Gone, or turned into a DENY: either way the grant no longer exists.
	if perm == nil || perm.State == "D" {
		resp.State.RemoveResource(ctx)
		return
	}

	data.WithGrantOption = types.BoolValue(perm.WithGrantOption)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DatabaseObjectPermissionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Update Not Supported", "Every attribute of an object permission forces a new resource.")
}

func (r *DatabaseObjectPermissionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data DatabaseObjectPermissionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.RevokeObjectPermission(ctx, data.DatabaseName.ValueString(), data.SchemaName.ValueString(), data.ObjectName.ValueString(),
		data.ColumnName.ValueString(), data.PrincipalName.ValueString(), data.Permission.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to revoke the object permission", err.Error())
	}
}

func (r *DatabaseObjectPermissionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	database, schemaName, objectName, columnName, principal, permission, err := parseObjectPermissionID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}

	perm, err := r.client.GetObjectPermission(ctx, database, schemaName, objectName, columnName, principal, permission)
	if err != nil {
		resp.Diagnostics.AddError("Failed to import the object permission", err.Error())
		return
	}
	if perm == nil || perm.State == "D" {
		resp.Diagnostics.AddError("Object permission not found", fmt.Sprintf("No GRANT of %s found for %q on %s.%s in %q.", permission, principal, schemaName, objectName, database))
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), objectPermissionID(database, schemaName, objectName, columnName, principal, permission))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("database_name"), database)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("schema_name"), perm.SchemaName)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("object_name"), perm.ObjectName)...)
	if perm.ColumnName != "" {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("column_name"), perm.ColumnName)...)
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("principal_name"), perm.PrincipalName)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("permission"), strings.ToUpper(permission))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("with_grant_option"), perm.WithGrantOption)...)
}
