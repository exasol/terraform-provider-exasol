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

var _ resource.Resource = &GrantResource{}
var _ resource.ResourceWithImportState = &GrantResource{}

// GrantResource implements a generic Exasol GRANT/REVOKE resource.
type GrantResource struct {
	db *sql.DB
}

func NewGrantResource() resource.Resource { return &GrantResource{} }

func (r *GrantResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_grant"
}

func (r *GrantResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Generic Exasol GRANT resource supporting SYSTEM privileges, OBJECT privileges, and ROLE grants.\n\n" +
			"For role grants, set privilege_type to either SYSTEM or OBJECT with object_type='ROLE'. " +
			"When granting a role, the privilege field should contain the role name, and for OBJECT type, " +
			"the object_name should also contain the role name.",
		Attributes: map[string]schema.Attribute{
			"grantee_name": schema.StringAttribute{
				Required:    true,
				Description: "User or role name that receives the privilege or role.",
			},
			"privilege_type": schema.StringAttribute{
				Required:    true,
				Description: `Either "SYSTEM" or "OBJECT". For role grants, use either type with object_type="ROLE".`,
			},
			"privilege": schema.StringAttribute{
				Required:    true,
				Description: "Privilege name (e.g. USAGE, SELECT, CREATE ANY TABLE...) or role name for role grants.",
			},
			"object_type": schema.StringAttribute{
				Optional:    true,
				Description: `Object type for OBJECT privileges (e.g. SCHEMA, TABLE, VIEW). Use "ROLE" for role grants.`,
			},
			"object_name": schema.StringAttribute{
				Optional:    true,
				Description: "Qualified object name for OBJECT privileges (e.g. MYSCHEMA.MYTABLE or MYSCHEMA). For role grants with OBJECT privilege_type, this should contain the role name.",
			},
			"with_admin_option": schema.BoolAttribute{
				Optional:    true,
				Description: "Grants the privilege/role with ADMIN OPTION. Applies to SYSTEM privileges and role grants.",
			},
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Synthetic ID representing the granted privilege or role.",
			},
		},
	}
}

func (r *GrantResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	if c, ok := req.ProviderData.(*exasolclient.Client); ok {
		r.db = c.DB
	}
}

type grantModel struct {
	ID              types.String `tfsdk:"id"`
	GranteeName     types.String `tfsdk:"grantee_name"`
	PrivilegeType   types.String `tfsdk:"privilege_type"`
	Privilege       types.String `tfsdk:"privilege"`
	ObjectType      types.String `tfsdk:"object_type"`
	ObjectName      types.String `tfsdk:"object_name"`
	WithAdminOption types.Bool   `tfsdk:"with_admin_option"`
}

func (r *GrantResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.db == nil {
		resp.Diagnostics.AddError("Database not configured", "Provider did not supply a database connection.")
		return
	}

	var plan grantModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	sqlGrant, err := buildGrantSQL(plan)
	if err != nil {
		resp.Diagnostics.AddError("Invalid grant", err.Error())
		return
	}
	tflog.Info(ctx, "Executing GRANT", map[string]any{"sql": sqlGrant})
	if _, err := r.db.ExecContext(ctx, sqlGrant); err != nil {
		resp.Diagnostics.AddError("GRANT failed", err.Error())
		return
	}

	plan.ID = types.StringValue(idForGrant(plan))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *GrantResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.db == nil {
		resp.Diagnostics.AddError("Database not configured", "Provider did not supply a database connection.")
		return
	}

	var state grantModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	exists, err := checkGrantExists(ctx, r.db, state)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read grant", err.Error())
		return
	}
	if !exists {
		resp.State.RemoveResource(ctx)
		return
	}

	// Re-assert ID to ensure Terraform never sees it as unknown
	state.ID = types.StringValue(idForGrant(state))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *GrantResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.db == nil {
		resp.Diagnostics.AddError("Database not configured", "Provider did not supply a database connection.")
		return
	}

	var plan, state grantModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check if this is a schema object rename - Exasol handles grants automatically
	if isSchemaObjectRename(plan, state) {
		tflog.Info(ctx, "Schema rename detected - skipping grant update as database handles it automatically",
			map[string]any{
				"old_object_name": state.ObjectName.ValueString(),
				"new_object_name": plan.ObjectName.ValueString(),
			})

		// Update only the Terraform state
		plan.ID = types.StringValue(idForGrant(plan))
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		return
	}

	oldID := idForGrant(state)
	newID := idForGrant(plan)

	if oldID != newID {
		// First revoke the old grant
		sqlRevoke, err := buildRevokeSQL(state)
		if err != nil {
			resp.Diagnostics.AddError("Invalid revoke statement", err.Error())
			return
		}

		tflog.Info(ctx, "Revoking old grant", map[string]any{"sql": sqlRevoke})
		if _, err := r.db.ExecContext(ctx, sqlRevoke); err != nil {
			resp.Diagnostics.AddError("REVOKE failed", err.Error())
			return
		}

		// Then create the new grant
		sqlGrant, err := buildGrantSQL(plan)
		if err != nil {
			resp.Diagnostics.AddError("Invalid grant statement", err.Error())
			return
		}

		tflog.Info(ctx, "Creating new grant", map[string]any{"sql": sqlGrant})
		if _, err := r.db.ExecContext(ctx, sqlGrant); err != nil {
			resp.Diagnostics.AddError("GRANT failed", err.Error())
			return
		}
	}

	plan.ID = types.StringValue(newID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// isSchemaObjectRename checks if this update is just a schema rename where
// only the object_name changed for a SCHEMA object type grant
func isSchemaObjectRename(plan, state grantModel) bool {
	// Must be OBJECT privilege on SCHEMA
	if !strings.EqualFold(plan.PrivilegeType.ValueString(), "OBJECT") ||
		!strings.EqualFold(state.PrivilegeType.ValueString(), "OBJECT") ||
		!strings.EqualFold(plan.ObjectType.ValueString(), "SCHEMA") ||
		!strings.EqualFold(state.ObjectType.ValueString(), "SCHEMA") {
		return false
	}

	// Only object_name should have changed
	return plan.GranteeName.ValueString() == state.GranteeName.ValueString() &&
		plan.Privilege.ValueString() == state.Privilege.ValueString() &&
		plan.WithAdminOption.ValueBool() == state.WithAdminOption.ValueBool() &&
		plan.ObjectName.ValueString() != state.ObjectName.ValueString()
}

func (r *GrantResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Serialize delete operations to prevent transaction collision errors
	lockDelete()
	defer unlockDelete()

	if r.db == nil {
		resp.Diagnostics.AddError("Database not configured", "Provider did not supply a database connection.")
		return
	}

	var state grantModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	sqlRevoke, err := buildRevokeSQL(state)
	if err != nil {
		resp.Diagnostics.AddError("Invalid revoke", err.Error())
		return
	}
	if _, err := r.db.ExecContext(ctx, sqlRevoke); err != nil {
		resp.Diagnostics.AddError("REVOKE failed", err.Error())
	}
}

func (r *GrantResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// ID format: GRANTEE|PRIVTYPE|PRIV|OBJTYPE|OBJNAME|WITHADMIN
	parts := strings.Split(req.ID, "|")
	if len(parts) != 6 {
		resp.Diagnostics.AddError("Invalid import ID",
			`Expected "GRANTEE|PRIVTYPE|PRIV|OBJTYPE|OBJNAME|WITHADMIN"`)
		return
	}
	resp.State.SetAttribute(ctx, path.Root("grantee_name"), parts[0])
	resp.State.SetAttribute(ctx, path.Root("privilege_type"), parts[1])
	resp.State.SetAttribute(ctx, path.Root("privilege"), parts[2])
	if parts[3] != "" {
		resp.State.SetAttribute(ctx, path.Root("object_type"), parts[3])
	}
	if parts[4] != "" {
		resp.State.SetAttribute(ctx, path.Root("object_name"), parts[4])
	}
	resp.State.SetAttribute(ctx, path.Root("with_admin_option"), strings.EqualFold(parts[5], "true"))
	resp.State.SetAttribute(ctx, path.Root("id"), req.ID)
}

func idForGrant(m grantModel) string {
	grantee := strings.ToUpper(m.GranteeName.ValueString())
	pt := strings.ToUpper(m.PrivilegeType.ValueString())
	priv := strings.ToUpper(m.Privilege.ValueString())
	objType := strings.ToUpper(m.ObjectType.ValueString())
	objName := m.ObjectName.ValueString()
	withAdmin := fmt.Sprintf("%t", m.WithAdminOption.ValueBool())

	return strings.Join([]string{
		grantee, pt, priv, objType, objName, withAdmin,
	}, "|")
}

func buildGrantSQL(m grantModel) (string, error) {
	granteeName := strings.ToUpper(m.GranteeName.ValueString())

	// Validate grantee name
	if !isValidIdentifier(granteeName) {
		return "", fmt.Errorf("invalid grantee name %q: must start with a letter and contain only letters, digits, and underscores", m.GranteeName.ValueString())
	}

	grantee := fmt.Sprintf(`"%s"`, granteeName)
	priv := strings.ToUpper(m.Privilege.ValueString())

	switch strings.ToUpper(m.PrivilegeType.ValueString()) {
	case "SYSTEM":
		sql := fmt.Sprintf(`GRANT %s TO %s`, priv, grantee)
		if m.WithAdminOption.ValueBool() {
			sql += " WITH ADMIN OPTION"
		}
		return sql, nil
	case "OBJECT":
		if m.ObjectType.IsNull() || m.ObjectName.IsNull() {
			return "", fmt.Errorf("object_type and object_name are required for OBJECT privileges")
		}
		objType := strings.ToUpper(m.ObjectType.ValueString())
		objName := qualify(m.ObjectName.ValueString())
		return fmt.Sprintf(`GRANT %s ON %s %s TO %s`, priv, objType, objName, grantee), nil
	default:
		return "", fmt.Errorf("privilege_type must be SYSTEM or OBJECT")
	}
}

func buildRevokeSQL(m grantModel) (string, error) {
	granteeName := strings.ToUpper(m.GranteeName.ValueString())

	// Validate grantee name
	if !isValidIdentifier(granteeName) {
		return "", fmt.Errorf("invalid grantee name %q: must start with a letter and contain only letters, digits, and underscores", m.GranteeName.ValueString())
	}

	grantee := fmt.Sprintf(`"%s"`, granteeName)
	priv := strings.ToUpper(m.Privilege.ValueString())

	switch strings.ToUpper(m.PrivilegeType.ValueString()) {
	case "SYSTEM":
		return fmt.Sprintf(`REVOKE %s FROM %s`, priv, grantee), nil
	case "OBJECT":
		if m.ObjectType.IsNull() || m.ObjectName.IsNull() {
			return "", fmt.Errorf("object_type and object_name are required for OBJECT privileges")
		}
		objType := strings.ToUpper(m.ObjectType.ValueString())
		objName := qualify(m.ObjectName.ValueString())
		return fmt.Sprintf(`REVOKE %s ON %s %s FROM %s`, priv, objType, objName, grantee), nil
	default:
		return "", fmt.Errorf("privilege_type must be SYSTEM or OBJECT")
	}
}

func checkGrantExists(ctx context.Context, db *sql.DB, m grantModel) (bool, error) {
	granteeName := strings.ToUpper(m.GranteeName.ValueString())
	privilege := strings.ToUpper(m.Privilege.ValueString())

	switch strings.ToUpper(m.PrivilegeType.ValueString()) {
	case "SYSTEM":
		// Check if this is actually a ROLE grant (when object_type = "ROLE")
		if !m.ObjectType.IsNull() && strings.EqualFold(m.ObjectType.ValueString(), "ROLE") {
			// This is a role grant, not a system privilege
			// Query EXA_DBA_ROLE_PRIVS for role assignments
			query := `SELECT 1 FROM EXA_DBA_ROLE_PRIVS WHERE GRANTEE = ? AND GRANTED_ROLE = ?`
			var dummy int
			err := db.QueryRowContext(ctx, query, granteeName, privilege).Scan(&dummy)
			if err == sql.ErrNoRows {
				return false, nil
			}
			if err != nil {
				return false, err
			}
			return true, nil
		}

		// Query EXA_DBA_SYS_PRIVS for system privileges
		query := `SELECT 1 FROM EXA_DBA_SYS_PRIVS WHERE GRANTEE = ? AND PRIVILEGE = ?`
		var dummy int
		err := db.QueryRowContext(ctx, query, granteeName, privilege).Scan(&dummy)
		if err == sql.ErrNoRows {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		return true, nil

	case "OBJECT":
		if m.ObjectType.IsNull() || m.ObjectName.IsNull() {
			return false, fmt.Errorf("object_type and object_name are required for OBJECT privileges")
		}

		objType := strings.ToUpper(m.ObjectType.ValueString())
		objName := strings.ToUpper(m.ObjectName.ValueString())

		// Special handling for ROLE type - this is actually a role grant
		if strings.EqualFold(objType, "ROLE") {
			// Query EXA_DBA_ROLE_PRIVS for role assignments
			// In this case, object_name contains the role name
			query := `SELECT 1 FROM EXA_DBA_ROLE_PRIVS WHERE GRANTEE = ? AND GRANTED_ROLE = ?`
			var dummy int
			err := db.QueryRowContext(ctx, query, granteeName, objName).Scan(&dummy)
			if err == sql.ErrNoRows {
				return false, nil
			}
			if err != nil {
				return false, err
			}
			return true, nil
		}

		// For object privileges, query EXA_DBA_OBJ_PRIVS
		// The object name might be schema-qualified (e.g., "SCHEMA.TABLE")
		// We need to handle both single identifiers and qualified names

		tflog.Debug(ctx, "Checking object privilege existence", map[string]any{
			"grantee":     granteeName,
			"privilege":   privilege,
			"object_type": objType,
			"object_name": objName,
		})

		// Special handling for "ALL" privilege
		// Exasol may expand "ALL" into individual privileges or store it as-is
		// We need to check both possibilities
		if privilege == "ALL" {
			// First, try to find "ALL" privilege directly
			query := `SELECT 1 FROM EXA_DBA_OBJ_PRIVS WHERE GRANTEE = ? AND PRIVILEGE = 'ALL' AND OBJECT_TYPE = ? AND OBJECT_NAME = ?`
			var dummy int
			err := db.QueryRowContext(ctx, query, granteeName, objType, objName).Scan(&dummy)
			if err == nil {
				tflog.Debug(ctx, "Object privilege 'ALL' found in EXA_DBA_OBJ_PRIVS")
				return true, nil
			}
			if err != sql.ErrNoRows {
				tflog.Error(ctx, "Error querying EXA_DBA_OBJ_PRIVS for ALL", map[string]any{"error": err.Error()})
				return false, err
			}

			// If "ALL" is not found directly, check if any individual privileges exist
			// This covers the case where "ALL" was expanded into individual privileges
			countQuery := `SELECT COUNT(*) FROM EXA_DBA_OBJ_PRIVS WHERE GRANTEE = ? AND OBJECT_TYPE = ? AND OBJECT_NAME = ?`
			var count int
			err = db.QueryRowContext(ctx, countQuery, granteeName, objType, objName).Scan(&count)
			if err != nil {
				tflog.Error(ctx, "Error counting privileges in EXA_DBA_OBJ_PRIVS", map[string]any{"error": err.Error()})
				return false, err
			}
			if count > 0 {
				tflog.Debug(ctx, "Object privileges found (ALL may have been expanded)", map[string]any{"count": count})
				return true, nil
			}
			tflog.Debug(ctx, "No object privileges found for grantee")
			return false, nil
		}

		// For non-ALL privileges, query directly
		query := `SELECT 1 FROM EXA_DBA_OBJ_PRIVS WHERE GRANTEE = ? AND PRIVILEGE = ? AND OBJECT_TYPE = ? AND OBJECT_NAME = ?`
		var dummy int
		err := db.QueryRowContext(ctx, query, granteeName, privilege, objType, objName).Scan(&dummy)
		if err == sql.ErrNoRows {
			tflog.Debug(ctx, "Object privilege not found in EXA_DBA_OBJ_PRIVS")
			return false, nil
		}
		if err != nil {
			tflog.Error(ctx, "Error querying EXA_DBA_OBJ_PRIVS", map[string]any{"error": err.Error()})
			return false, err
		}
		tflog.Debug(ctx, "Object privilege found in EXA_DBA_OBJ_PRIVS")
		return true, nil

	default:
		return false, fmt.Errorf("privilege_type must be SYSTEM or OBJECT")
	}
}
