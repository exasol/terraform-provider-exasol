# Migration Guide: exasol_grant → New Grant Resources

## Overview

The `exasol_grant` resource has been deprecated in favor of three new, purpose-specific resources that are clearer and easier to use:

1. **`exasol_system_privilege`** - For system-level privileges
2. **`exasol_object_privilege`** - For object-level privileges
3. **`exasol_role_grant`** - For granting roles to users/roles

## Why the Change?

The old `exasol_grant` resource was confusing because:
- It used `privilege_type` + `object_type` combinations that were unclear
- Setting `object_type = "ROLE"` for role grants was counter-intuitive
- The same `with_admin_option` attribute had different meanings in different contexts
- It was hard to remember which attributes were required for which grant type

The new resources are **self-documenting** and make it clear what you're granting.

## Migration Examples

### Example 1: System Privilege

**OLD (exasol_grant):**
```hcl
resource "exasol_grant" "create_session" {
  grantee_name      = "ANALYST_ROLE"
  privilege_type    = "SYSTEM"
  privilege         = "CREATE SESSION"
  with_admin_option = false
}
```

**NEW (exasol_system_privilege):**
```hcl
resource "exasol_system_privilege" "create_session" {
  grantee   = "ANALYST_ROLE"
  privilege = "CREATE SESSION"
  # with_admin_option defaults to false, so can be omitted
}
```

**Changes:**
- `grantee_name` → `grantee`
- Removed `privilege_type` (implicit in resource type)
- `with_admin_option` is optional and defaults to `false`

---

### Example 2: Object Privilege on Schema

**OLD (exasol_grant):**
```hcl
resource "exasol_grant" "schema_usage" {
  grantee_name   = "ANALYST_ROLE"
  privilege_type = "OBJECT"
  privilege      = "ALL"
  object_type    = "SCHEMA"
  object_name    = "DBT"
}
```

**NEW (exasol_object_privilege):**
```hcl
resource "exasol_object_privilege" "schema_usage" {
  grantee     = "ANALYST_ROLE"
  privilege   = "ALL"
  object_type = "SCHEMA"
  object_name = "DBT"
}
```

**Changes:**
- `grantee_name` → `grantee`
- Removed `privilege_type` (implicit in resource type)
- All other attributes remain the same

---

### Example 3: Object Privilege on Table

**OLD (exasol_grant):**
```hcl
resource "exasol_grant" "select_table" {
  grantee_name   = "ANALYST_ROLE"
  privilege_type = "OBJECT"
  privilege      = "SELECT"
  object_type    = "TABLE"
  object_name    = "MYSCHEMA.MYTABLE"
}
```

**NEW (exasol_object_privilege):**
```hcl
resource "exasol_object_privilege" "select_table" {
  grantee     = "ANALYST_ROLE"
  privilege   = "SELECT"
  object_type = "TABLE"
  object_name = "MYSCHEMA.MYTABLE"
}
```

**Changes:**
- `grantee_name` → `grantee`
- Removed `privilege_type`

---

### Example 4: Role Grant (Most Confusing!)

**OLD (exasol_grant) - Method 1:**
```hcl
resource "exasol_grant" "role_to_user" {
  grantee_name      = "TEST_USER"
  privilege_type    = "SYSTEM"  # ← Confusing!
  privilege         = "TEST_ROLE"
  object_type       = "ROLE"    # ← Why is ROLE an object_type?
  object_name       = "TEST_ROLE"
  with_admin_option = false
}
```

**OLD (exasol_grant) - Method 2:**
```hcl
resource "exasol_grant" "role_to_user" {
  grantee_name      = "TEST_USER"
  privilege_type    = "OBJECT"  # ← Also confusing!
  privilege         = "TEST_ROLE"
  object_type       = "ROLE"
  object_name       = "TEST_ROLE"
}
```

**NEW (exasol_role_grant) - Clear and Simple:**
```hcl
resource "exasol_role_grant" "role_to_user" {
  role    = "TEST_ROLE"
  grantee = "TEST_USER"
  # with_admin_option defaults to false
}
```

**Changes:**
- Completely new, purpose-built resource
- `privilege` → `role` (more accurate naming)
- `grantee_name` → `grantee`
- Removed confusing `privilege_type` and `object_type`

---

## Migration Steps

### Step 1: Update your Terraform configuration

Replace all `exasol_grant` resources with the appropriate new resource type.

### Step 2: Update state (if needed)

If you're migrating existing infrastructure, you have two options:

#### Option A: Let Terraform recreate the grants (Recommended)

This is the simplest approach:

1. Update your configuration files
2. Run `terraform plan` - you'll see the old grants being destroyed and new ones created
3. Run `terraform apply` - grants will be momentarily revoked and re-granted

**Pros:** Simple, clean state
**Cons:** Brief moment where grants don't exist (usually fine for roles/privileges)

#### Option B: Manual state migration (Advanced)

To avoid any downtime, you can manually update the Terraform state:

```bash
# Remove old grant from state
terraform state rm exasol_grant.role_to_user

# Import the same grant with new resource type
terraform import exasol_role_grant.role_to_user "TEST_ROLE|TEST_USER|false"
```

### Step 3: Verify

Run `terraform plan` to ensure no unexpected changes.

## Complete Before/After Example

**OLD Configuration:**
```hcl
resource "exasol_user" "analyst" {
  name      = "ANALYST_USER"
  auth_type = "PASSWORD"
  password  = "SecurePass123"
}

resource "exasol_role" "analyst_role" {
  name = "ANALYST_ROLE"
}

# System privilege
resource "exasol_grant" "create_session" {
  grantee_name      = exasol_user.analyst.name
  privilege_type    = "SYSTEM"
  privilege         = "CREATE SESSION"
}

# Object privilege
resource "exasol_grant" "schema_usage" {
  grantee_name   = exasol_role.analyst_role.name
  privilege_type = "OBJECT"
  privilege      = "USAGE"
  object_type    = "SCHEMA"
  object_name    = "ANALYTICS"
}

# Role grant
resource "exasol_grant" "role_assignment" {
  grantee_name   = exasol_user.analyst.name
  privilege_type = "SYSTEM"
  privilege      = exasol_role.analyst_role.name
  object_type    = "ROLE"
  object_name    = exasol_role.analyst_role.name
}
```

**NEW Configuration:**
```hcl
resource "exasol_user" "analyst" {
  name      = "ANALYST_USER"
  auth_type = "PASSWORD"
  password  = "SecurePass123"
}

resource "exasol_role" "analyst_role" {
  name = "ANALYST_ROLE"
}

# System privilege - Clear and concise
resource "exasol_system_privilege" "create_session" {
  grantee   = exasol_user.analyst.name
  privilege = "CREATE SESSION"
}

# Object privilege - Self-documenting
resource "exasol_object_privilege" "schema_usage" {
  grantee     = exasol_role.analyst_role.name
  privilege   = "USAGE"
  object_type = "SCHEMA"
  object_name = "ANALYTICS"
}

# Role grant - Obvious what it does
resource "exasol_role_grant" "role_assignment" {
  role    = exasol_role.analyst_role.name
  grantee = exasol_user.analyst.name
}
```

## Attribute Mapping Reference

### System Privileges

| Old (exasol_grant) | New (exasol_system_privilege) |
|-------------------|-------------------------------|
| `grantee_name` | `grantee` |
| `privilege_type = "SYSTEM"` | *(removed, implicit)* |
| `privilege` | `privilege` |
| `with_admin_option` | `with_admin_option` |

### Object Privileges

| Old (exasol_grant) | New (exasol_object_privilege) |
|-------------------|-------------------------------|
| `grantee_name` | `grantee` |
| `privilege_type = "OBJECT"` | *(removed, implicit)* |
| `privilege` | `privilege` |
| `object_type` | `object_type` |
| `object_name` | `object_name` |

### Role Grants

| Old (exasol_grant) | New (exasol_role_grant) |
|-------------------|------------------------|
| `grantee_name` | `grantee` |
| `privilege_type = "SYSTEM"/"OBJECT"` | *(removed, implicit)* |
| `privilege` | `role` |
| `object_type = "ROLE"` | *(removed, implicit)* |
| `object_name` | *(removed, redundant)* |
| `with_admin_option` | `with_admin_option` |

## Import Format

If you need to import existing grants:

```bash
# System privilege: GRANTEE|PRIVILEGE|ADMIN_OPTION
terraform import exasol_system_privilege.example "ANALYST_ROLE|CREATE SESSION|false"

# Object privilege: GRANTEE|PRIVILEGE|OBJECT_TYPE|OBJECT_NAME
terraform import exasol_object_privilege.example "ANALYST_ROLE|SELECT|TABLE|MYSCHEMA.MYTABLE"

# Role grant: ROLE|GRANTEE|ADMIN_OPTION
terraform import exasol_role_grant.example "ANALYST_ROLE|JOHN_DOE|false"
```

## Benefits of New Resources

✅ **Self-documenting** - Resource type makes intent clear
✅ **Simpler** - Fewer attributes, less confusion
✅ **Type-safe** - Can't mix incompatible attribute combinations
✅ **Better validation** - Resource-specific validation rules
✅ **Clearer errors** - Error messages are context-specific
✅ **Easier to read** - Anyone can understand your Terraform code

## Questions?

For examples, see:
- [examples/privileges/main.tf](examples/privileges/main.tf) - Comprehensive examples of all three resource types
- [README.md](README.md) - Quick start guide