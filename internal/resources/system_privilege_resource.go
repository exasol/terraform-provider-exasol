package resources

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"terraform-provider-exasol/internal/exasolclient"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &SystemPrivilegeResource{}
var _ resource.ResourceWithImportState = &SystemPrivilegeResource{}

// SystemPrivilegeResource manages Exasol system privileges.
// System privileges include: CREATE SESSION, CREATE TABLE, CREATE SCHEMA, etc.
type SystemPrivilegeResource struct {
	db *sql.DB
}

func NewSystemPrivilegeResource() resource.Resource {
	return &SystemPrivilegeResource{}
}

func (r *SystemPrivilegeResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_system_privilege"
}

func (r *SystemPrivilegeResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Grants system-level privileges to users or roles. " +
			"System privileges include CREATE SESSION, CREATE TABLE, CREATE SCHEMA, USE ANY SCHEMA, etc.",
		Attributes: map[string]schema.Attribute{
			"grantee": schema.StringAttribute{
				Required:    true,
				Description: "User or role name receiving the privilege.",
			},
			"privilege": schema.StringAttribute{
				Required:    true,
				Description: "System privilege name (e.g., 'CREATE SESSION', 'CREATE TABLE', 'USE ANY SCHEMA').",
			},
			"with_admin_option": schema.BoolAttribute{
				Optional:    true,
				Description: "Grant the privilege with ADMIN OPTION, allowing the grantee to grant this privilege to others.",
			},
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Terraform ID in format: GRANTEE|PRIVILEGE|ADMIN_OPTION",
			},
		},
	}
}

func (r *SystemPrivilegeResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	if c, ok := req.ProviderData.(*exasolclient.Client); ok {
		r.db = c.DB
	}
}

type systemPrivilegeModel struct {
	ID              types.String `tfsdk:"id"`
	Grantee         types.String `tfsdk:"grantee"`
	Privilege       types.String `tfsdk:"privilege"`
	WithAdminOption types.Bool   `tfsdk:"with_admin_option"`
}

func (r *SystemPrivilegeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan systemPrivilegeModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if r.db == nil {
		resp.Diagnostics.AddError("Database not configured", "Provider did not supply a database connection.")
		return
	}

	grantee := strings.ToUpper(plan.Grantee.ValueString())
	privilege := strings.ToUpper(plan.Privilege.ValueString())

	// Validate identifiers
	if !isValidIdentifier(grantee) {
		resp.Diagnostics.AddError("Invalid grantee", "Grantee name contains invalid characters")
		return
	}

	// Build GRANT statement
	stmt := fmt.Sprintf(`GRANT %s TO "%s"`, privilege, grantee)
	if !plan.WithAdminOption.IsNull() && plan.WithAdminOption.ValueBool() {
		stmt += " WITH ADMIN OPTION"
	}

	tflog.Info(ctx, "Granting system privilege", map[string]any{"sql": stmt})
	if _, err := r.db.ExecContext(ctx, stmt); err != nil {
		resp.Diagnostics.AddError("GRANT failed", err.Error())
		return
	}

	plan.ID = types.StringValue(systemPrivilegeID(plan))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *SystemPrivilegeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.db == nil {
		resp.Diagnostics.AddError("Database not configured", "Provider did not supply a database connection.")
		return
	}

	var state systemPrivilegeModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	grantee := strings.ToUpper(state.Grantee.ValueString())
	privilege := strings.ToUpper(state.Privilege.ValueString())

	// Check if privilege exists in EXA_DBA_SYS_PRIVS
	query := `SELECT ADMIN_OPTION FROM EXA_DBA_SYS_PRIVS WHERE GRANTEE = ? AND PRIVILEGE = ?`
	var adminOption string
	err := r.db.QueryRowContext(ctx, query, grantee, privilege).Scan(&adminOption)
	if err == sql.ErrNoRows {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Read system privilege failed", err.Error())
		return
	}

	// Only update with_admin_option if it was explicitly set in the configuration
	// If it's null in the plan, keep it null to avoid phantom diffs
	if !state.WithAdminOption.IsNull() {
		state.WithAdminOption = types.BoolValue(adminOption == "TRUE")
	}
	state.ID = types.StringValue(systemPrivilegeID(state))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *SystemPrivilegeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state systemPrivilegeModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if r.db == nil {
		resp.Diagnostics.AddError("Database not configured", "Provider did not supply a database connection.")
		return
	}

	// If grantee or privilege changed, need to revoke old and grant new
	if plan.Grantee.ValueString() != state.Grantee.ValueString() ||
		plan.Privilege.ValueString() != state.Privilege.ValueString() {

		// Revoke old privilege
		oldGrantee := strings.ToUpper(state.Grantee.ValueString())
		oldPrivilege := strings.ToUpper(state.Privilege.ValueString())
		revokeStmt := fmt.Sprintf(`REVOKE %s FROM "%s"`, oldPrivilege, oldGrantee)
		tflog.Info(ctx, "Revoking old system privilege", map[string]any{"sql": revokeStmt})
		if _, err := r.db.ExecContext(ctx, revokeStmt); err != nil {
			resp.Diagnostics.AddError("REVOKE failed", err.Error())
			return
		}

		// Grant new privilege
		newGrantee := strings.ToUpper(plan.Grantee.ValueString())
		newPrivilege := strings.ToUpper(plan.Privilege.ValueString())
		grantStmt := fmt.Sprintf(`GRANT %s TO "%s"`, newPrivilege, newGrantee)
		if !plan.WithAdminOption.IsNull() && plan.WithAdminOption.ValueBool() {
			grantStmt += " WITH ADMIN OPTION"
		}
		tflog.Info(ctx, "Granting new system privilege", map[string]any{"sql": grantStmt})
		if _, err := r.db.ExecContext(ctx, grantStmt); err != nil {
			resp.Diagnostics.AddError("GRANT failed", err.Error())
			return
		}
	} else if plan.WithAdminOption.ValueBool() != state.WithAdminOption.ValueBool() {
		// Only admin option changed - need to revoke and re-grant
		grantee := strings.ToUpper(plan.Grantee.ValueString())
		privilege := strings.ToUpper(plan.Privilege.ValueString())

		revokeStmt := fmt.Sprintf(`REVOKE %s FROM "%s"`, privilege, grantee)
		tflog.Info(ctx, "Revoking system privilege to update admin option", map[string]any{"sql": revokeStmt})
		if _, err := r.db.ExecContext(ctx, revokeStmt); err != nil {
			resp.Diagnostics.AddError("REVOKE failed", err.Error())
			return
		}

		grantStmt := fmt.Sprintf(`GRANT %s TO "%s"`, privilege, grantee)
		if !plan.WithAdminOption.IsNull() && plan.WithAdminOption.ValueBool() {
			grantStmt += " WITH ADMIN OPTION"
		}
		tflog.Info(ctx, "Re-granting system privilege with updated admin option", map[string]any{"sql": grantStmt})
		if _, err := r.db.ExecContext(ctx, grantStmt); err != nil {
			resp.Diagnostics.AddError("GRANT failed", err.Error())
			return
		}
	}

	plan.ID = types.StringValue(systemPrivilegeID(plan))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *SystemPrivilegeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state systemPrivilegeModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if r.db == nil {
		resp.Diagnostics.AddError("Database not configured", "Provider did not supply a database connection.")
		return
	}

	grantee := strings.ToUpper(state.Grantee.ValueString())
	privilege := strings.ToUpper(state.Privilege.ValueString())
	stmt := fmt.Sprintf(`REVOKE %s FROM "%s"`, privilege, grantee)

	tflog.Info(ctx, "Revoking system privilege", map[string]any{"sql": stmt})
	if _, err := r.db.ExecContext(ctx, stmt); err != nil {
		resp.Diagnostics.AddError("REVOKE failed", err.Error())
	}
}

func (r *SystemPrivilegeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// ID format: GRANTEE|PRIVILEGE|ADMIN_OPTION
	parts := strings.Split(req.ID, "|")
	if len(parts) != 3 {
		resp.Diagnostics.AddError("Invalid import ID",
			`Expected format: "GRANTEE|PRIVILEGE|true|false"`)
		return
	}
	resp.State.SetAttribute(ctx, path.Root("grantee"), parts[0])
	resp.State.SetAttribute(ctx, path.Root("privilege"), parts[1])
	resp.State.SetAttribute(ctx, path.Root("with_admin_option"), strings.EqualFold(parts[2], "true"))
	resp.State.SetAttribute(ctx, path.Root("id"), req.ID)
}

func systemPrivilegeID(m systemPrivilegeModel) string {
	grantee := strings.ToUpper(m.Grantee.ValueString())
	privilege := strings.ToUpper(m.Privilege.ValueString())
	adminOption := "false"
	if !m.WithAdminOption.IsNull() && m.WithAdminOption.ValueBool() {
		adminOption = "true"
	}
	return fmt.Sprintf("%s|%s|%s", grantee, privilege, adminOption)
}