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
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/muecahit94/terraform-provider-mssql/internal/mssql"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &DatabaseResource{}
var _ resource.ResourceWithImportState = &DatabaseResource{}

// NewDatabaseResource creates a new database resource.
func NewDatabaseResource() resource.Resource {
	return &DatabaseResource{}
}

// DatabaseResource defines the resource implementation.
type DatabaseResource struct {
	client *mssql.Client
}

// DatabaseResourceModel describes the resource data model.
type DatabaseResourceModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

// Metadata returns the resource type name.
func (r *DatabaseResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_database"
}

// Schema defines the schema for the resource.
func (r *DatabaseResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a SQL Server database.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The database ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the database.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *DatabaseResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// Create creates the resource and sets the initial Terraform state.
func (r *DatabaseResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data DatabaseResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating database", map[string]interface{}{
		"name": data.Name.ValueString(),
	})

	db, err := r.client.CreateDatabase(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to create database", err.Error())
		return
	}

	data.ID = types.StringValue(strconv.Itoa(db.ID))
	data.Name = types.StringValue(db.Name)

	tflog.Debug(ctx, "Created database", map[string]interface{}{
		"id":   data.ID.ValueString(),
		"name": data.Name.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *DatabaseResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data DatabaseResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	var db *mssql.Database
	var err error

	// Try to find by ID first
	id, parseErr := strconv.Atoi(data.ID.ValueString())
	if parseErr == nil {
		db, err = r.client.GetDatabaseByID(ctx, id)
		if err != nil {
			resp.Diagnostics.AddError("Failed to read database", err.Error())
			return
		}
	}

	// If not found by ID, try to find by name (handles ID changes)
	if db == nil && !data.Name.IsNull() {
		db, err = r.client.GetDatabase(ctx, data.Name.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Failed to read database", err.Error())
			return
		}
	}

	// Resource no longer exists
	if db == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	// Update state with current values (including potentially changed ID)
	data.ID = types.StringValue(strconv.Itoa(db.ID))
	data.Name = types.StringValue(db.Name)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *DatabaseResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Database name changes require replacement, so this should not be called
	resp.Diagnostics.AddError(
		"Update Not Supported",
		"Database resources do not support updates. Changes to the name require replacement.",
	)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *DatabaseResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data DatabaseResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting database", map[string]interface{}{
		"name": data.Name.ValueString(),
	})

	err := r.client.DropDatabase(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete database", err.Error())
		return
	}

	tflog.Debug(ctx, "Deleted database", map[string]interface{}{
		"name": data.Name.ValueString(),
	})
}

// ImportState imports an existing resource into Terraform.
func (r *DatabaseResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import by name
	db, err := r.client.GetDatabase(ctx, req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to import database", err.Error())
		return
	}

	if db == nil {
		resp.Diagnostics.AddError("Database not found", fmt.Sprintf("Database '%s' not found", req.ID))
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), strconv.Itoa(db.ID))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), db.Name)...)
}
