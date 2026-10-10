// Copyright (c) 2024 muecahit94
// SPDX-License-Identifier: MIT

package provider

import (
	"context"
	"fmt"
	"regexp"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/muecahit94/terraform-provider-mssql/internal/mssql"
)

var _ resource.Resource = &WindowsLoginResource{}
var _ resource.ResourceWithImportState = &WindowsLoginResource{}
var _ resource.ResourceWithValidateConfig = &WindowsLoginResource{}

// windowsLoginNamePattern matches DOMAIN\name, the only form CREATE LOGIN ... FROM WINDOWS accepts.
var windowsLoginNamePattern = regexp.MustCompile(`^[^\\@]+\\[^\\@]+$`)

// ValidateConfig rejects names that SQL Server would refuse at apply time.
func (r *WindowsLoginResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data WindowsLoginResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() || data.Name.IsNull() || data.Name.IsUnknown() {
		return
	}
	if !windowsLoginNamePattern.MatchString(data.Name.ValueString()) {
		resp.Diagnostics.AddAttributeError(path.Root("name"), "Invalid Windows login name",
			fmt.Sprintf("%q must be written as DOMAIN\\name; SQL Server does not accept a UPN (name@domain) for a Windows login.", data.Name.ValueString()))
	}
}

// NewWindowsLoginResource creates a new Windows login resource.
func NewWindowsLoginResource() resource.Resource {
	return &WindowsLoginResource{}
}

// WindowsLoginResource manages the login of a Windows or Active Directory user or group.
type WindowsLoginResource struct {
	client *mssql.Client
}

// WindowsLoginResourceModel describes the resource data model.
type WindowsLoginResourceModel struct {
	ID              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	Type            types.String `tfsdk:"type"`
	DefaultDatabase types.String `tfsdk:"default_database"`
	DefaultLanguage types.String `tfsdk:"default_language"`
	IsDisabled      types.Bool   `tfsdk:"is_disabled"`
}

func (r *WindowsLoginResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_windows_login"
}

func (r *WindowsLoginResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the login of a Windows or Active Directory user or group (`CREATE LOGIN ... FROM WINDOWS`).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The login principal ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the Windows user or group, as `DOMAIN\\name`. " +
					"It must exist in Windows or Active Directory; a UPN (`name@domain`) is not accepted by SQL Server. " +
					"Changing this forces a new resource to be created.",
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"type": schema.StringAttribute{
				Description: "`WINDOWS_LOGIN` for a user, `WINDOWS_GROUP` for a group.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"default_database": schema.StringAttribute{
				Description: "The default database of the login. Defaults to the value SQL Server assigns (`master`).",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"default_language": schema.StringAttribute{
				Description: "The default language of the login. Defaults to the value SQL Server assigns.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"is_disabled": schema.BoolAttribute{
				Description: "Whether the login is disabled. Defaults to `false`. SQL Server cannot disable the login of a Windows group, only of a user.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
		},
	}
}

func (r *WindowsLoginResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// applyWindowsLogin copies server values into the model.
func applyWindowsLogin(data *WindowsLoginResourceModel, login *mssql.WindowsLogin) {
	data.ID = types.StringValue(strconv.Itoa(login.PrincipalID))
	// SQL Server matches Windows names case-insensitively and may store another spelling.
	data.Name = keepCase(data.Name, login.Name)
	data.Type = types.StringValue(login.Type)
	data.DefaultDatabase = types.StringValue(login.DefaultDatabase)
	data.DefaultLanguage = types.StringValue(login.DefaultLanguage)
	data.IsDisabled = types.BoolValue(login.IsDisabled)
}

func (r *WindowsLoginResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data WindowsLoginResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	login, err := r.client.CreateWindowsLogin(ctx, mssql.CreateWindowsLoginOptions{
		Name:            data.Name.ValueString(),
		DefaultDatabase: data.DefaultDatabase.ValueString(),
		DefaultLanguage: data.DefaultLanguage.ValueString(),
		Disabled:        data.IsDisabled.ValueBool(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create Windows login", err.Error())
		return
	}
	if login == nil {
		resp.Diagnostics.AddError("Failed to create Windows login", "The login was not found after creation.")
		return
	}

	applyWindowsLogin(&data, login)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WindowsLoginResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data WindowsLoginResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	login, err := r.client.GetWindowsLogin(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read Windows login", err.Error())
		return
	}
	if login == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	applyWindowsLogin(&data, login)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WindowsLoginResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state WindowsLoginResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var opts mssql.UpdateWindowsLoginOptions
	if !plan.DefaultDatabase.IsUnknown() && !plan.DefaultDatabase.Equal(state.DefaultDatabase) {
		v := plan.DefaultDatabase.ValueString()
		opts.DefaultDatabase = &v
	}
	if !plan.DefaultLanguage.IsUnknown() && !plan.DefaultLanguage.Equal(state.DefaultLanguage) {
		v := plan.DefaultLanguage.ValueString()
		opts.DefaultLanguage = &v
	}
	if !plan.IsDisabled.IsUnknown() && !plan.IsDisabled.Equal(state.IsDisabled) {
		v := plan.IsDisabled.ValueBool()
		opts.Disabled = &v
	}

	if err := r.client.UpdateWindowsLogin(ctx, state.Name.ValueString(), opts); err != nil {
		resp.Diagnostics.AddError("Failed to update Windows login", err.Error())
		return
	}

	login, err := r.client.GetWindowsLogin(ctx, state.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read Windows login", err.Error())
		return
	}
	if login == nil {
		resp.Diagnostics.AddError("Failed to update Windows login", "The login was not found after the update.")
		return
	}

	applyWindowsLogin(&plan, login)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *WindowsLoginResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data WindowsLoginResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DropWindowsLogin(ctx, data.Name.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to delete Windows login", err.Error())
	}
}

func (r *WindowsLoginResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	login, err := r.client.GetWindowsLogin(ctx, req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to import Windows login", err.Error())
		return
	}
	if login == nil {
		resp.Diagnostics.AddError("Windows login not found", fmt.Sprintf("No Windows login named %q.", req.ID))
		return
	}

	var data WindowsLoginResourceModel
	applyWindowsLogin(&data, login)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
