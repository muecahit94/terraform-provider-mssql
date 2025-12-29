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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/muecahit94/terraform-provider-mssql/internal/mssql"
)

var _ resource.Resource = &SQLLoginResource{}
var _ resource.ResourceWithImportState = &SQLLoginResource{}

func NewSQLLoginResource() resource.Resource {
	return &SQLLoginResource{}
}

type SQLLoginResource struct {
	client *mssql.Client
}

type SQLLoginResourceModel struct {
	ID                     types.String `tfsdk:"id"`
	Name                   types.String `tfsdk:"name"`
	Password               types.String `tfsdk:"password"`
	DefaultDatabase        types.String `tfsdk:"default_database"`
	DefaultLanguage        types.String `tfsdk:"default_language"`
	CheckExpirationEnabled types.Bool   `tfsdk:"check_expiration_enabled"`
	CheckPolicyEnabled     types.Bool   `tfsdk:"check_policy_enabled"`
	IsDisabled             types.Bool   `tfsdk:"is_disabled"`
}

func (r *SQLLoginResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sql_login"
}

func (r *SQLLoginResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a SQL Server login.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The login principal ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the login.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"password": schema.StringAttribute{
				Description: "The password for the login.",
				Required:    true,
				Sensitive:   true,
			},
			"default_database": schema.StringAttribute{
				Description: "The default database for the login.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("master"),
			},
			"default_language": schema.StringAttribute{
				Description: "The default language for the login.",
				Optional:    true,
				Computed:    true,
			},
			"check_expiration_enabled": schema.BoolAttribute{
				Description: "Whether password expiration is checked.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"check_policy_enabled": schema.BoolAttribute{
				Description: "Whether password policy is enforced.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"is_disabled": schema.BoolAttribute{
				Description: "Whether the login is disabled.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
		},
	}
}

func (r *SQLLoginResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *SQLLoginResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data SQLLoginResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating SQL login", map[string]interface{}{
		"name": data.Name.ValueString(),
	})

	opts := mssql.CreateSQLLoginOptions{
		Name:                   data.Name.ValueString(),
		Password:               data.Password.ValueString(),
		DefaultDatabase:        data.DefaultDatabase.ValueString(),
		DefaultLanguage:        data.DefaultLanguage.ValueString(),
		CheckExpirationEnabled: data.CheckExpirationEnabled.ValueBool(),
		CheckPolicyEnabled:     data.CheckPolicyEnabled.ValueBool(),
	}

	login, err := r.client.CreateSQLLogin(ctx, opts)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create SQL login", err.Error())
		return
	}

	// Handle disabled state
	if data.IsDisabled.ValueBool() {
		disabled := true
		_, err := r.client.UpdateSQLLogin(ctx, mssql.UpdateSQLLoginOptions{
			Name:       data.Name.ValueString(),
			IsDisabled: &disabled,
		})
		if err != nil {
			resp.Diagnostics.AddError("Failed to disable SQL login", err.Error())
			return
		}
	}

	data.ID = types.StringValue(strconv.Itoa(login.PrincipalID))
	data.DefaultLanguage = types.StringValue(login.DefaultLanguageName)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SQLLoginResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data SQLLoginResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	var login *mssql.SQLLogin
	var err error

	// Try to find by ID first
	id, parseErr := strconv.Atoi(data.ID.ValueString())
	if parseErr == nil {
		login, err = r.client.GetSQLLoginByID(ctx, id)
		if err != nil {
			resp.Diagnostics.AddError("Failed to read SQL login", err.Error())
			return
		}
	}

	// If not found by ID, try to find by name (handles ID changes)
	if login == nil && !data.Name.IsNull() {
		login, err = r.client.GetSQLLogin(ctx, data.Name.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Failed to read SQL login", err.Error())
			return
		}
	}

	if login == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	// Update state with current values
	data.ID = types.StringValue(strconv.Itoa(login.PrincipalID))
	data.Name = types.StringValue(login.Name)
	data.DefaultDatabase = types.StringValue(login.DefaultDatabaseName)
	data.DefaultLanguage = types.StringValue(login.DefaultLanguageName)
	data.CheckExpirationEnabled = types.BoolValue(login.CheckExpirationEnabled)
	data.CheckPolicyEnabled = types.BoolValue(login.CheckPolicyEnabled)
	data.IsDisabled = types.BoolValue(login.IsDisabled)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SQLLoginResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data SQLLoginResourceModel
	var state SQLLoginResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating SQL login", map[string]interface{}{
		"name": data.Name.ValueString(),
	})

	opts := mssql.UpdateSQLLoginOptions{
		Name: data.Name.ValueString(),
	}

	// Check what changed
	if !data.Password.Equal(state.Password) {
		password := data.Password.ValueString()
		opts.Password = &password
	}
	if !data.DefaultDatabase.Equal(state.DefaultDatabase) {
		db := data.DefaultDatabase.ValueString()
		opts.DefaultDatabase = &db
	}
	if !data.DefaultLanguage.Equal(state.DefaultLanguage) {
		lang := data.DefaultLanguage.ValueString()
		opts.DefaultLanguage = &lang
	}
	if !data.CheckExpirationEnabled.Equal(state.CheckExpirationEnabled) {
		exp := data.CheckExpirationEnabled.ValueBool()
		opts.CheckExpirationEnabled = &exp
	}
	if !data.CheckPolicyEnabled.Equal(state.CheckPolicyEnabled) {
		policy := data.CheckPolicyEnabled.ValueBool()
		opts.CheckPolicyEnabled = &policy
	}
	if !data.IsDisabled.Equal(state.IsDisabled) {
		disabled := data.IsDisabled.ValueBool()
		opts.IsDisabled = &disabled
	}

	_, err := r.client.UpdateSQLLogin(ctx, opts)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update SQL login", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SQLLoginResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data SQLLoginResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting SQL login", map[string]interface{}{
		"name": data.Name.ValueString(),
	})

	err := r.client.DropSQLLogin(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete SQL login", err.Error())
		return
	}
}

func (r *SQLLoginResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	login, err := r.client.GetSQLLogin(ctx, req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to import SQL login", err.Error())
		return
	}

	if login == nil {
		resp.Diagnostics.AddError("SQL login not found", fmt.Sprintf("Login '%s' not found", req.ID))
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), strconv.Itoa(login.PrincipalID))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), login.Name)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("password"), "")...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("default_database"), login.DefaultDatabaseName)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("default_language"), login.DefaultLanguageName)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("check_expiration_enabled"), login.CheckExpirationEnabled)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("check_policy_enabled"), login.CheckPolicyEnabled)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("is_disabled"), login.IsDisabled)...)
}
