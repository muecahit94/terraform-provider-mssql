// Copyright (c) 2024 muecahit94
// SPDX-License-Identifier: MIT

package provider

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/muecahit94/terraform-provider-mssql/internal/mssql"
)

var _ resource.Resource = &SQLUserResource{}
var _ resource.ResourceWithImportState = &SQLUserResource{}

func NewSQLUserResource() resource.Resource {
	return &SQLUserResource{}
}

type SQLUserResource struct {
	client *mssql.Client
}

type SQLUserResourceModel struct {
	ID            types.String `tfsdk:"id"`
	DatabaseName  types.String `tfsdk:"database_name"`
	Name          types.String `tfsdk:"name"`
	LoginName     types.String `tfsdk:"login_name"`
	DefaultSchema types.String `tfsdk:"default_schema"`
	Roles         types.Set    `tfsdk:"roles"`
}

func (r *SQLUserResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sql_user"
}

func (r *SQLUserResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a SQL Server database user mapped to a login.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The user ID in format 'database_id/principal_id'.",
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
				Description: "The name of the user.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"login_name": schema.StringAttribute{
				Description: "The name of the login to map this user to.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"default_schema": schema.StringAttribute{
				Description: "The default schema for the user.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("dbo"),
			},
			"roles": schema.SetAttribute{
				Description: "List of database roles to assign to this user.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (r *SQLUserResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*mssql.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *mssql.Client, got: %T.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *SQLUserResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data SQLUserResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating SQL user", map[string]interface{}{
		"database": data.DatabaseName.ValueString(),
		"name":     data.Name.ValueString(),
	})

	opts := mssql.CreateSQLUserOptions{
		DatabaseName:  data.DatabaseName.ValueString(),
		UserName:      data.Name.ValueString(),
		LoginName:     data.LoginName.ValueString(),
		DefaultSchema: data.DefaultSchema.ValueString(),
	}

	user, err := r.client.CreateSQLUser(ctx, opts)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create SQL user", err.Error())
		return
	}

	// Assign roles if specified
	var roles []string
	if !data.Roles.IsNull() && !data.Roles.IsUnknown() {
		resp.Diagnostics.Append(data.Roles.ElementsAs(ctx, &roles, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		for _, role := range roles {
			err := r.client.AddDatabaseRoleMember(ctx, data.DatabaseName.ValueString(), role, data.Name.ValueString())
			if err != nil {
				resp.Diagnostics.AddError("Failed to assign role", fmt.Sprintf("Failed to add user to role '%s': %s", role, err.Error()))
				return
			}
		}
	}

	data.ID = types.StringValue(fmt.Sprintf("%d/%d", user.DatabaseID, user.PrincipalID))
	data.DefaultSchema = types.StringValue(user.DefaultSchemaName)

	// Set roles in state
	if len(roles) > 0 {
		roleValues := make([]attr.Value, len(roles))
		for i, role := range roles {
			roleValues[i] = types.StringValue(role)
		}
		data.Roles, _ = types.SetValue(types.StringType, roleValues)
	} else {
		data.Roles, _ = types.SetValue(types.StringType, []attr.Value{})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SQLUserResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data SQLUserResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Always lookup by name - handles ID changes gracefully
	user, err := r.client.GetUser(ctx, data.DatabaseName.ValueString(), data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read SQL user", err.Error())
		return
	}

	if user == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	// Update state with current values (including potentially changed ID)
	data.ID = types.StringValue(fmt.Sprintf("%d/%d", user.DatabaseID, user.PrincipalID))
	data.DefaultSchema = types.StringValue(user.DefaultSchemaName)
	data.LoginName = types.StringValue(user.LoginName)

	// Read user's roles
	roles, err := r.client.GetUserRoles(ctx, data.DatabaseName.ValueString(), data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read user roles", err.Error())
		return
	}
	roleValues := make([]attr.Value, len(roles))
	for i, role := range roles {
		roleValues[i] = types.StringValue(role)
	}
	data.Roles, _ = types.SetValue(types.StringType, roleValues)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SQLUserResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data SQLUserResourceModel
	var state SQLUserResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating SQL user", map[string]interface{}{
		"database": data.DatabaseName.ValueString(),
		"name":     data.Name.ValueString(),
	})

	opts := mssql.UpdateSQLUserOptions{
		DatabaseName: data.DatabaseName.ValueString(),
		UserName:     data.Name.ValueString(),
	}

	if !data.DefaultSchema.Equal(state.DefaultSchema) {
		schema := data.DefaultSchema.ValueString()
		opts.DefaultSchema = &schema
	}

	_, err := r.client.UpdateSQLUser(ctx, opts)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update SQL user", err.Error())
		return
	}

	// Update roles if changed
	if !data.Roles.Equal(state.Roles) {
		var desiredRoles, currentRoles []string
		resp.Diagnostics.Append(data.Roles.ElementsAs(ctx, &desiredRoles, false)...)
		resp.Diagnostics.Append(state.Roles.ElementsAs(ctx, &currentRoles, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		// Find roles to add and remove
		currentSet := make(map[string]bool)
		for _, role := range currentRoles {
			currentSet[role] = true
		}
		desiredSet := make(map[string]bool)
		for _, role := range desiredRoles {
			desiredSet[role] = true
		}

		// Add new roles
		for _, role := range desiredRoles {
			if !currentSet[role] {
				err := r.client.AddDatabaseRoleMember(ctx, data.DatabaseName.ValueString(), role, data.Name.ValueString())
				if err != nil {
					resp.Diagnostics.AddError("Failed to add role", fmt.Sprintf("Failed to add user to role '%s': %s", role, err.Error()))
					return
				}
			}
		}

		// Remove old roles
		for _, role := range currentRoles {
			if !desiredSet[role] {
				err := r.client.RemoveDatabaseRoleMember(ctx, data.DatabaseName.ValueString(), role, data.Name.ValueString())
				if err != nil {
					resp.Diagnostics.AddError("Failed to remove role", fmt.Sprintf("Failed to remove user from role '%s': %s", role, err.Error()))
					return
				}
			}
		}

		// Update state with sorted roles
		sort.Strings(desiredRoles)
		roleValues := make([]attr.Value, len(desiredRoles))
		for i, role := range desiredRoles {
			roleValues[i] = types.StringValue(role)
		}
		data.Roles, _ = types.SetValue(types.StringType, roleValues)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SQLUserResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data SQLUserResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting SQL user", map[string]interface{}{
		"database": data.DatabaseName.ValueString(),
		"name":     data.Name.ValueString(),
	})

	err := r.client.DropUser(ctx, data.DatabaseName.ValueString(), data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete SQL user", err.Error())
		return
	}
}

func (r *SQLUserResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: database_name/user_name
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"Import ID must be in format 'database_name/user_name'",
		)
		return
	}

	databaseName := parts[0]
	userName := parts[1]

	user, err := r.client.GetUser(ctx, databaseName, userName)
	if err != nil {
		resp.Diagnostics.AddError("Failed to import SQL user", err.Error())
		return
	}

	if user == nil {
		resp.Diagnostics.AddError("SQL user not found", fmt.Sprintf("User '%s' not found in database '%s'", userName, databaseName))
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), fmt.Sprintf("%d/%d", user.DatabaseID, user.PrincipalID))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("database_name"), databaseName)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), user.Name)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("login_name"), user.LoginName)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("default_schema"), user.DefaultSchemaName)...)
}
