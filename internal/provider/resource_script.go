// Copyright (c) 2024 muecahit94
// SPDX-License-Identifier: MIT

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/muecahit94/terraform-provider-mssql/internal/mssql"
)

var _ resource.Resource = &ScriptResource{}

func NewScriptResource() resource.Resource {
	return &ScriptResource{}
}

type ScriptResource struct {
	client *mssql.Client
}

type ScriptResourceModel struct {
	ID           types.String `tfsdk:"id"`
	DatabaseName types.String `tfsdk:"database_name"`
	CreateScript types.String `tfsdk:"create_script"`
	ReadScript   types.String `tfsdk:"read_script"`
	UpdateScript types.String `tfsdk:"update_script"`
	DeleteScript types.String `tfsdk:"delete_script"`
	State        types.Map    `tfsdk:"state"`
}

func (r *ScriptResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_script"
}

func (r *ScriptResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Executes custom SQL scripts for create, read, update, and delete operations.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"database_name": schema.StringAttribute{
				Description: "The database to execute scripts in. Empty for server-level scripts.",
				Optional:    true,
			},
			"create_script": schema.StringAttribute{
				Description: "SQL script to execute on resource creation.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"read_script": schema.StringAttribute{
				Description: "SQL script to execute on resource read. Should return a single row.",
				Optional:    true,
			},
			"update_script": schema.StringAttribute{
				Description: "SQL script to execute on resource update.",
				Optional:    true,
			},
			"delete_script": schema.StringAttribute{
				Description: "SQL script to execute on resource deletion.",
				Required:    true,
			},
			"state": schema.MapAttribute{
				Description: "The state returned from the read script.",
				Computed:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (r *ScriptResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ScriptResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ScriptResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.ExecuteScriptNoResult(ctx, data.DatabaseName.ValueString(), data.CreateScript.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to execute create script", err.Error())
		return
	}

	data.ID = types.StringValue(mssql.GenerateScriptID(data.CreateScript.ValueString(), data.DatabaseName.ValueString()))

	// Execute read script if provided
	if !data.ReadScript.IsNull() && data.ReadScript.ValueString() != "" {
		state, err := r.client.ExecuteScript(ctx, data.DatabaseName.ValueString(), data.ReadScript.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Failed to execute read script", err.Error())
			return
		}
		stateMap, diags := types.MapValueFrom(ctx, types.StringType, state)
		resp.Diagnostics.Append(diags...)
		data.State = stateMap
	} else {
		data.State = types.MapNull(types.StringType)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ScriptResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ScriptResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !data.ReadScript.IsNull() && data.ReadScript.ValueString() != "" {
		state, err := r.client.ExecuteScript(ctx, data.DatabaseName.ValueString(), data.ReadScript.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Failed to execute read script", err.Error())
			return
		}
		stateMap, diags := types.MapValueFrom(ctx, types.StringType, state)
		resp.Diagnostics.Append(diags...)
		data.State = stateMap
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ScriptResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ScriptResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !data.UpdateScript.IsNull() && data.UpdateScript.ValueString() != "" {
		err := r.client.ExecuteScriptNoResult(ctx, data.DatabaseName.ValueString(), data.UpdateScript.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Failed to execute update script", err.Error())
			return
		}
	}

	// Execute read script if provided
	if !data.ReadScript.IsNull() && data.ReadScript.ValueString() != "" {
		state, err := r.client.ExecuteScript(ctx, data.DatabaseName.ValueString(), data.ReadScript.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Failed to execute read script", err.Error())
			return
		}
		stateMap, diags := types.MapValueFrom(ctx, types.StringType, state)
		resp.Diagnostics.Append(diags...)
		data.State = stateMap
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ScriptResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ScriptResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.ExecuteScriptNoResult(ctx, data.DatabaseName.ValueString(), data.DeleteScript.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to execute delete script", err.Error())
		return
	}
}
