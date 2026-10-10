// Copyright (c) 2024 muecahit94
// SPDX-License-Identifier: MIT

package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/muecahit94/terraform-provider-mssql/internal/mssql"
)

var _ resource.Resource = &SQLLoginResource{}
var _ resource.ResourceWithImportState = &SQLLoginResource{}
var _ resource.ResourceWithValidateConfig = &SQLLoginResource{}
var _ resource.ResourceWithModifyPlan = &SQLLoginResource{}

func NewSQLLoginResource() resource.Resource {
	return &SQLLoginResource{}
}

type SQLLoginResource struct {
	client *mssql.Client
}

// ServerModel describes the server connection configuration when overridden on the resource.
type ServerModel struct {
	Hostname  types.String    `tfsdk:"hostname"`
	Host      types.String    `tfsdk:"host"`
	Port      types.Int64     `tfsdk:"port"`
	SQLAuth   *SQLAuthModel   `tfsdk:"sql_auth"`
	Login     *SQLAuthModel   `tfsdk:"login"`
	AzureAuth *AzureAuthModel `tfsdk:"azure_auth"`
}

type SQLLoginResourceModel struct {
	ID                     types.String `tfsdk:"id"`
	Name                   types.String `tfsdk:"name"`
	LoginName              types.String `tfsdk:"login_name"`
	Password               types.String `tfsdk:"password"`
	PasswordWO             types.String `tfsdk:"password_wo"`
	PasswordWOVersion      types.String `tfsdk:"password_wo_version"`
	SID                    types.String `tfsdk:"sid"`
	DefaultDatabase        types.String `tfsdk:"default_database"`
	DefaultLanguage        types.String `tfsdk:"default_language"`
	CheckExpirationEnabled types.Bool   `tfsdk:"check_expiration_enabled"`
	CheckPolicyEnabled     types.Bool   `tfsdk:"check_policy_enabled"`
	IsDisabled             types.Bool   `tfsdk:"is_disabled"`
	Server                 *ServerModel `tfsdk:"server"`
}

// validateLoginPasswordOnCreate rejects a new login without a password. SQL Server cannot
// create a SQL login without one, while an existing login (imported, or created earlier)
// needs none in the configuration: its password is not readable and is left alone.
func validateLoginPasswordOnCreate(config SQLLoginResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics
	emptyPassword := !config.Password.IsNull() && !config.Password.IsUnknown() && config.Password.ValueString() == ""
	emptyWriteOnly := !config.PasswordWO.IsNull() && !config.PasswordWO.IsUnknown() && config.PasswordWO.ValueString() == ""
	if (config.Password.IsNull() && config.PasswordWO.IsNull()) || emptyPassword || emptyWriteOnly {
		diags.AddAttributeError(
			path.Root("password"),
			"Missing password",
			"A password is needed to create a SQL login: set `password`, or `password_wo` to keep it out of the plan "+
				"and state files (it requires Terraform 1.11 or later). A login that already exists, for example an "+
				"imported one, needs neither; its password is then not managed.",
		)
	}
	return diags
}

// changedKnown reports whether a known planned value differs from the state; unknown values are
// decided at apply time, where CreateSQLLogin still rejects an empty password.
func changedKnown(plan, state attr.Value) bool {
	return !plan.IsUnknown() && !plan.Equal(state)
}

// loginReplaced reports whether the plan replaces the login, judged by the attributes with RequiresReplace
// (the resource-level ModifyPlan does not receive the attribute-level result).
func loginReplaced(plan, state SQLLoginResourceModel) bool {
	if changedKnown(plan.Name, state.Name) || changedKnown(plan.LoginName, state.LoginName) || changedKnown(plan.SID, state.SID) {
		return true
	}
	planServer, stateServer := plan.Server, state.Server
	if planServer == nil {
		planServer = &ServerModel{}
	}
	if stateServer == nil {
		stateServer = &ServerModel{}
	}
	return changedKnown(planServer.Hostname, stateServer.Hostname) || changedKnown(planServer.Host, stateServer.Host) ||
		(!planServer.Port.IsNull() && !stateServer.Port.IsNull() && changedKnown(planServer.Port, stateServer.Port))
}

// ModifyPlan requires a password when the login is about to be created, including a replacement.
func (r *SQLLoginResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}

	if !req.State.Raw.IsNull() {
		var plan, state SQLLoginResourceModel
		resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
		resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
		if resp.Diagnostics.HasError() || !loginReplaced(plan, state) {
			return
		}
	}

	var config SQLLoginResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(validateLoginPasswordOnCreate(config)...)
}

// validateLoginPassword checks that at most one of the two password attributes
// is configured. Unknown counts as configured: an ephemeral value assigned to
// password_wo is unknown until apply.
func validateLoginPassword(data SQLLoginResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	passwordSet := !data.Password.IsNull()
	writeOnlySet := !data.PasswordWO.IsNull()

	if passwordSet && writeOnlySet {
		diags.AddAttributeError(
			path.Root("password_wo"),
			"Conflicting password attributes",
			"Only one of `password` and `password_wo` can be set.",
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

// getLoginName resolves the login name from either 'name' or 'login_name'.
func getLoginName(data SQLLoginResourceModel) string {
	if !data.Name.IsNull() && data.Name.ValueString() != "" {
		return data.Name.ValueString()
	}
	if !data.LoginName.IsNull() && data.LoginName.ValueString() != "" {
		return data.LoginName.ValueString()
	}
	return ""
}

// validateLoginName ensures exactly one of 'name' or 'login_name' is configured.
func validateLoginName(data SQLLoginResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics
	hasName := !data.Name.IsNull() && !data.Name.IsUnknown() && data.Name.ValueString() != ""
	hasLoginName := !data.LoginName.IsNull() && !data.LoginName.IsUnknown() && data.LoginName.ValueString() != ""

	if hasName && hasLoginName && data.Name.ValueString() != data.LoginName.ValueString() {
		diags.AddAttributeError(
			path.Root("login_name"),
			"Conflicting login name attributes",
			"Only one of `name` and `login_name` can be set.",
		)
	} else if !hasName && !hasLoginName {
		if data.Name.IsNull() && data.LoginName.IsNull() {
			diags.AddAttributeError(
				path.Root("name"),
				"Missing login name",
				"One of `name` or `login_name` must be set.",
			)
		}
	}

	return diags
}

// validateServerConfig ensures valid server override attributes.
func validateServerConfig(server *ServerModel) diag.Diagnostics {
	var diags diag.Diagnostics
	if server == nil {
		return diags
	}

	hasHostname := !server.Hostname.IsNull() && !server.Hostname.IsUnknown() && server.Hostname.ValueString() != ""
	hasHost := !server.Host.IsNull() && !server.Host.IsUnknown() && server.Host.ValueString() != ""

	if hasHostname && hasHost && server.Hostname.ValueString() != server.Host.ValueString() {
		diags.AddAttributeError(
			path.Root("server").AtName("host"),
			"Conflicting hostname attributes",
			"Both 'hostname' and 'host' are specified with different values.",
		)
	}

	if !hasHostname && !hasHost && !server.Hostname.IsUnknown() && !server.Host.IsUnknown() {
		diags.AddAttributeError(
			path.Root("server").AtName("hostname"),
			"Missing server hostname",
			"Either 'hostname' or 'host' must be specified in the 'server' block.",
		)
	}

	hasSQLAuth := server.SQLAuth != nil
	hasLogin := server.Login != nil
	hasAzureAuth := server.AzureAuth != nil

	if hasSQLAuth && hasLogin {
		diags.AddAttributeError(
			path.Root("server").AtName("login"),
			"Conflicting authentication blocks",
			"Only one of 'sql_auth' and 'login' can be specified.",
		)
	}

	if (hasSQLAuth || hasLogin) && hasAzureAuth {
		diags.AddAttributeError(
			path.Root("server").AtName("azure_auth"),
			"Conflicting authentication methods",
			"Only one of 'sql_auth' (or 'login') and 'azure_auth' can be configured.",
		)
	}

	if !hasSQLAuth && !hasLogin && !hasAzureAuth {
		diags.AddAttributeError(
			path.Root("server"),
			"Missing authentication configuration",
			"Either 'sql_auth', 'login', or 'azure_auth' must be configured in the 'server' block.",
		)
	}

	return diags
}

// serverToConfig converts a ServerModel to mssql.Config.
func serverToConfig(server *ServerModel) (*mssql.Config, diag.Diagnostics) {
	var diags diag.Diagnostics
	if server == nil {
		return nil, diags
	}

	var hostname string
	if !server.Hostname.IsNull() && server.Hostname.ValueString() != "" {
		hostname = server.Hostname.ValueString()
	} else if !server.Host.IsNull() && server.Host.ValueString() != "" {
		hostname = server.Host.ValueString()
	}

	if hostname == "" {
		diags.AddAttributeError(
			path.Root("server").AtName("hostname"),
			"Missing server hostname",
			"Either 'hostname' or 'host' must be specified in the 'server' block.",
		)
		return nil, diags
	}

	port := 1433
	if !server.Port.IsNull() && server.Port.ValueInt64() > 0 {
		port = int(server.Port.ValueInt64())
	}

	auth := server.SQLAuth
	if auth == nil {
		auth = server.Login
	}

	cfg := &mssql.Config{
		Hostname: hostname,
		Port:     port,
	}

	if auth != nil {
		cfg.SQLAuth = &mssql.SQLAuthConfig{
			Username: auth.Username.ValueString(),
			Password: auth.Password.ValueString(),
		}
	} else if server.AzureAuth != nil {
		cfg.AzureAuth = &mssql.AzureAuthConfig{
			ClientID:     server.AzureAuth.ClientID.ValueString(),
			ClientSecret: server.AzureAuth.ClientSecret.ValueString(),
			TenantID:     server.AzureAuth.TenantID.ValueString(),
		}
	} else {
		diags.AddAttributeError(
			path.Root("server"),
			"Missing authentication configuration",
			"Either 'sql_auth', 'login', or 'azure_auth' must be configured in the 'server' block.",
		)
		return nil, diags
	}

	return cfg, diags
}

// getClient returns the client to use: either from the resource server block or provider default.
func (r *SQLLoginResource) getClient(ctx context.Context, server *ServerModel) (*mssql.Client, diag.Diagnostics) {
	var diags diag.Diagnostics

	if server != nil && (!server.Hostname.IsNull() || !server.Host.IsNull()) {
		cfg, d := serverToConfig(server)
		diags.Append(d...)
		if diags.HasError() {
			return nil, diags
		}

		if r.client == nil {
			diags.AddError(
				"MSSQL Provider Not Configured",
				"The MSSQL provider has not been initialized.",
			)
			return nil, diags
		}

		client, err := r.client.GetClientForServer(ctx, cfg)
		if err != nil {
			diags.AddError(
				"Failed to Connect to SQL Server",
				fmt.Sprintf("Failed to establish connection to server %s:%d: %s", cfg.Hostname, cfg.Port, err.Error()),
			)
			return nil, diags
		}
		return client, diags
	}

	// Fall back to provider default client
	if r.client == nil || r.client.DB() == nil {
		diags.AddError(
			"Default MSSQL Connection Not Configured",
			"No connection configuration was found. You must either configure connection settings in the provider block (hostname, auth) or provide a 'server' block on the resource.",
		)
		return nil, diags
	}

	return r.client, diags
}

// loginCreatePassword returns the password for a new login. Write-only values
// are stripped from the plan, so password_wo has to be read from the config.
func loginCreatePassword(plan, config SQLLoginResourceModel) string {
	if !config.PasswordWO.IsNull() {
		return config.PasswordWO.ValueString()
	}
	return plan.Password.ValueString()
}

// loginUpdatePassword returns the password to write to the server, or nil when
// the login's password does not need changing.
func loginUpdatePassword(plan, state, config SQLLoginResourceModel) *string {
	if !config.PasswordWO.IsNull() {
		// A write-only value is in neither the plan nor the state, so it cannot
		// be compared. The signals that the server needs a new password are a
		// changed password_wo_version and a migration off `password`, whose
		// value is still in state.
		if plan.PasswordWOVersion.Equal(state.PasswordWOVersion) && state.Password.IsNull() {
			return nil
		}
		password := config.PasswordWO.ValueString()
		return &password
	}

	if plan.Password.Equal(state.Password) || plan.Password.IsNull() || plan.Password.ValueString() == "" {
		return nil
	}

	password := plan.Password.ValueString()
	return &password
}

type sidPlanModifier struct{}

func (m sidPlanModifier) Description(ctx context.Context) string {
	return "Suppresses diffs when the planned SID and state SID are semantically equivalent."
}

func (m sidPlanModifier) MarkdownDescription(ctx context.Context) string {
	return "Suppresses diffs when the planned SID and state SID are semantically equivalent."
}

func (m sidPlanModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.StateValue.IsNull() || req.StateValue.IsUnknown() || req.PlanValue.IsNull() || req.PlanValue.IsUnknown() {
		return
	}

	normState, errState := mssql.NormalizeSID(req.StateValue.ValueString())
	normPlan, errPlan := mssql.NormalizeSID(req.PlanValue.ValueString())

	if errState == nil && errPlan == nil && normState == normPlan {
		resp.PlanValue = req.StateValue
	}
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
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"login_name": schema.StringAttribute{
				Description: "Alias for `name`. The name of the login.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"password": schema.StringAttribute{
				Description: "The password for the login. Persisted in the plan and state files; " +
					"use `password_wo` instead to avoid that. At most one of `password` and `password_wo` can be set; one of them is needed to create the login, an existing login needs neither.",
				Optional:  true,
				Sensitive: true,
			},
			"password_wo": schema.StringAttribute{
				Description: "The password for the login, as a write-only attribute. Accepts ephemeral values, " +
					"such as those from `ephemeral.random_password`, and is written to neither the plan nor the " +
					"state file. Requires Terraform 1.11 or later. At most one of `password` and `password_wo` " +
					"can be set. Because Terraform has no stored value to compare against, changing this alone " +
					"does not update the login; change `password_wo_version` to apply a new password.",
				Optional:  true,
				Sensitive: true,
				WriteOnly: true,
			},
			"password_wo_version": schema.StringAttribute{
				Description: "An arbitrary token whose change triggers an `ALTER LOGIN` with the current " +
					"`password_wo` value. Only valid together with `password_wo`. Without it, a rotated " +
					"`password_wo` is never applied.",
				Optional: true,
			},
			"sid": schema.StringAttribute{
				Description: "The SID (Security Identifier) of the SQL login in hexadecimal format (e.g., 0x0105...). Changing this forces a new resource to be created.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					sidPlanModifier{},
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
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
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
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
		Blocks: map[string]schema.Block{
			"server": schema.SingleNestedBlock{
				Description: "SQL Server instance configuration. If omitted, the default provider connection settings are used.",
				Attributes: map[string]schema.Attribute{
					"hostname": schema.StringAttribute{
						Description: "FQDN or IP address of the SQL endpoint.",
						Optional:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
					"host": schema.StringAttribute{
						Description: "Alias for `hostname`. FQDN or IP address of the SQL endpoint.",
						Optional:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
					"port": schema.Int64Attribute{
						Description: "TCP port of SQL endpoint. Defaults to 1433.",
						Optional:    true,
						Computed:    true,
						Default:     int64default.StaticInt64(1433),
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.RequiresReplace(),
						},
					},
				},
				Blocks: map[string]schema.Block{
					"sql_auth": schema.SingleNestedBlock{
						Description: "SQL authentication credentials. Either sql_auth, login, or azure_auth must be provided.",
						Attributes: map[string]schema.Attribute{
							"username": schema.StringAttribute{
								Description: "Username for SQL authentication.",
								Optional:    true,
							},
							"password": schema.StringAttribute{
								Description: "Password for SQL authentication.",
								Optional:    true,
								Sensitive:   true,
							},
						},
					},
					"login": schema.SingleNestedBlock{
						Description: "Alias for `sql_auth`. SQL authentication credentials.",
						Attributes: map[string]schema.Attribute{
							"username": schema.StringAttribute{
								Description: "Username for SQL authentication.",
								Optional:    true,
							},
							"password": schema.StringAttribute{
								Description: "Password for SQL authentication.",
								Optional:    true,
								Sensitive:   true,
							},
						},
					},
					"azure_auth": schema.SingleNestedBlock{
						Description: "Azure AD authentication configuration.",
						Attributes: map[string]schema.Attribute{
							"client_id": schema.StringAttribute{
								Description: "Service Principal client (application) ID.",
								Optional:    true,
							},
							"client_secret": schema.StringAttribute{
								Description: "Service Principal secret.",
								Optional:    true,
								Sensitive:   true,
							},
							"tenant_id": schema.StringAttribute{
								Description: "Azure AD tenant ID.",
								Optional:    true,
							},
						},
					},
				},
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

func (r *SQLLoginResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data SQLLoginResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(validateLoginPassword(data)...)
	resp.Diagnostics.Append(validateLoginName(data)...)
	resp.Diagnostics.Append(validateServerConfig(data.Server)...)
}

func (r *SQLLoginResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data, config SQLLoginResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)

	if resp.Diagnostics.HasError() {
		return
	}

	client, d := r.getClient(ctx, data.Server)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	loginName := getLoginName(data)

	tflog.Debug(ctx, "Creating SQL login", map[string]interface{}{
		"name": loginName,
	})

	opts := mssql.CreateSQLLoginOptions{
		Name:                   loginName,
		Password:               loginCreatePassword(data, config),
		SID:                    data.SID.ValueString(),
		DefaultDatabase:        data.DefaultDatabase.ValueString(),
		DefaultLanguage:        data.DefaultLanguage.ValueString(),
		CheckExpirationEnabled: data.CheckExpirationEnabled.ValueBool(),
		CheckPolicyEnabled:     data.CheckPolicyEnabled.ValueBool(),
	}

	login, err := client.CreateSQLLogin(ctx, opts)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create SQL login", err.Error())
		return
	}

	// Handle disabled state
	if data.IsDisabled.ValueBool() {
		disabled := true
		_, err := client.UpdateSQLLogin(ctx, mssql.UpdateSQLLoginOptions{
			Name:       loginName,
			IsDisabled: &disabled,
		})
		if err != nil {
			resp.Diagnostics.AddError("Failed to disable SQL login", err.Error())
			return
		}
	}

	data.ID = types.StringValue(strconv.Itoa(login.PrincipalID))
	data.Name = types.StringValue(login.Name)
	if !data.LoginName.IsNull() {
		data.LoginName = types.StringValue(login.Name)
	}
	data.SID = types.StringValue(login.SID)
	data.DefaultLanguage = types.StringValue(login.DefaultLanguageName)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SQLLoginResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data SQLLoginResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	client, d := r.getClient(ctx, data.Server)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	loginName := getLoginName(data)

	var login *mssql.SQLLogin
	var err error

	// Try to find by ID first
	id, parseErr := strconv.Atoi(data.ID.ValueString())
	if parseErr == nil {
		login, err = client.GetSQLLoginByID(ctx, id)
		if err != nil {
			resp.Diagnostics.AddError("Failed to read SQL login", err.Error())
			return
		}
	}

	// If not found by ID, try to find by name (handles ID changes)
	if login == nil && loginName != "" {
		login, err = client.GetSQLLogin(ctx, loginName)
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
	if !data.LoginName.IsNull() {
		data.LoginName = types.StringValue(login.Name)
	}
	data.SID = types.StringValue(login.SID)
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
	var config SQLLoginResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)

	if resp.Diagnostics.HasError() {
		return
	}

	client, d := r.getClient(ctx, data.Server)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	loginName := getLoginName(data)

	tflog.Debug(ctx, "Updating SQL login", map[string]interface{}{
		"name": loginName,
	})

	opts := mssql.UpdateSQLLoginOptions{
		Name: loginName,
	}

	// Check what changed - only update if values actually differ
	opts.Password = loginUpdatePassword(data, state, config)
	if !data.DefaultDatabase.Equal(state.DefaultDatabase) {
		db := data.DefaultDatabase.ValueString()
		opts.DefaultDatabase = &db
	}
	// Only update language if explicitly changed and not empty
	if !data.DefaultLanguage.Equal(state.DefaultLanguage) && !data.DefaultLanguage.IsNull() && data.DefaultLanguage.ValueString() != "" {
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

	// Skip update if nothing changed
	if opts.Password == nil && opts.DefaultDatabase == nil && opts.DefaultLanguage == nil &&
		opts.CheckExpirationEnabled == nil && opts.CheckPolicyEnabled == nil && opts.IsDisabled == nil {
		if data.DefaultLanguage.IsUnknown() {
			data.DefaultLanguage = state.DefaultLanguage
		}
		if data.SID.IsUnknown() {
			data.SID = state.SID
		}
		data.Name = types.StringValue(loginName)
		if !data.LoginName.IsNull() {
			data.LoginName = types.StringValue(loginName)
		}
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		return
	}

	login, err := client.UpdateSQLLogin(ctx, opts)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update SQL login", err.Error())
		return
	}

	// Update state with actual values from the server to resolve "known after apply" values
	data.Name = types.StringValue(login.Name)
	if !data.LoginName.IsNull() {
		data.LoginName = types.StringValue(login.Name)
	}
	data.SID = types.StringValue(login.SID)
	data.DefaultLanguage = types.StringValue(login.DefaultLanguageName)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SQLLoginResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data SQLLoginResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	client, d := r.getClient(ctx, data.Server)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	loginName := getLoginName(data)

	tflog.Debug(ctx, "Deleting SQL login", map[string]interface{}{
		"name": loginName,
	})

	err := client.DropSQLLogin(ctx, loginName)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete SQL login", err.Error())
		return
	}
}

func (r *SQLLoginResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	client, d := r.getClient(ctx, nil)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	login, err := client.GetSQLLogin(ctx, req.ID)
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
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("sid"), login.SID)...)
	// The password is not readable, so it stays unset: a configuration without one then shows no diff,
	// and one with a password (or a new password_wo_version) writes it on the first apply.
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("default_database"), login.DefaultDatabaseName)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("default_language"), login.DefaultLanguageName)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("check_expiration_enabled"), login.CheckExpirationEnabled)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("check_policy_enabled"), login.CheckPolicyEnabled)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("is_disabled"), login.IsDisabled)...)
}
