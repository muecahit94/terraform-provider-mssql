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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/muecahit94/terraform-provider-mssql/internal/mssql"
)

var _ resource.Resource = &LinkedServerResource{}
var _ resource.ResourceWithImportState = &LinkedServerResource{}
var _ resource.ResourceWithValidateConfig = &LinkedServerResource{}

func NewLinkedServerResource() resource.Resource {
	return &LinkedServerResource{}
}

type LinkedServerResource struct {
	client *mssql.Client
}

type LinkedServerResourceModel struct {
	ID                             types.String `tfsdk:"id"`
	Name                           types.String `tfsdk:"name"`
	Product                        types.String `tfsdk:"product"`
	ProviderName                   types.String `tfsdk:"provider_name"`
	DataSource                     types.String `tfsdk:"data_source"`
	Location                       types.String `tfsdk:"location"`
	ProviderString                 types.String `tfsdk:"provider_string"`
	Catalog                        types.String `tfsdk:"catalog"`
	RPC                            types.Bool   `tfsdk:"rpc"`
	RPCOut                         types.Bool   `tfsdk:"rpc_out"`
	DataAccess                     types.Bool   `tfsdk:"data_access"`
	CollationCompatible            types.Bool   `tfsdk:"collation_compatible"`
	UseRemoteCollation             types.Bool   `tfsdk:"use_remote_collation"`
	CollationName                  types.String `tfsdk:"collation_name"`
	ConnectTimeout                 types.Int64  `tfsdk:"connect_timeout"`
	QueryTimeout                   types.Int64  `tfsdk:"query_timeout"`
	LazySchemaValidation           types.Bool   `tfsdk:"lazy_schema_validation"`
	RemoteProcTransactionPromotion types.Bool   `tfsdk:"remote_proc_transaction_promotion"`
}

// definitionAttribute builds an attribute that sp_addlinkedserver sets at creation and
// SQL Server cannot alter, so a change forces a new linked server. It is Computed because
// SQL Server fills in some of them itself, for example the provider of a SQL Server product.
func definitionAttribute(description string, sensitive bool) schema.StringAttribute {
	return schema.StringAttribute{
		Description: description + " Changing this forces a new resource to be created.",
		Optional:    true,
		Computed:    true,
		Sensitive:   sensitive,
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.RequiresReplace(),
			stringplanmodifier.UseStateForUnknown(),
		},
	}
}

func (r *LinkedServerResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_linked_server"
}

func (r *LinkedServerResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a SQL Server linked server. Login mappings are managed with `mssql_linked_server_login`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The server ID of the linked server.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the linked server. Changing this forces a new resource to be created.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"product":         definitionAttribute("The product name of the data source. Use `SQL Server` for a remote SQL Server whose network name is `name` (no `data_source` needed), or an empty string together with `provider_name` otherwise.", false),
			"provider_name":   definitionAttribute("The unique programmatic identifier (PROGID) of the OLE DB provider, for example `MSOLEDBSQL` or `MSDASQL`. Named `provider_name` because `provider` is reserved by Terraform.", false),
			"data_source":     definitionAttribute("The name of the data source as interpreted by the OLE DB provider.", false),
			"location":        definitionAttribute("The location of the database as interpreted by the OLE DB provider.", false),
			"provider_string": definitionAttribute("The OLE DB provider-specific connection string. Marked sensitive because it commonly embeds credentials, for example `Uid`/`Pwd` of an ODBC connection string. It is read back from the server, so an existing linked server can be imported and the attribute left out of the configuration.", true),
			"catalog":         definitionAttribute("The catalog or default database to use when connecting to the provider.", false),
			"rpc": schema.BoolAttribute{
				Description: "Enables remote procedure calls from the remote server to this server. Defaults to `false`.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"rpc_out": schema.BoolAttribute{
				Description: "Enables remote procedure calls from this server to the remote server. Defaults to `false`.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"data_access": schema.BoolAttribute{
				Description: "Enables the linked server for distributed query access. Defaults to `true`.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"collation_compatible": schema.BoolAttribute{
				Description: "Whether the remote server has the same collation as this server. Defaults to `false`.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"use_remote_collation": schema.BoolAttribute{
				Description: "Whether to use the collation of the remote server instead of `collation_name`. Defaults to `true`.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"collation_name": schema.StringAttribute{
				Description: "The collation to use for the remote server. Only valid when `use_remote_collation` is `false`.",
				Optional:    true,
			},
			"connect_timeout": schema.Int64Attribute{
				Description: "The connection timeout in seconds. `0` uses the server default. Defaults to `0`.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(0),
			},
			"query_timeout": schema.Int64Attribute{
				Description: "The query timeout in seconds. `0` uses the server default. Defaults to `0`.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(0),
			},
			"lazy_schema_validation": schema.BoolAttribute{
				Description: "Skips checking the schema of remote tables at the start of a query. Defaults to `false`.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"remote_proc_transaction_promotion": schema.BoolAttribute{
				Description: "Whether calling a remote stored procedure starts a distributed transaction (MSDTC). Defaults to `true`.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
		},
	}
}

func (r *LinkedServerResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *LinkedServerResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data LinkedServerResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// For the `SQL Server` product the linked server name is the network name of the
	// remote instance, and SQL Server rejects a data source without an OLE DB provider.
	if !data.Product.IsNull() && !data.Product.IsUnknown() && data.Product.ValueString() == "SQL Server" &&
		!data.DataSource.IsNull() && data.ProviderName.IsNull() {
		resp.Diagnostics.AddAttributeError(
			path.Root("data_source"),
			"Conflicting data source settings",
			"`data_source` cannot be combined with `product = \"SQL Server\"` unless `provider_name` is set. "+
				"With that product the linked server `name` is the network name of the remote instance; "+
				"to use a different name, set `provider_name` (for example `MSOLEDBSQL`) and `product = \"\"`.",
		)
	}

	// An unknown value is resolved at apply time and is not rejected here.
	if !data.CollationName.IsNull() && !data.CollationName.IsUnknown() &&
		!data.UseRemoteCollation.IsNull() && !data.UseRemoteCollation.IsUnknown() &&
		data.UseRemoteCollation.ValueBool() {
		resp.Diagnostics.AddAttributeError(
			path.Root("collation_name"),
			"Conflicting collation settings",
			"`collation_name` can only be set when `use_remote_collation` is `false`.",
		)
	}
	if !data.CollationName.IsNull() && !data.CollationName.IsUnknown() &&
		data.UseRemoteCollation.IsNull() {
		resp.Diagnostics.AddAttributeError(
			path.Root("collation_name"),
			"Conflicting collation settings",
			"`collation_name` requires `use_remote_collation = false`; it defaults to `true`.",
		)
	}
}

func linkedServerOptions(data LinkedServerResourceModel) mssql.LinkedServerOptions {
	return mssql.LinkedServerOptions{
		RPC:                  data.RPC.ValueBool(),
		RPCOut:               data.RPCOut.ValueBool(),
		DataAccess:           data.DataAccess.ValueBool(),
		CollationCompatible:  data.CollationCompatible.ValueBool(),
		UseRemoteCollation:   data.UseRemoteCollation.ValueBool(),
		CollationName:        data.CollationName.ValueString(),
		ConnectTimeout:       int(data.ConnectTimeout.ValueInt64()),
		QueryTimeout:         int(data.QueryTimeout.ValueInt64()),
		LazySchemaValidation: data.LazySchemaValidation.ValueBool(),
		RemoteProcTransPromo: data.RemoteProcTransactionPromotion.ValueBool(),
	}
}

// applyLinkedServer copies server state into the model. Empty optional values are mapped to
// null for the attributes that are not Computed, so an unset attribute does not show a diff.
func applyLinkedServer(data *LinkedServerResourceModel, s *mssql.LinkedServer) {
	data.ID = types.StringValue(strconv.Itoa(s.ServerID))
	data.Name = types.StringValue(s.Name)
	data.Product = types.StringValue(s.Product)
	data.ProviderName = types.StringValue(s.Provider)
	data.DataSource = types.StringValue(s.DataSource)
	data.Location = types.StringValue(s.Location)
	data.ProviderString = types.StringValue(s.ProviderString)
	data.Catalog = types.StringValue(s.Catalog)
	data.RPC = types.BoolValue(s.IsRemoteLoginEnabled)
	data.RPCOut = types.BoolValue(s.IsRPCOutEnabled)
	data.DataAccess = types.BoolValue(s.IsDataAccessEnabled)
	data.CollationCompatible = types.BoolValue(s.IsCollationCompatible)
	data.UseRemoteCollation = types.BoolValue(s.UsesRemoteCollation)
	if s.CollationName == "" {
		data.CollationName = types.StringNull()
	} else {
		data.CollationName = types.StringValue(s.CollationName)
	}
	data.ConnectTimeout = types.Int64Value(int64(s.ConnectTimeout))
	data.QueryTimeout = types.Int64Value(int64(s.QueryTimeout))
	data.LazySchemaValidation = types.BoolValue(s.LazySchemaValidation)
	data.RemoteProcTransactionPromotion = types.BoolValue(s.RemoteProcTransPromoted)
}

func (r *LinkedServerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data LinkedServerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	server, err := r.client.CreateLinkedServer(ctx, mssql.CreateLinkedServerOptions{
		Name:           data.Name.ValueString(),
		Product:        data.Product.ValueString(),
		Provider:       data.ProviderName.ValueString(),
		DataSource:     data.DataSource.ValueString(),
		Location:       data.Location.ValueString(),
		ProviderString: data.ProviderString.ValueString(),
		Catalog:        data.Catalog.ValueString(),
		Options:        linkedServerOptions(data),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create linked server", err.Error())
		return
	}
	if server == nil {
		resp.Diagnostics.AddError("Failed to create linked server", "The linked server was not found after creation.")
		return
	}

	applyLinkedServer(&data, server)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *LinkedServerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data LinkedServerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var server *mssql.LinkedServer
	var err error
	if id, convErr := strconv.Atoi(data.ID.ValueString()); convErr == nil {
		server, err = r.client.GetLinkedServerByID(ctx, id)
	}
	// The server ID changes when the linked server is recreated outside Terraform.
	if err == nil && server == nil {
		server, err = r.client.GetLinkedServer(ctx, data.Name.ValueString())
	}
	if err != nil {
		resp.Diagnostics.AddError("Failed to read linked server", err.Error())
		return
	}
	if server == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	applyLinkedServer(&data, server)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *LinkedServerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data LinkedServerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Only the sp_serveroption settings can change in place; everything else forces replacement.
	if err := r.client.SetLinkedServerOptions(ctx, data.Name.ValueString(), linkedServerOptions(data)); err != nil {
		resp.Diagnostics.AddError("Failed to update linked server", err.Error())
		return
	}

	server, err := r.client.GetLinkedServer(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read linked server", err.Error())
		return
	}
	if server == nil {
		resp.Diagnostics.AddError("Failed to update linked server", "The linked server was not found after the update.")
		return
	}

	applyLinkedServer(&data, server)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *LinkedServerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data LinkedServerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DropLinkedServer(ctx, data.Name.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to delete linked server", err.Error())
		return
	}
}

func (r *LinkedServerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	server, err := r.client.GetLinkedServer(ctx, req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to import linked server", err.Error())
		return
	}
	if server == nil {
		resp.Diagnostics.AddError("Linked server not found", fmt.Sprintf("Linked server '%s' not found", req.ID))
		return
	}

	var data LinkedServerResourceModel
	applyLinkedServer(&data, server)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
