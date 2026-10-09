// Copyright (c) 2024 muecahit94
// SPDX-License-Identifier: MIT

package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/muecahit94/terraform-provider-mssql/internal/mssql"
)

var _ resource.Resource = &LinkedServerLoginResource{}
var _ resource.ResourceWithImportState = &LinkedServerLoginResource{}
var _ resource.ResourceWithValidateConfig = &LinkedServerLoginResource{}

func NewLinkedServerLoginResource() resource.Resource {
	return &LinkedServerLoginResource{}
}

type LinkedServerLoginResource struct {
	client *mssql.Client
}

type LinkedServerLoginResourceModel struct {
	ID                types.String `tfsdk:"id"`
	ServerName        types.String `tfsdk:"server_name"`
	LocalLogin        types.String `tfsdk:"local_login"`
	UseSelf           types.Bool   `tfsdk:"use_self"`
	RemoteUser        types.String `tfsdk:"remote_user"`
	Password          types.String `tfsdk:"password"`
	PasswordWO        types.String `tfsdk:"password_wo"`
	PasswordWOVersion types.String `tfsdk:"password_wo_version"`
}

// linkedServerLoginID builds the resource ID. An empty local login stands for all local logins.
func linkedServerLoginID(serverName, localLogin string) string {
	return serverName + "/" + localLogin
}

// parseLinkedServerLoginID splits an ID built by linkedServerLoginID.
func parseLinkedServerLoginID(id string) (serverName, localLogin string, ok bool) {
	serverName, localLogin, ok = strings.Cut(id, "/")
	if !ok || serverName == "" {
		return "", "", false
	}
	return serverName, localLogin, true
}

// validateLinkedServerLogin checks the combination of login mapping attributes.
// Unknown counts as configured: an ephemeral value assigned to password_wo is
// unknown until apply.
func validateLinkedServerLogin(data LinkedServerLoginResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	if data.UseSelf.IsUnknown() {
		return diags
	}

	passwordSet := !data.Password.IsNull()
	writeOnlySet := !data.PasswordWO.IsNull()
	remoteUserSet := !data.RemoteUser.IsNull()

	if data.UseSelf.ValueBool() {
		if remoteUserSet || passwordSet || writeOnlySet {
			diags.AddAttributeError(
				path.Root("use_self"),
				"Conflicting login mapping attributes",
				"`remote_user`, `password` and `password_wo` cannot be set when `use_self` is `true`.",
			)
		}
		return diags
	}

	if !remoteUserSet {
		diags.AddAttributeError(
			path.Root("remote_user"),
			"Missing remote user",
			"`remote_user` is required unless `use_self` is `true`.",
		)
	}

	switch {
	case passwordSet && writeOnlySet:
		diags.AddAttributeError(
			path.Root("password_wo"),
			"Conflicting password attributes",
			"Only one of `password` and `password_wo` can be set.",
		)
	case !passwordSet && !writeOnlySet:
		diags.AddAttributeError(
			path.Root("password"),
			"Missing password",
			"One of `password` or `password_wo` must be set unless `use_self` is `true`. Use `password_wo` "+
				"to keep the password out of the plan and state files; it requires Terraform 1.11 or later.",
		)
	}

	if !writeOnlySet && !data.PasswordWOVersion.IsNull() {
		diags.AddAttributeError(
			path.Root("password_wo_version"),
			"Missing write-only password",
			"`password_wo_version` only has an effect together with `password_wo`.",
		)
	}

	return diags
}

// linkedServerLoginPassword returns the remote password. Write-only values are
// stripped from the plan, so password_wo has to be read from the config.
func linkedServerLoginPassword(plan, config LinkedServerLoginResourceModel) string {
	if !config.PasswordWO.IsNull() {
		return config.PasswordWO.ValueString()
	}
	return plan.Password.ValueString()
}

func (r *LinkedServerLoginResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_linked_server_login"
}

func (r *LinkedServerLoginResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the mapping of a local login to a remote login on a SQL Server linked server.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The resource ID, in the form `server_name/local_login`. The local login is empty for a mapping that applies to all local logins.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"server_name": schema.StringAttribute{
				Description: "The name of the linked server. Changing this forces a new resource to be created.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"local_login": schema.StringAttribute{
				Description: "The local login the mapping applies to. If omitted, the mapping applies to all local logins. " +
					"Changing this forces a new resource to be created.",
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"use_self": schema.BoolAttribute{
				Description: "Whether the local login connects with its own credentials instead of a remote login. " +
					"Defaults to `false`. When `true`, `remote_user` and the password attributes cannot be set.",
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},
			"remote_user": schema.StringAttribute{
				Description: "The remote login to connect as. Required unless `use_self` is `true`.",
				Optional:    true,
			},
			"password": schema.StringAttribute{
				Description: "The password of the remote login. Persisted in the plan and state files; " +
					"use `password_wo` instead to avoid that. Exactly one of `password` and `password_wo` " +
					"must be set unless `use_self` is `true`.",
				Optional:  true,
				Sensitive: true,
			},
			"password_wo": schema.StringAttribute{
				Description: "The password of the remote login, as a write-only attribute. Accepts ephemeral values " +
					"and is written to neither the plan nor the state file. Requires Terraform 1.11 or later. " +
					"Because Terraform has no stored value to compare against, changing this alone does not update " +
					"the mapping; change `password_wo_version` to apply a new password.",
				Optional:  true,
				Sensitive: true,
				WriteOnly: true,
			},
			"password_wo_version": schema.StringAttribute{
				Description: "An arbitrary token whose change re-applies the mapping with the current `password_wo` " +
					"value. Only valid together with `password_wo`. Without it, a rotated `password_wo` is never applied.",
				Optional: true,
			},
		},
	}
}

func (r *LinkedServerLoginResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *LinkedServerLoginResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data LinkedServerLoginResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(validateLinkedServerLogin(data)...)
}

// apply writes the mapping to the server. sp_addlinkedsrvlogin replaces an existing
// mapping for the same local login, so it serves both Create and Update.
func (r *LinkedServerLoginResource) apply(ctx context.Context, plan, config LinkedServerLoginResourceModel) (*mssql.LinkedServerLogin, error) {
	return r.client.SetLinkedServerLogin(ctx, mssql.SetLinkedServerLoginOptions{
		ServerName: plan.ServerName.ValueString(),
		LocalLogin: plan.LocalLogin.ValueString(),
		UseSelf:    plan.UseSelf.ValueBool(),
		RemoteUser: plan.RemoteUser.ValueString(),
		Password:   linkedServerLoginPassword(plan, config),
	})
}

func (r *LinkedServerLoginResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data, config LinkedServerLoginResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	login, err := r.apply(ctx, data, config)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create linked server login", err.Error())
		return
	}
	if login == nil {
		resp.Diagnostics.AddError("Failed to create linked server login", "The login mapping was not found after creation.")
		return
	}

	data.ID = types.StringValue(linkedServerLoginID(data.ServerName.ValueString(), data.LocalLogin.ValueString()))
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *LinkedServerLoginResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data LinkedServerLoginResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	login, err := r.client.GetLinkedServerLogin(ctx, data.ServerName.ValueString(), data.LocalLogin.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read linked server login", err.Error())
		return
	}
	if login == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	data.UseSelf = types.BoolValue(login.UsesSelf)
	if login.RemoteUser == "" {
		data.RemoteUser = types.StringNull()
	} else {
		data.RemoteUser = types.StringValue(login.RemoteUser)
	}
	// SQL Server never returns the remote password, so a changed password cannot be detected.
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *LinkedServerLoginResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data, config LinkedServerLoginResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	login, err := r.apply(ctx, data, config)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update linked server login", err.Error())
		return
	}
	if login == nil {
		resp.Diagnostics.AddError("Failed to update linked server login", "The login mapping was not found after the update.")
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *LinkedServerLoginResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data LinkedServerLoginResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DropLinkedServerLogin(ctx, data.ServerName.ValueString(), data.LocalLogin.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to delete linked server login", err.Error())
		return
	}
}

func (r *LinkedServerLoginResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	serverName, localLogin, ok := parseLinkedServerLoginID(req.ID)
	if !ok {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			fmt.Sprintf("Expected `server_name/local_login` (leave `local_login` empty for all logins), got '%s'.", req.ID),
		)
		return
	}

	login, err := r.client.GetLinkedServerLogin(ctx, serverName, localLogin)
	if err != nil {
		resp.Diagnostics.AddError("Failed to import linked server login", err.Error())
		return
	}
	if login == nil {
		resp.Diagnostics.AddError("Linked server login not found", fmt.Sprintf("No login mapping found for '%s'.", req.ID))
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("server_name"), login.ServerName)...)
	if login.LocalLogin != "" {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("local_login"), login.LocalLogin)...)
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("use_self"), login.UsesSelf)...)
	if login.RemoteUser != "" {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("remote_user"), login.RemoteUser)...)
	}
}
