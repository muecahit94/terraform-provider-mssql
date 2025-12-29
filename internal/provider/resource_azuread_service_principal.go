// Copyright (c) 2024 muecahit94
// SPDX-License-Identifier: MIT

package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/muecahit94/terraform-provider-mssql/internal/mssql"
)

var _ resource.Resource = &AzureADServicePrincipalResource{}
var _ resource.ResourceWithImportState = &AzureADServicePrincipalResource{}

func NewAzureADServicePrincipalResource() resource.Resource {
	return &AzureADServicePrincipalResource{}
}

type AzureADServicePrincipalResource struct {
	client *mssql.Client
}

type AzureADServicePrincipalResourceModel struct {
	ID            types.String `tfsdk:"id"`
	DatabaseName  types.String `tfsdk:"database_name"`
	Name          types.String `tfsdk:"name"`
	ClientID      types.String `tfsdk:"client_id"`
	DefaultSchema types.String `tfsdk:"default_schema"`
}

func (r *AzureADServicePrincipalResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_azuread_service_principal"
}

func (r *AzureADServicePrincipalResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an Azure AD service principal in a SQL Server database.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"database_name": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The display name of the service principal.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"client_id": schema.StringAttribute{
				Description: "The Azure AD client (application) ID.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"default_schema": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("dbo"),
			},
		},
	}
}

func (r *AzureADServicePrincipalResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *AzureADServicePrincipalResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data AzureADServicePrincipalResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	user, err := r.client.CreateAzureADServicePrincipal(ctx, mssql.CreateAzureADServicePrincipalOptions{
		DatabaseName:  data.DatabaseName.ValueString(),
		Name:          data.Name.ValueString(),
		ClientID:      data.ClientID.ValueString(),
		DefaultSchema: data.DefaultSchema.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create Azure AD service principal", err.Error())
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%d/%d", user.DatabaseID, user.PrincipalID))
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AzureADServicePrincipalResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data AzureADServicePrincipalResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	user, err := r.client.GetUser(ctx, data.DatabaseName.ValueString(), data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read Azure AD service principal", err.Error())
		return
	}
	if user == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	data.DefaultSchema = types.StringValue(user.DefaultSchemaName)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AzureADServicePrincipalResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data, state AzureADServicePrincipalResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !data.DefaultSchema.Equal(state.DefaultSchema) {
		schema := data.DefaultSchema.ValueString()
		_, err := r.client.UpdateSQLUser(ctx, mssql.UpdateSQLUserOptions{
			DatabaseName:  data.DatabaseName.ValueString(),
			UserName:      data.Name.ValueString(),
			DefaultSchema: &schema,
		})
		if err != nil {
			resp.Diagnostics.AddError("Failed to update Azure AD service principal", err.Error())
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AzureADServicePrincipalResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data AzureADServicePrincipalResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DropUser(ctx, data.DatabaseName.ValueString(), data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete Azure AD service principal", err.Error())
		return
	}
}

func (r *AzureADServicePrincipalResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 {
		resp.Diagnostics.AddError("Invalid import ID", "Import ID must be in format 'database_name/principal_name'")
		return
	}

	user, err := r.client.GetUser(ctx, parts[0], parts[1])
	if err != nil {
		resp.Diagnostics.AddError("Failed to import Azure AD service principal", err.Error())
		return
	}
	if user == nil {
		resp.Diagnostics.AddError("Azure AD service principal not found", fmt.Sprintf("Principal '%s' not found in database '%s'", parts[1], parts[0]))
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), fmt.Sprintf("%d/%d", user.DatabaseID, user.PrincipalID))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("database_name"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), user.Name)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("client_id"), "")...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("default_schema"), user.DefaultSchemaName)...)
}
