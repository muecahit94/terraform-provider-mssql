// Copyright (c) 2024 muecahit94
// SPDX-License-Identifier: MIT

package provider

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/muecahit94/terraform-provider-mssql/internal/mssql"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &DatabaseResource{}
var _ resource.ResourceWithImportState = &DatabaseResource{}
var _ resource.ResourceWithValidateConfig = &DatabaseResource{}
var _ resource.ResourceWithModifyPlan = &DatabaseResource{}

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
	ID                 types.String `tfsdk:"id"`
	Name               types.String `tfsdk:"name"`
	Collation          types.String `tfsdk:"collation"`
	CompatibilityLevel types.Int64  `tfsdk:"compatibility_level"`
	RecoveryModel      types.String `tfsdk:"recovery_model"`
}

// keepCase returns the configured value when it only differs from the server value by case.
// SQL Server stores the canonical spelling of a collation, but accepts any case, and
// Terraform requires the applied value to equal the planned one.
func keepCase(configured types.String, actual string) types.String {
	if !configured.IsNull() && !configured.IsUnknown() && strings.EqualFold(configured.ValueString(), actual) {
		return configured
	}
	return types.StringValue(actual)
}

// applyDatabase copies server values into the model.
func applyDatabase(data *DatabaseResourceModel, db *mssql.Database) {
	data.ID = types.StringValue(strconv.Itoa(db.ID))
	data.Name = types.StringValue(db.Name)
	data.Collation = keepCase(data.Collation, db.Collation)
	data.CompatibilityLevel = types.Int64Value(int64(db.CompatibilityLevel))
	data.RecoveryModel = types.StringValue(db.RecoveryModel)
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
			"collation": schema.StringAttribute{
				Description: "The collation of the database, for example `SQL_Latin1_General_CP1_CI_AS`. " +
					"Set when the database is created; defaults to the collation of the server. " +
					"SQL Server does not change the collation of existing columns when the database collation " +
					"changes, so changing it on an existing database is rejected instead of replacing (and " +
					"dropping) the database.",
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"compatibility_level": schema.Int64Attribute{
				Description: "The compatibility level of the database, for example `150` or `160`. " +
					"Defaults to the level of the server's `model` database. Can be changed in place.",
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"recovery_model": schema.StringAttribute{
				Description: "The recovery model of the database: `FULL`, `SIMPLE` or `BULK_LOGGED`. " +
					"Defaults to the model of the server's `model` database. Can be changed in place. " +
					"Switching to `FULL` or `BULK_LOGGED` does not start the log backup chain until a full backup is taken.",
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// ValidateConfig checks the values that SQL Server only rejects at apply time.
func (r *DatabaseResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data DatabaseResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !data.RecoveryModel.IsNull() && !data.RecoveryModel.IsUnknown() &&
		!slices.Contains(mssql.RecoveryModels, data.RecoveryModel.ValueString()) {
		resp.Diagnostics.AddAttributeError(
			path.Root("recovery_model"),
			"Invalid recovery model",
			fmt.Sprintf("`recovery_model` must be one of %s (upper case), got %q.",
				strings.Join(mssql.RecoveryModels, ", "), data.RecoveryModel.ValueString()),
		)
	}

	if !data.CompatibilityLevel.IsNull() && !data.CompatibilityLevel.IsUnknown() && data.CompatibilityLevel.ValueInt64() <= 0 {
		resp.Diagnostics.AddAttributeError(
			path.Root("compatibility_level"),
			"Invalid compatibility level",
			"`compatibility_level` must be a positive number such as 150 or 160.",
		)
	}
}

// ModifyPlan rejects a change of the collation of an existing database. The collation is
// not replaced like a name, because that would drop the database and its data.
func (r *DatabaseResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}

	var plan, state DatabaseResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.Collation.IsNull() || plan.Collation.IsUnknown() || state.Collation.IsNull() || state.Collation.IsUnknown() {
		return
	}
	if !strings.EqualFold(plan.Collation.ValueString(), state.Collation.ValueString()) {
		resp.Diagnostics.AddAttributeError(
			path.Root("collation"),
			"Collation of an existing database cannot be changed",
			fmt.Sprintf("Database %q has collation %q, but %q is configured. The collation can only be set when "+
				"the database is created: ALTER DATABASE does not change the collation of existing columns, and replacing "+
				"the database would drop its data. Set `collation` to %q, or remove it from the configuration.",
				state.Name.ValueString(), state.Collation.ValueString(), plan.Collation.ValueString(), state.Collation.ValueString()),
		)
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

	db, err := r.client.CreateDatabase(ctx, mssql.CreateDatabaseOptions{
		Name:               data.Name.ValueString(),
		Collation:          data.Collation.ValueString(),
		CompatibilityLevel: int(data.CompatibilityLevel.ValueInt64()),
		RecoveryModel:      data.RecoveryModel.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create database", err.Error())
		return
	}
	if db == nil {
		resp.Diagnostics.AddError("Failed to create database", "The database was not found after creation.")
		return
	}

	applyDatabase(&data, db)

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
	applyDatabase(&data, db)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update applies the settings that SQL Server can change in place.
func (r *DatabaseResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state DatabaseResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := state.Name.ValueString()

	if !plan.CompatibilityLevel.IsUnknown() && !plan.CompatibilityLevel.Equal(state.CompatibilityLevel) {
		if err := r.client.SetDatabaseCompatibilityLevel(ctx, name, int(plan.CompatibilityLevel.ValueInt64())); err != nil {
			resp.Diagnostics.AddError("Failed to update database", err.Error())
			return
		}
	}
	if !plan.RecoveryModel.IsUnknown() && !plan.RecoveryModel.Equal(state.RecoveryModel) {
		if err := r.client.SetDatabaseRecoveryModel(ctx, name, plan.RecoveryModel.ValueString()); err != nil {
			resp.Diagnostics.AddError("Failed to update database", err.Error())
			return
		}
	}

	db, err := r.client.GetDatabase(ctx, name)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read database", err.Error())
		return
	}
	if db == nil {
		resp.Diagnostics.AddError("Failed to update database", "The database was not found after the update.")
		return
	}

	applyDatabase(&plan, db)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
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
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("collation"), db.Collation)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("compatibility_level"), int64(db.CompatibilityLevel))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("recovery_model"), db.RecoveryModel)...)
}
