# Release Notes - Exasol Terraform Provider v0.2.0

## Summary

Major release with improved grant management, new privilege resources, security fixes, and enhanced usability. This release introduces breaking changes for better clarity and includes critical security improvements.

---

## 🎉 New Features

### New Grant Resources (Breaking Change)
Introduced three purpose-specific resources to replace the confusing `exasol_grant` resource:

- **`exasol_system_privilege`** - Grants system-level privileges (CREATE SESSION, CREATE TABLE, etc.)
- **`exasol_object_privilege`** - Grants object-level privileges on schemas, tables, views, etc.
- **`exasol_role_grant`** - Grants roles to users or other roles

**Benefits:**
- Self-documenting resource names
- Clearer attribute structure
- Type-safe configurations
- Better validation and error messages

**Migration Guide:** See [MIGRATION_GUIDE.md](MIGRATION_GUIDE.md) for step-by-step instructions.

### Multiple Privileges Support
The `exasol_object_privilege` resource now supports granting multiple privileges in a single resource:

```hcl
resource "exasol_object_privilege" "table_access" {
  grantee     = "ANALYST_ROLE"
  privileges  = ["SELECT", "INSERT", "UPDATE", "DELETE"]
  object_type = "TABLE"
  object_name = "MYSCHEMA.MYTABLE"
}
```

**Features:**
- Reduces configuration repetition
- Intelligent updates (only grants/revokes changed privileges)
- Works with single or multiple privileges
- Clear declaration of related privileges

### Connection Management
Added `exasol_connection` resource for managing external connections:

```hcl
resource "exasol_connection" "s3_bucket" {
  name     = "MY_S3_BUCKET"
  to       = "https://my-bucket.s3.us-east-1.amazonaws.com"
  user     = "access_key"
  password = "secret_key"
}
```

Supports S3, FTP, JDBC, Oracle, and other connection types.

---

## 🔒 Security Fixes

### Critical: SQL Injection Vulnerabilities (CVE-TBD)
**Severity:** Critical
**Impact:** All resources executing SQL statements

**Fixed vulnerabilities in:**
- User resource (password handling, LDAP DN, OpenID subject)
- Role resource (role names)
- Schema resource (schema names)
- Grant resource (grantee names, object names)

**Improvements:**
- Added identifier validation using regex `^[A-Z][A-Z0-9_]*$`
- Implemented proper SQL escaping for all user inputs
- Added password sanitization in logs
- Enhanced error messages for invalid identifiers

**Recommendation:** Upgrade immediately if using provider in production.

### Security Enhancements
- Created centralized security functions in `security.go`
- Added `isValidIdentifier()` for input validation
- Added `sanitizeLogSQL()` for password redaction in logs
- Added `escapeStringLiteral()` and `escapeIdentifierLiteral()` for SQL escaping

---

## 🐛 Bug Fixes

### Grant Detection Issues
**Problem:** Grants were being recreated on every Terraform run.

**Fixed:**
1. **Role grants not detected** - Now properly queries `EXA_DBA_ROLE_PRIVS` table
2. **"ALL" privilege expansion** - Handles both direct "ALL" storage and expanded individual privileges
3. **Phantom updates** - Fixed `with_admin_option` causing unnecessary updates when not specified

**Impact:** Eliminates unnecessary grant revoke/re-grant cycles, improving stability.

### Provider Address Format
**Problem:** Provider failed to start with error: "expected hostname/namespace/type format"

**Fixed:** Updated provider address from `exasol/bi-terraform-provider-exasol` to `registry.terraform.io/exasol/bi-terraform-provider-exasol`

### Database Connection Checks
Added missing nil checks in all resource CRUD operations to prevent panics when database connection is not initialized.

---

## 💡 Improvements

### Documentation
- Updated [README.md](README.md) with new resource examples
- Created comprehensive [MIGRATION_GUIDE.md](MIGRATION_GUIDE.md)
- Added [examples/privileges/](examples/privileges/) with all grant types
- Added [examples/connections/](examples/connections/) for various connection types
- Enhanced inline documentation for all resources

### Code Quality
- Consistent error handling across all resources
- Improved logging with context-specific messages
- Better validation with clear error messages
- Standardized case handling for identifiers

### Developer Experience
- Clear resource naming (intent is obvious)
- Fewer attributes to remember
- Better autocomplete in IDEs
- Improved Terraform plan output readability

---

## 📋 Breaking Changes

### Grant Resources Restructured
The `exasol_grant` resource is now **legacy** (still supported but deprecated).

**Action Required:** Migrate to new resources:
- System privileges → `exasol_system_privilege`
- Object privileges → `exasol_object_privilege`
- Role grants → `exasol_role_grant`

**Attribute Changes:**
- `grantee_name` → `grantee` (all resources)
- `privilege` → `privileges` (list) in `exasol_object_privilege`
- Removed `privilege_type` (implicit in resource type)
- Removed confusing `object_type = "ROLE"` pattern

### Provider Configuration
No changes to provider configuration - remains backward compatible.

---

## 🔄 Deprecations

### Deprecated Resources
- `exasol_grant` - Use specific grant resources instead

**Timeline:**
- v0.2.0: Marked as deprecated, still functional
- v0.3.0: Will show warnings
- v1.0.0: Planned removal

---

## 📦 Installation

### Terraform Registry
```hcl
terraform {
  required_providers {
    exasol = {
      source  = "registry.terraform.io/exasol/bi-terraform-provider-exasol"
      version = "~> 0.2.0"
    }
  }
}
```

### Local Development
```bash
make install-local
```

---

## 🧪 Testing

All changes have been:
- ✅ Code compiled successfully
- ✅ Provider builds without errors
- ✅ Manual testing with real Exasol database
- ✅ Examples validated

**Note:** Automated test suite coming in v0.3.0

---

## 📊 Statistics

- **7 resources total**: user, role, schema, connection, grant (legacy), system_privilege, object_privilege, role_grant
- **New files:** 4 new resource files, 1 security utilities file
- **Fixed vulnerabilities:** 12+ SQL injection points
- **Lines of code:** ~3,500
- **Examples:** 4 comprehensive example directories

---

## 🙏 Acknowledgments

Thanks to all users who reported issues and provided feedback on the initial release.

---

## 📚 Additional Resources

- [README.md](README.md) - Quick start guide
- [MIGRATION_GUIDE.md](MIGRATION_GUIDE.md) - Detailed migration instructions
- [examples/](examples/) - Working examples for all resources
- [GitHub Issues](https://github.com/exasol/terraform-provider-exasol/issues) - Report bugs or request features

---

## 🔜 Coming in v0.3.0

- Automated test suite (unit + integration tests)
- Data sources for reading existing resources
- Enhanced connection pooling configuration
- Additional object types support
- Performance optimizations

---

**Full Changelog:** v0.1.0...v0.2.0