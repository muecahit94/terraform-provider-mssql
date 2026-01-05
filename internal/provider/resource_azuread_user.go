// Copyright (c) 2024 muecahit94
// SPDX-License-Identifier: MIT

package provider

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/muecahit94/terraform-provider-mssql/internal/mssql"
)

var _ resource.Resource = &AzureADUserResource{}
var _ resource.ResourceWithImportState = &AzureADUserResource{}
var _ resource.ResourceWithMoveState = &AzureADUserResource{}

func NewAzureADUserResource() resource.Resource {
	return &AzureADUserResource{}
}

type AzureADUserResource struct {
	client *mssql.Client
}

type AzureADUserResourceModel struct {
	ID            types.String `tfsdk:"id"`
	DatabaseName  types.String `tfsdk:"database_name"`
	Name          types.String `tfsdk:"name"`
	ObjectID      types.String `tfsdk:"object_id"`
	DefaultSchema types.String `tfsdk:"default_schema"`
	Roles         types.Set    `tfsdk:"roles"`
}

func (r *AzureADUserResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_azuread_user"
}

func (r *AzureADUserResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an Azure AD user in a SQL Server database.",
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
				Description: "The display name of the Azure AD user.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"object_id": schema.StringAttribute{
				Description: "The Azure AD object ID of the user. Required for managed identities, optional for email-based users.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"default_schema": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("dbo"),
			},
			"roles": schema.SetAttribute{
				Description: "List of database roles to assign to this user.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (r *AzureADUserResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *AzureADUserResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data AzureADUserResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	objectID := data.ObjectID.ValueString()

	_, err := r.client.CreateAzureADUser(ctx, mssql.CreateAzureADUserOptions{
		DatabaseName:  data.DatabaseName.ValueString(),
		UserName:      data.Name.ValueString(),
		ObjectID:      objectID,
		DefaultSchema: data.DefaultSchema.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create Azure AD user", err.Error())
		return
	}

	// Assign roles if specified
	var roles []string
	if !data.Roles.IsNull() && !data.Roles.IsUnknown() {
		resp.Diagnostics.Append(data.Roles.ElementsAs(ctx, &roles, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		for _, role := range roles {
			err := r.client.AddDatabaseRoleMember(ctx, data.DatabaseName.ValueString(), role, data.Name.ValueString())
			if err != nil {
				resp.Diagnostics.AddError("Failed to assign role", fmt.Sprintf("Failed to add user to role '%s': %s", role, err.Error()))
				return
			}
		}
	}

	data.ID = types.StringValue(fmt.Sprintf("sqlserver://%s:%d/%s/%s", r.client.Hostname(), r.client.Port(), data.DatabaseName.ValueString(), data.Name.ValueString()))
	data.ObjectID = types.StringValue(objectID)

	// Set roles in state
	if len(roles) > 0 {
		roleValues := make([]attr.Value, len(roles))
		for i, role := range roles {
			roleValues[i] = types.StringValue(role)
		}
		data.Roles, _ = types.SetValue(types.StringType, roleValues)
	} else {
		data.Roles, _ = types.SetValue(types.StringType, []attr.Value{})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AzureADUserResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data AzureADUserResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	user, err := r.client.GetUser(ctx, data.DatabaseName.ValueString(), data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read Azure AD user", err.Error())
		return
	}
	if user == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	// Update ID with proper URL format
	data.ID = types.StringValue(fmt.Sprintf("sqlserver://%s:%d/%s/%s", r.client.Hostname(), r.client.Port(), data.DatabaseName.ValueString(), data.Name.ValueString()))
	data.DefaultSchema = types.StringValue(user.DefaultSchemaName)

	// Read user's roles
	roles, err := r.client.GetUserRoles(ctx, data.DatabaseName.ValueString(), data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read user roles", err.Error())
		return
	}
	roleValues := make([]attr.Value, len(roles))
	for i, role := range roles {
		roleValues[i] = types.StringValue(role)
	}
	data.Roles, _ = types.SetValue(types.StringType, roleValues)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AzureADUserResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data, state AzureADUserResourceModel
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
			resp.Diagnostics.AddError("Failed to update Azure AD user", err.Error())
			return
		}
	}

	// Update roles if changed
	if !data.Roles.Equal(state.Roles) {
		var desiredRoles, currentRoles []string
		resp.Diagnostics.Append(data.Roles.ElementsAs(ctx, &desiredRoles, false)...)
		resp.Diagnostics.Append(state.Roles.ElementsAs(ctx, &currentRoles, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		// Find roles to add and remove
		currentSet := make(map[string]bool)
		for _, role := range currentRoles {
			currentSet[role] = true
		}
		desiredSet := make(map[string]bool)
		for _, role := range desiredRoles {
			desiredSet[role] = true
		}

		// Add new roles
		for _, role := range desiredRoles {
			if !currentSet[role] {
				err := r.client.AddDatabaseRoleMember(ctx, data.DatabaseName.ValueString(), role, data.Name.ValueString())
				if err != nil {
					resp.Diagnostics.AddError("Failed to add role", fmt.Sprintf("Failed to add user to role '%s': %s", role, err.Error()))
					return
				}
			}
		}

		// Remove old roles
		for _, role := range currentRoles {
			if !desiredSet[role] {
				err := r.client.RemoveDatabaseRoleMember(ctx, data.DatabaseName.ValueString(), role, data.Name.ValueString())
				if err != nil {
					resp.Diagnostics.AddError("Failed to remove role", fmt.Sprintf("Failed to remove user from role '%s': %s", role, err.Error()))
					return
				}
			}
		}

		// Update state with sorted roles
		sort.Strings(desiredRoles)
		roleValues := make([]attr.Value, len(desiredRoles))
		for i, role := range desiredRoles {
			roleValues[i] = types.StringValue(role)
		}
		data.Roles, _ = types.SetValue(types.StringType, roleValues)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AzureADUserResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data AzureADUserResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DropUser(ctx, data.DatabaseName.ValueString(), data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete Azure AD user", err.Error())
		return
	}
}

func (r *AzureADUserResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 {
		resp.Diagnostics.AddError("Invalid import ID", "Import ID must be in format 'database_name/user_name'")
		return
	}

	user, err := r.client.GetUser(ctx, parts[0], parts[1])
	if err != nil {
		resp.Diagnostics.AddError("Failed to import Azure AD user", err.Error())
		return
	}
	if user == nil {
		resp.Diagnostics.AddError("Azure AD user not found", fmt.Sprintf("User '%s' not found in database '%s'", parts[1], parts[0]))
		return
	}

	// Use URL-based ID format
	id := fmt.Sprintf("sqlserver://%s:%d/%s/%s", r.client.Hostname(), r.client.Port(), parts[0], user.Name)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("database_name"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), user.Name)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("object_id"), "")...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("default_schema"), user.DefaultSchemaName)...)
}

// MoveState implements resource.ResourceWithMoveState.
// This allows migrating state from other providers' mssql_user resources.
func (r *AzureADUserResource) MoveState(ctx context.Context) []resource.StateMover {
	return []resource.StateMover{
		{
			// Support moving from betr-io/mssql mssql_user
			StateMover: func(ctx context.Context, req resource.MoveStateRequest, resp *resource.MoveStateResponse) {
				// Check if this is from a supported source
				if req.SourceTypeName != "mssql_user" {
					return
				}

				// Accept from betr-io/mssql provider (allow any hostname)
				if !strings.HasSuffix(req.SourceProviderAddress, "betr-io/mssql") {
					return
				}

				// Parse the source state using raw state
				// betr-io/mssql mssql_user has these attributes:
				// - database (string)
				// - username (string)
				// - default_schema (string)
				// - roles (list of strings)
				// - principal_id (number)
				// - sid (string)
				rawStateValue, err := req.SourceRawState.Unmarshal(tftypes.Object{
					AttributeTypes: map[string]tftypes.Type{
						"id":                  tftypes.String,
						"database":            tftypes.String,
						"username":            tftypes.String,
						"password":            tftypes.String,
						"login_name":          tftypes.String,
						"default_schema":      tftypes.String,
						"default_language":    tftypes.String,
						"roles":               tftypes.List{ElementType: tftypes.String},
						"principal_id":        tftypes.Number,
						"sid":                 tftypes.String,
						"authentication_type": tftypes.String,
						"object_id":           tftypes.String,
						// Server block is a list of objects with nested auth blocks
						"server": tftypes.List{ElementType: tftypes.Object{
							AttributeTypes: map[string]tftypes.Type{
								"host": tftypes.String,
								"port": tftypes.String,
								"login": tftypes.List{ElementType: tftypes.Object{
									AttributeTypes: map[string]tftypes.Type{
										"username":  tftypes.String,
										"password":  tftypes.String,
										"object_id": tftypes.String,
									},
								}},
								"azure_login": tftypes.List{ElementType: tftypes.Object{
									AttributeTypes: map[string]tftypes.Type{
										"tenant_id":     tftypes.String,
										"client_id":     tftypes.String,
										"client_secret": tftypes.String,
									},
								}},
								"azuread_default_chain_auth": tftypes.List{ElementType: tftypes.Object{
									AttributeTypes: map[string]tftypes.Type{},
								}},
								"azuread_managed_identity_auth": tftypes.List{ElementType: tftypes.Object{
									AttributeTypes: map[string]tftypes.Type{
										"user_id": tftypes.String,
									},
								}},
							},
						}},
						// Timeouts block
						"timeouts": tftypes.Object{
							AttributeTypes: map[string]tftypes.Type{
								"create": tftypes.String,
								"read":   tftypes.String,
								"update": tftypes.String,
								"delete": tftypes.String,
							},
						},
					},
				})
				if err != nil {
					resp.Diagnostics.AddError(
						"Unable to Unmarshal Source State",
						err.Error(),
					)
					return
				}

				var rawState map[string]tftypes.Value
				if err := rawStateValue.As(&rawState); err != nil {
					resp.Diagnostics.AddError(
						"Unable to Convert Source State",
						err.Error(),
					)
					return
				}

				// Extract required values
				var database *string
				if err := rawState["database"].As(&database); err != nil {
					resp.Diagnostics.AddAttributeError(
						path.Root("database_name"),
						"Unable to Convert Source State",
						err.Error(),
					)
					return
				}

				var username *string
				if err := rawState["username"].As(&username); err != nil {
					resp.Diagnostics.AddAttributeError(
						path.Root("name"),
						"Unable to Convert Source State",
						err.Error(),
					)
					return
				}

				var defaultSchema *string
				if err := rawState["default_schema"].As(&defaultSchema); err != nil {
					// Use dbo as default if not available
					dbo := "dbo"
					defaultSchema = &dbo
				}

				// Extract roles if available
				var rolesList []string
				if rawState["roles"].IsKnown() && !rawState["roles"].IsNull() {
					var rolesValues []tftypes.Value
					if err := rawState["roles"].As(&rolesValues); err == nil {
						for _, rv := range rolesValues {
							var role string
							if err := rv.As(&role); err == nil {
								rolesList = append(rolesList, role)
							}
						}
					}
				}

				// Extract object_id if available
				var objectID *string
				if rawState["object_id"].IsKnown() && !rawState["object_id"].IsNull() {
					if err := rawState["object_id"].As(&objectID); err != nil {
						objectID = nil
					}
				}

				// Build target state
				// We need to generate an ID - use a placeholder that will be updated on first read
				idPlaceholder := "migrated/pending"

				var rolesSet types.Set
				if len(rolesList) > 0 {
					roleValues := make([]attr.Value, len(rolesList))
					for i, role := range rolesList {
						roleValues[i] = types.StringValue(role)
					}
					rolesSet = types.SetValueMust(types.StringType, roleValues)
				} else {
					rolesSet = types.SetValueMust(types.StringType, []attr.Value{})
				}

				// Use object_id from source if available, otherwise empty string
				var objectIDValue types.String
				if objectID != nil && *objectID != "" {
					objectIDValue = types.StringValue(*objectID)
				} else {
					objectIDValue = types.StringValue("")
				}

				targetStateData := AzureADUserResourceModel{
					ID:            types.StringValue(idPlaceholder),
					DatabaseName:  types.StringPointerValue(database),
					Name:          types.StringPointerValue(username),
					ObjectID:      objectIDValue,
					DefaultSchema: types.StringPointerValue(defaultSchema),
					Roles:         rolesSet,
				}

				resp.Diagnostics.Append(resp.TargetState.Set(ctx, targetStateData)...)
			},
		},
	}
}
