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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
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

	AutoClose             types.Bool   `tfsdk:"auto_close"`
	AutoShrink            types.Bool   `tfsdk:"auto_shrink"`
	PageVerify            types.String `tfsdk:"page_verify"`
	SnapshotIsolation     types.Bool   `tfsdk:"snapshot_isolation"`
	ReadCommittedSnapshot types.Bool   `tfsdk:"read_committed_snapshot"`
	QueryStore            types.Bool   `tfsdk:"query_store"`
	Trustworthy           types.Bool   `tfsdk:"trustworthy"`
}

// boolSetting returns a pointer to the planned value when it is set and, for an update, differs from the state.
func boolSetting(plan, state types.Bool, update bool) *bool {
	if plan.IsNull() || plan.IsUnknown() || (update && plan.Equal(state)) {
		return nil
	}
	v := plan.ValueBool()
	return &v
}

// databaseSettings picks the settings to apply: everything that is configured on create,
// only what changed on update (state != nil).
func databaseSettings(plan DatabaseResourceModel, state *DatabaseResourceModel) mssql.DatabaseSettings {
	update := state != nil
	if state == nil {
		state = &DatabaseResourceModel{}
	}
	s := mssql.DatabaseSettings{
		AutoClose:             boolSetting(plan.AutoClose, state.AutoClose, update),
		AutoShrink:            boolSetting(plan.AutoShrink, state.AutoShrink, update),
		SnapshotIsolation:     boolSetting(plan.SnapshotIsolation, state.SnapshotIsolation, update),
		ReadCommittedSnapshot: boolSetting(plan.ReadCommittedSnapshot, state.ReadCommittedSnapshot, update),
		QueryStore:            boolSetting(plan.QueryStore, state.QueryStore, update),
		Trustworthy:           boolSetting(plan.Trustworthy, state.Trustworthy, update),
	}
	if !plan.PageVerify.IsNull() && !plan.PageVerify.IsUnknown() && (!update || !plan.PageVerify.Equal(state.PageVerify)) {
		v := plan.PageVerify.ValueString()
		s.PageVerify = &v
	}
	return s
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
	data.AutoClose = types.BoolValue(db.AutoClose)
	data.AutoShrink = types.BoolValue(db.AutoShrink)
	data.PageVerify = types.StringValue(db.PageVerify)
	data.SnapshotIsolation = types.BoolValue(db.SnapshotIsolation)
	data.ReadCommittedSnapshot = types.BoolValue(db.ReadCommittedSnapshot)
	data.QueryStore = types.BoolValue(db.QueryStore)
	data.Trustworthy = types.BoolValue(db.Trustworthy)
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
			"auto_close": databaseBoolAttribute("Whether the database closes and frees its resources when the last user disconnects (`AUTO_CLOSE`). Usually `false`."),
			"auto_shrink": databaseBoolAttribute("Whether the database files are shrunk automatically (`AUTO_SHRINK`). Usually `false`: " +
				"it fragments the indexes and costs performance."),
			"page_verify": schema.StringAttribute{
				Description: "How page corruption is detected: `CHECKSUM`, `TORN_PAGE_DETECTION` or `NONE`. " +
					"`CHECKSUM` is the default of new databases. Can be changed in place.",
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"snapshot_isolation": databaseBoolAttribute("Whether transactions can use snapshot isolation (`ALLOW_SNAPSHOT_ISOLATION`)."),
			"read_committed_snapshot": databaseBoolAttribute("Whether READ COMMITTED reads row versions instead of taking shared locks (`READ_COMMITTED_SNAPSHOT`). " +
				"Changing it needs exclusive access to the database, so it fails at once (`WITH NO_WAIT`) when other connections are open."),
			"query_store": databaseBoolAttribute("Whether Query Store records query plans and statistics (`QUERY_STORE`)."),
			"trustworthy": databaseBoolAttribute("Whether the database is trusted for access to resources outside it (`TRUSTWORTHY`). " +
				"Keep it `false` unless a signed-module alternative is impossible: it is a privilege escalation path."),
		},
	}
}

// databaseBoolAttribute builds an optional, computed boolean setting that can be changed in place.
func databaseBoolAttribute(description string) schema.BoolAttribute {
	return schema.BoolAttribute{
		Description: description + " Defaults to the value SQL Server assigns. Can be changed in place.",
		Optional:    true,
		Computed:    true,
		PlanModifiers: []planmodifier.Bool{
			boolplanmodifier.UseStateForUnknown(),
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

	if !data.PageVerify.IsNull() && !data.PageVerify.IsUnknown() &&
		!slices.Contains(mssql.PageVerifyOptions, data.PageVerify.ValueString()) {
		resp.Diagnostics.AddAttributeError(
			path.Root("page_verify"),
			"Invalid page verify option",
			fmt.Sprintf("`page_verify` must be one of %s (upper case), got %q.",
				strings.Join(mssql.PageVerifyOptions, ", "), data.PageVerify.ValueString()),
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
		Settings:           databaseSettings(data, nil),
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

	if err := r.client.ApplyDatabaseSettings(ctx, name, databaseSettings(plan, &state)); err != nil {
		resp.Diagnostics.AddError("Failed to update database", err.Error())
		return
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
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("auto_close"), db.AutoClose)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("auto_shrink"), db.AutoShrink)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("page_verify"), db.PageVerify)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("snapshot_isolation"), db.SnapshotIsolation)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("read_committed_snapshot"), db.ReadCommittedSnapshot)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("query_store"), db.QueryStore)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("trustworthy"), db.Trustworthy)...)
}
