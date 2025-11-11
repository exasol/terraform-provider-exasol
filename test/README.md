# Test Directory

This directory contains comprehensive test configurations for the Exasol Terraform Provider.

## Prerequisites

1. Exasol Docker container running:
   ```bash
   docker ps | grep exasol
   # If not running:
   docker run -d -p 8563:8563 --name exasol exasol/docker-db:latest
   ```

2. Provider built and installed locally:
   ```bash
   make install-local
   ```

## Test Organization

### Automated Test Runner

The comprehensive test suite can be run using the automated test runner:

```bash
./run-tests.sh
```

This will run all test suites and verify no drift occurs after apply.

### Test Suites

#### Suite 1: Role Grants (suite-1-role-grants/)
**Tests**: TC-RG-001 through TC-RG-007
**Focus**: Admin option handling, state transitions, case sensitivity
**Coverage**:
- Role grants without admin option (no drift)
- Role grants with admin option
- Role-to-role hierarchical grants
- Case insensitivity testing

#### Suite 2: Object Privileges (suite-2-object-privileges/)
**Tests**: TC-OP-001 through TC-OP-008
**Focus**: Privilege list ordering independence, multiple privileges
**Coverage**:
- Single privilege grants
- Multiple privilege grants
- ALL privilege handling
- Privilege ordering independence

#### Suite 3: System Privileges (suite-3-system-privileges/)
**Tests**: TC-SP-001 through TC-SP-006
**Focus**: System-level privileges with admin options
**Coverage**:
- Basic DDL privileges (CREATE TABLE, CREATE SCHEMA)
- Admin option delegation
- Permission escalation privileges (GRANT ANY PRIVILEGE)
- ETL pipeline privileges (IMPORT, EXPORT)

#### Suite 4: Connection Grants (suite-4-connection-grants/)
**Tests**: TC-CG-001 through TC-CG-004
**Focus**: Connection access grants
**Coverage**:
- Direct user connection grants
- Role-based connection grants
- Multiple connection access
- Connection workflow patterns

#### Suite 5: Real-World Production Setup (suite-5-real-world/)
**Tests**: TC-RW-001
**Focus**: Production module replication
**Coverage**:
- 4-layer data pipeline (RAW → SNAP → STG → MART)
- Aggregative role patterns
- Cross-layer grants with admin options
- Technical users (ETL, BI)
- Connection grants workflow

### Legacy Tests

#### test-admin-option.tf
Original admin option drift test - now superseded by suite-1-role-grants

#### admin-transitions/
State transition tests for admin_option changes

## Running Individual Test Suites

### Quick Test (Single Suite)

```bash
cd suite-1-role-grants
terraform init
terraform apply -auto-approve
terraform plan  # Should show "No changes"
terraform destroy -auto-approve
```

Expected result: Second `terraform plan` should show **No changes**.

## Connection Configuration

The test uses `validate_server_certificate = false` to bypass TLS certificate validation errors with local Docker containers.

## Known Issues

### Transaction Collision During Destroy (FIXED)

~~You may occasionally see transaction collision errors during `terraform destroy`.~~

**Status**: FIXED - The provider now serializes all delete operations using a global mutex to prevent transaction collisions.

**Future improvement**: The current implementation uses a mutex which makes deletes slower. A better solution would be retry logic with exponential backoff. See `TODO.md` in the project root for details.

## Test Documentation

Comprehensive documentation is available in this directory:

- **TEST_DOCUMENTATION_INDEX.md** - Navigation guide for all documentation
- **COMPREHENSIVE_TEST_PATTERNS.md** - Detailed test cases with examples (50+ tests)
- **QUICK_TEST_REFERENCE.md** - One-page quick reference for developers
- **TEST_PATTERNS_VISUAL_SUMMARY.txt** - High-level visual overview

## Test Coverage Summary

- **50+ test cases** across 4 grant resource types
- **4 privilege layers** (RAW, SNAP, STG, MART) in production setup
- **7 critical edge cases** covered (admin option null vs false, privilege ordering, etc.)
- **Multiple test execution modes** (automated suite runner, individual tests, manual transitions)

### Coverage by Resource Type

| Resource Type | Test Suites | Key Focus Areas |
|---|---|---|
| exasol_role_grant | Suite 1, Suite 5 | Admin option transitions, hierarchical roles |
| exasol_system_privilege | Suite 3, Suite 5 | Admin option handling, ETL privileges |
| exasol_object_privilege | Suite 2, Suite 5 | Privilege ordering, multiple privileges |
| exasol_connection_grant | Suite 4, Suite 5 | User/role patterns, workflow testing |

## Cleaning Up

To reset the test environment:

```bash
cd test
rm -rf .terraform .terraform.lock.hcl terraform.tfstate*
```

To clean a specific test suite:

```bash
cd suite-1-role-grants
terraform destroy -auto-approve
rm -rf .terraform .terraform.lock.hcl terraform.tfstate*
```
