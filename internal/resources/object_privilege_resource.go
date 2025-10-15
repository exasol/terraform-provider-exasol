package resources

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"

	"terraform-provider-exasol/internal/exasolclient"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &ObjectPrivilegeResource{}
var _ resource.ResourceWithImportState = &ObjectPrivilegeResource{}

// ObjectPrivilegeResource manages Exasol object privileges.
// Object privileges are granted on schemas, tables, views, scripts, etc.
type ObjectPrivilegeResource struct {
	db *sql.DB
}

func NewObjectPrivilegeResource() resource.Resource {
	return &ObjectPrivilegeResource{}
}

func (r *ObjectPrivilegeResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_object_privilege"
}

func (r *ObjectPrivilegeResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Grants object-level privileges to users or roles. " +
			"Object privileges include SELECT, INSERT, UPDATE, DELETE on tables; " +
			"USAGE, CREATE TABLE on schemas; etc. " +
			"You can specify a single privilege or a list of privileges. " +
			"Use 'ALL' to grant all applicable privileges for the object type.",
		Attributes: map[string]schema.Attribute{
			"grantee": schema.StringAttribute{
				Required:    true,
				Description: "User or role name receiving the privilege.",
			},
			"privileges": schema.ListAttribute{
				ElementType: types.StringType,
				Required:    true,
				Description: "List of privilege names: SELECT, INSERT, UPDATE, DELETE, USAGE, CREATE TABLE, ALTER, DROP, or ALL. Can be a single privilege or multiple.",
			},
			"object_type": schema.StringAttribute{
				Required:    true,
				Description: "Object type: SCHEMA, TABLE, VIEW, SCRIPT, FUNCTION, etc.",
			},
			"object_name": schema.StringAttribute{
				Required:    true,
				Description: "Qualified object name (e.g., 'MYSCHEMA' for schema, 'MYSCHEMA.MYTABLE' for table).",
			},
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Terraform ID in format: GRANTEE|PRIVILEGES|OBJECT_TYPE|OBJECT_NAME",
			},
		},
	}
}

func (r *ObjectPrivilegeResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	if c, ok := req.ProviderData.(*exasolclient.Client); ok {
		r.db = c.DB
	}
}

type objectPrivilegeModel struct {
	ID         types.String `tfsdk:"id"`
	Grantee    types.String `tfsdk:"grantee"`
	Privileges types.List   `tfsdk:"privileges"`
	ObjectType types.String `tfsdk:"object_type"`
	ObjectName types.String `tfsdk:"object_name"`
}

func (r *ObjectPrivilegeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan objectPrivilegeModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if r.db == nil {
		resp.Diagnostics.AddError("Database not configured", "Provider did not supply a database connection.")
		return
	}

	grantee := strings.ToUpper(plan.Grantee.ValueString())
	objectType := strings.ToUpper(plan.ObjectType.ValueString())
	objectName := qualify(plan.ObjectName.ValueString())

	// Validate identifiers
	if !isValidIdentifier(grantee) {
		resp.Diagnostics.AddError("Invalid grantee", "Grantee name contains invalid characters")
		return
	}

	// Extract privileges from list
	var privileges []string
	resp.Diagnostics.Append(plan.Privileges.ElementsAs(ctx, &privileges, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Grant each privilege
	for _, privilege := range privileges {
		priv := strings.ToUpper(privilege)
		stmt := fmt.Sprintf(`GRANT %s ON %s %s TO "%s"`, priv, objectType, objectName, grantee)
		tflog.Info(ctx, "Granting object privilege", map[string]any{"sql": stmt})
		if _, err := r.db.ExecContext(ctx, stmt); err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("GRANT %s failed", priv), err.Error())
			return
		}
	}

	plan.ID = types.StringValue(objectPrivilegeID(plan))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ObjectPrivilegeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.db == nil {
		resp.Diagnostics.AddError("Database not configured", "Provider did not supply a database connection.")
		return
	}

	var state objectPrivilegeModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	grantee := strings.ToUpper(state.Grantee.ValueString())
	objectType := strings.ToUpper(state.ObjectType.ValueString())
	objectName := strings.ToUpper(state.ObjectName.ValueString())

	// Extract privileges from list
	var privileges []string
	resp.Diagnostics.Append(state.Privileges.ElementsAs(ctx, &privileges, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check if privileges exist
	var foundPrivileges []string
	for _, privilege := range privileges {
		priv := strings.ToUpper(privilege)
		exists, err := checkObjectPrivilegeExists(ctx, r.db, grantee, priv, objectType, objectName)
		if err != nil {
			resp.Diagnostics.AddError("Read object privilege failed", err.Error())
			return
		}
		if exists {
			foundPrivileges = append(foundPrivileges, priv)
		}
	}

	// If no privileges found, remove resource
	if len(foundPrivileges) == 0 {
		resp.State.RemoveResource(ctx)
		return
	}

	// Update state with found privileges (in case some were revoked outside Terraform)
	privList, diags := types.ListValueFrom(ctx, types.StringType, foundPrivileges)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.Privileges = privList
	state.ID = types.StringValue(objectPrivilegeID(state))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ObjectPrivilegeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state objectPrivilegeModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if r.db == nil {
		resp.Diagnostics.AddError("Database not configured", "Provider did not supply a database connection.")
		return
	}

	// Extract old and new privileges
	var oldPrivileges, newPrivileges []string
	resp.Diagnostics.Append(state.Privileges.ElementsAs(ctx, &oldPrivileges, false)...)
	resp.Diagnostics.Append(plan.Privileges.ElementsAs(ctx, &newPrivileges, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	oldGrantee := strings.ToUpper(state.Grantee.ValueString())
	newGrantee := strings.ToUpper(plan.Grantee.ValueString())
	oldObjectType := strings.ToUpper(state.ObjectType.ValueString())
	newObjectType := strings.ToUpper(plan.ObjectType.ValueString())
	oldObjectName := qualify(state.ObjectName.ValueString())
	newObjectName := qualify(plan.ObjectName.ValueString())

	// If grantee, object type, or object name changed, revoke all old and grant all new
	if oldGrantee != newGrantee || oldObjectType != newObjectType || oldObjectName != newObjectName {
		// Revoke old privileges
		for _, privilege := range oldPrivileges {
			priv := strings.ToUpper(privilege)
			revokeStmt := fmt.Sprintf(`REVOKE %s ON %s %s FROM "%s"`, priv, oldObjectType, oldObjectName, oldGrantee)
			tflog.Info(ctx, "Revoking old object privilege", map[string]any{"sql": revokeStmt})
			if _, err := r.db.ExecContext(ctx, revokeStmt); err != nil {
				tflog.Warn(ctx, "REVOKE failed (privilege may not exist)", map[string]any{"error": err.Error()})
			}
		}

		// Grant new privileges
		for _, privilege := range newPrivileges {
			priv := strings.ToUpper(privilege)
			grantStmt := fmt.Sprintf(`GRANT %s ON %s %s TO "%s"`, priv, newObjectType, newObjectName, newGrantee)
			tflog.Info(ctx, "Granting new object privilege", map[string]any{"sql": grantStmt})
			if _, err := r.db.ExecContext(ctx, grantStmt); err != nil {
				resp.Diagnostics.AddError(fmt.Sprintf("GRANT %s failed", priv), err.Error())
				return
			}
		}
	} else {
		// Only privileges changed - calculate diff
		oldPrivSet := make(map[string]bool)
		for _, p := range oldPrivileges {
			oldPrivSet[strings.ToUpper(p)] = true
		}
		newPrivSet := make(map[string]bool)
		for _, p := range newPrivileges {
			newPrivSet[strings.ToUpper(p)] = true
		}

		// Revoke privileges that are no longer in the list
		for priv := range oldPrivSet {
			if !newPrivSet[priv] {
				revokeStmt := fmt.Sprintf(`REVOKE %s ON %s %s FROM "%s"`, priv, newObjectType, newObjectName, newGrantee)
				tflog.Info(ctx, "Revoking removed privilege", map[string]any{"sql": revokeStmt})
				if _, err := r.db.ExecContext(ctx, revokeStmt); err != nil {
					tflog.Warn(ctx, "REVOKE failed (privilege may not exist)", map[string]any{"error": err.Error()})
				}
			}
		}

		// Grant new privileges
		for priv := range newPrivSet {
			if !oldPrivSet[priv] {
				grantStmt := fmt.Sprintf(`GRANT %s ON %s %s TO "%s"`, priv, newObjectType, newObjectName, newGrantee)
				tflog.Info(ctx, "Granting new privilege", map[string]any{"sql": grantStmt})
				if _, err := r.db.ExecContext(ctx, grantStmt); err != nil {
					resp.Diagnostics.AddError(fmt.Sprintf("GRANT %s failed", priv), err.Error())
					return
				}
			}
		}
	}

	plan.ID = types.StringValue(objectPrivilegeID(plan))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ObjectPrivilegeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state objectPrivilegeModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if r.db == nil {
		resp.Diagnostics.AddError("Database not configured", "Provider did not supply a database connection.")
		return
	}

	grantee := strings.ToUpper(state.Grantee.ValueString())
	objectType := strings.ToUpper(state.ObjectType.ValueString())
	objectName := qualify(state.ObjectName.ValueString())

	// Extract privileges from list
	var privileges []string
	resp.Diagnostics.Append(state.Privileges.ElementsAs(ctx, &privileges, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Revoke each privilege
	for _, privilege := range privileges {
		priv := strings.ToUpper(privilege)
		stmt := fmt.Sprintf(`REVOKE %s ON %s %s FROM "%s"`, priv, objectType, objectName, grantee)
		tflog.Info(ctx, "Revoking object privilege", map[string]any{"sql": stmt})
		if _, err := r.db.ExecContext(ctx, stmt); err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("REVOKE %s failed", priv), err.Error())
		}
	}
}

func (r *ObjectPrivilegeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// ID format: GRANTEE|PRIVILEGES|OBJECT_TYPE|OBJECT_NAME
	// Privileges are comma-separated: GRANTEE|SELECT,INSERT,UPDATE|TABLE|MYSCHEMA.MYTABLE
	parts := strings.Split(req.ID, "|")
	if len(parts) != 4 {
		resp.Diagnostics.AddError("Invalid import ID",
			`Expected format: "GRANTEE|PRIVILEGE1,PRIVILEGE2|OBJECT_TYPE|OBJECT_NAME"`)
		return
	}

	privileges := strings.Split(parts[1], ",")
	var privList []attr.Value
	for _, priv := range privileges {
		privList = append(privList, types.StringValue(strings.TrimSpace(priv)))
	}

	resp.State.SetAttribute(ctx, path.Root("grantee"), parts[0])
	resp.State.SetAttribute(ctx, path.Root("privileges"), types.ListValueMust(types.StringType, privList))
	resp.State.SetAttribute(ctx, path.Root("object_type"), parts[2])
	resp.State.SetAttribute(ctx, path.Root("object_name"), parts[3])
	resp.State.SetAttribute(ctx, path.Root("id"), req.ID)
}

func objectPrivilegeID(m objectPrivilegeModel) string {
	grantee := strings.ToUpper(m.Grantee.ValueString())
	objectType := strings.ToUpper(m.ObjectType.ValueString())
	objectName := strings.ToUpper(m.ObjectName.ValueString())

	// Extract and sort privileges for consistent ID
	var privileges []string
	m.Privileges.ElementsAs(context.Background(), &privileges, false)
	for i, p := range privileges {
		privileges[i] = strings.ToUpper(p)
	}
	sort.Strings(privileges)
	privilegesStr := strings.Join(privileges, ",")

	return fmt.Sprintf("%s|%s|%s|%s", grantee, privilegesStr, objectType, objectName)
}

func checkObjectPrivilegeExists(ctx context.Context, db *sql.DB, grantee, privilege, objectType, objectName string) (bool, error) {
	tflog.Debug(ctx, "Checking object privilege existence", map[string]any{
		"grantee":     grantee,
		"privilege":   privilege,
		"object_type": objectType,
		"object_name": objectName,
	})

	// Special handling for "ALL" privilege
	if privilege == "ALL" {
		// First, try to find "ALL" privilege directly
		query := `SELECT 1 FROM EXA_DBA_OBJ_PRIVS WHERE GRANTEE = ? AND PRIVILEGE = 'ALL' AND OBJECT_TYPE = ? AND OBJECT_NAME = ?`
		var dummy int
		err := db.QueryRowContext(ctx, query, grantee, objectType, objectName).Scan(&dummy)
		if err == nil {
			tflog.Debug(ctx, "Object privilege 'ALL' found in EXA_DBA_OBJ_PRIVS")
			return true, nil
		}
		if err != sql.ErrNoRows {
			return false, err
		}

		// If "ALL" is not found directly, check if any individual privileges exist
		countQuery := `SELECT COUNT(*) FROM EXA_DBA_OBJ_PRIVS WHERE GRANTEE = ? AND OBJECT_TYPE = ? AND OBJECT_NAME = ?`
		var count int
		err = db.QueryRowContext(ctx, countQuery, grantee, objectType, objectName).Scan(&count)
		if err != nil {
			return false, err
		}
		if count > 0 {
			tflog.Debug(ctx, "Object privileges found (ALL may have been expanded)", map[string]any{"count": count})
			return true, nil
		}
		return false, nil
	}

	// For non-ALL privileges, query directly
	query := `SELECT 1 FROM EXA_DBA_OBJ_PRIVS WHERE GRANTEE = ? AND PRIVILEGE = ? AND OBJECT_TYPE = ? AND OBJECT_NAME = ?`
	var dummy int
	err := db.QueryRowContext(ctx, query, grantee, privilege, objectType, objectName).Scan(&dummy)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}