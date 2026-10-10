// Copyright (c) 2024 muecahit94
// SPDX-License-Identifier: MIT

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/muecahit94/terraform-provider-mssql/internal/mssql"
)

var _ resource.Resource = &ServerConfigurationResource{}
var _ resource.ResourceWithImportState = &ServerConfigurationResource{}

// NewServerConfigurationResource creates a new server configuration resource.
func NewServerConfigurationResource() resource.Resource {
	return &ServerConfigurationResource{}
}

// ServerConfigurationResource manages one option of sp_configure.
type ServerConfigurationResource struct {
	client *mssql.Client
}

// ServerConfigurationResourceModel describes the resource data model.
type ServerConfigurationResourceModel struct {
	ID              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	Value           types.Int64  `tfsdk:"value"`
	PreviousValue   types.Int64  `tfsdk:"previous_value"`
	ValueInUse      types.Int64  `tfsdk:"value_in_use"`
	RestartRequired types.Bool   `tfsdk:"restart_required"`
}

func (r *ServerConfigurationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_server_configuration"
}

func (r *ServerConfigurationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages one server configuration option (`sp_configure`), for example `max server memory (MB)` or `xp_cmdshell`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The name of the option.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the option exactly as `sp_configure` and `sys.configurations` show it, " +
					"for example `max degree of parallelism`. Changing this forces a new resource to be created.",
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"value": schema.Int64Attribute{
				Description: "The value of the option. It is checked against the range the server reports for the option.",
				Required:    true,
			},
			"previous_value": schema.Int64Attribute{
				Description: "The value the option had when this resource took it over. It is restored when the resource is destroyed. " +
					"For an imported option it is the value at the time of the import, so destroying an imported option changes nothing.",
				Computed: true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"value_in_use": schema.Int64Attribute{
				Description: "The value the running server uses. It differs from `value` for an option that is not dynamic until the server is restarted.",
				Computed:    true,
			},
			"restart_required": schema.BoolAttribute{
				Description: "Whether the configured value takes effect only after the server is restarted.",
				Computed:    true,
			},
		},
	}
}

func (r *ServerConfigurationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// applyServerConfiguration copies the values read from the server into the model.
func applyServerConfiguration(data *ServerConfigurationResourceModel, cfg *mssql.ServerConfiguration) {
	data.ID = types.StringValue(cfg.Name)
	// sys.configurations matches names case-insensitively; keep the configured spelling to avoid a replacement.
	data.Name = keepCase(data.Name, cfg.Name)
	data.Value = types.Int64Value(cfg.Value)
	data.ValueInUse = types.Int64Value(cfg.ValueInUse)
	data.RestartRequired = types.BoolValue(cfg.RestartRequired())
}

func (r *ServerConfigurationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ServerConfigurationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := data.Name.ValueString()
	current, err := r.client.GetServerConfiguration(ctx, name)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read the server configuration", err.Error())
		return
	}
	if current == nil {
		resp.Diagnostics.AddError("Unknown server configuration option",
			fmt.Sprintf("The server has no option %q. Take the name from `sys.configurations` or the output of `sp_configure`.", name))
		return
	}

	if err := r.client.SetServerConfiguration(ctx, name, data.Value.ValueInt64()); err != nil {
		resp.Diagnostics.AddError("Failed to set the server configuration", err.Error())
		return
	}

	updated, err := r.client.GetServerConfiguration(ctx, name)
	if err != nil || updated == nil {
		resp.Diagnostics.AddError("Failed to read the server configuration after the change", fmt.Sprint(err))
		return
	}

	data.PreviousValue = types.Int64Value(current.Value)
	applyServerConfiguration(&data, updated)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ServerConfigurationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ServerConfigurationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cfg, err := r.client.GetServerConfiguration(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read the server configuration", err.Error())
		return
	}
	if cfg == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	applyServerConfiguration(&data, cfg)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ServerConfigurationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state ServerConfigurationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := state.Name.ValueString()
	if err := r.client.SetServerConfiguration(ctx, name, plan.Value.ValueInt64()); err != nil {
		resp.Diagnostics.AddError("Failed to set the server configuration", err.Error())
		return
	}

	cfg, err := r.client.GetServerConfiguration(ctx, name)
	if err != nil || cfg == nil {
		resp.Diagnostics.AddError("Failed to read the server configuration after the change", fmt.Sprint(err))
		return
	}

	plan.PreviousValue = state.PreviousValue
	applyServerConfiguration(&plan, cfg)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete restores the value the option had before the resource took it over.
func (r *ServerConfigurationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ServerConfigurationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.PreviousValue.IsNull() || data.PreviousValue.IsUnknown() {
		return
	}
	cfg, err := r.client.GetServerConfiguration(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read the server configuration", err.Error())
		return
	}
	if cfg == nil || cfg.Value == data.PreviousValue.ValueInt64() {
		return
	}

	if err := r.client.SetServerConfiguration(ctx, data.Name.ValueString(), data.PreviousValue.ValueInt64()); err != nil {
		resp.Diagnostics.AddError("Failed to restore the server configuration", err.Error())
	}
}

func (r *ServerConfigurationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	cfg, err := r.client.GetServerConfiguration(ctx, req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to import the server configuration", err.Error())
		return
	}
	if cfg == nil {
		resp.Diagnostics.AddError("Unknown server configuration option", fmt.Sprintf("The server has no option %q.", req.ID))
		return
	}

	// An imported option is not changed by the import: its current value is the one to go back to.
	data := ServerConfigurationResourceModel{PreviousValue: types.Int64Value(cfg.Value)}
	applyServerConfiguration(&data, cfg)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
