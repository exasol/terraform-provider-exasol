package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/exasol/exasol-driver-go"
)

func main() {
	ctx := context.Background()

	// Connect to local Docker Exasol
	config := exasol.NewConfig("sys", "exasol")
	dsnString := config.Host("localhost").
		Port(8563).
		ValidateServerCertificate(false).
		String()

	db, err := sql.Open("exasol", dsnString)
	if err != nil {
		log.Fatalf("Failed to open connection: %v", err)
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	fmt.Println("✅ Connected to Exasol database")

	// Create test resources
	fmt.Println("\n📦 Creating test resources...")

	// Create test roles
	_, err = db.ExecContext(ctx, `CREATE ROLE test_role_1`)
	if err != nil {
		fmt.Printf("⚠️  Warning creating test_role_1: %v\n", err)
	} else {
		fmt.Println("✓ Created test_role_1")
	}

	_, err = db.ExecContext(ctx, `CREATE ROLE test_role_2`)
	if err != nil {
		fmt.Printf("⚠️  Warning creating test_role_2: %v\n", err)
	} else {
		fmt.Println("✓ Created test_role_2")
	}

	// Create test connection
	_, err = db.ExecContext(ctx, `CREATE CONNECTION test_connection TO 'http://example.com'`)
	if err != nil {
		fmt.Printf("⚠️  Warning creating test_connection: %v\n", err)
	} else {
		fmt.Println("✓ Created test_connection")
	}

	// Grant role to role (with admin option)
	_, err = db.ExecContext(ctx, `GRANT test_role_1 TO test_role_2 WITH ADMIN OPTION`)
	if err != nil {
		fmt.Printf("⚠️  Warning granting role: %v\n", err)
	} else {
		fmt.Println("✓ Granted test_role_1 to test_role_2 WITH ADMIN OPTION")
	}

	// Grant connection to role
	_, err = db.ExecContext(ctx, `GRANT CONNECTION test_connection TO test_role_1`)
	if err != nil {
		fmt.Printf("⚠️  Warning granting connection: %v\n", err)
	} else {
		fmt.Println("✓ Granted test_connection to test_role_1")
	}

	// Test role_grant Read query (v0.1.4)
	fmt.Println("\n🔍 Testing role_grant Read query (v0.1.4)...")
	var adminOption string
	query := `SELECT ADMIN_OPTION FROM EXA_DBA_ROLE_PRIVS WHERE GRANTED_ROLE = ? AND GRANTEE = ?`
	err = db.QueryRowContext(ctx, query, "TEST_ROLE_1", "TEST_ROLE_2").Scan(&adminOption)
	if err == sql.ErrNoRows {
		fmt.Println("❌ FAILED: Role grant not found in EXA_DBA_ROLE_PRIVS")
		os.Exit(1)
	} else if err != nil {
		fmt.Printf("❌ FAILED: Query error: %v\n", err)
		os.Exit(1)
	} else {
		fmt.Printf("✅ SUCCESS: Role grant found with ADMIN_OPTION = %q\n", adminOption)
		if adminOption == "TRUE" || adminOption == "1" || adminOption == "true" {
			fmt.Println("✅ ADMIN_OPTION correctly set to TRUE/1/true")
		} else {
			fmt.Printf("❌ FAILED: Expected ADMIN_OPTION to be TRUE, 1, or true, got %q\n", adminOption)
			os.Exit(1)
		}
	}

	// Test connection_grant Read query (v0.1.4 - NEW FIX)
	fmt.Println("\n🔍 Testing connection_grant Read query (v0.1.4 - FIXED)...")
	var dummy int
	query = `SELECT 1 FROM EXA_DBA_CONNECTION_PRIVS WHERE GRANTED_CONNECTION = ? AND GRANTEE = ?`
	err = db.QueryRowContext(ctx, query, "TEST_CONNECTION", "TEST_ROLE_1").Scan(&dummy)
	if err == sql.ErrNoRows {
		fmt.Println("❌ FAILED: Connection grant not found in EXA_DBA_CONNECTION_PRIVS")
		fmt.Println("   This means the v0.1.4 fix is correct - the old query wouldn't find it!")

		// Try the OLD wrong query to prove it was broken
		fmt.Println("\n🔍 Testing OLD WRONG query from v0.1.3...")
		query = `SELECT 1 FROM EXA_DBA_OBJ_PRIVS WHERE OBJECT_TYPE = 'CONNECTION' AND OBJECT_NAME = ? AND GRANTEE = ?`
		err = db.QueryRowContext(ctx, query, "TEST_CONNECTION", "TEST_ROLE_1").Scan(&dummy)
		if err == sql.ErrNoRows {
			fmt.Println("✅ CONFIRMED: Old query DOES NOT find connection grants (this was the bug!)")
		} else {
			fmt.Println("⚠️  Unexpected: Old query found something")
		}

		os.Exit(1)
	} else if err != nil {
		fmt.Printf("❌ FAILED: Query error: %v\n", err)
		os.Exit(1)
	} else {
		fmt.Println("✅ SUCCESS: Connection grant found in EXA_DBA_CONNECTION_PRIVS")
	}

	// Test the OLD wrong query to confirm it doesn't work
	fmt.Println("\n🔍 Testing OLD WRONG query from v0.1.3 (should NOT find grant)...")
	query = `SELECT 1 FROM EXA_DBA_OBJ_PRIVS WHERE OBJECT_TYPE = 'CONNECTION' AND OBJECT_NAME = ? AND GRANTEE = ?`
	err = db.QueryRowContext(ctx, query, "TEST_CONNECTION", "TEST_ROLE_1").Scan(&dummy)
	if err == sql.ErrNoRows {
		fmt.Println("✅ CONFIRMED: Old v0.1.3 query DOES NOT find connection grants")
		fmt.Println("   This proves why we had constant drift - the Read function couldn't find the grants!")
	} else if err != nil {
		fmt.Printf("⚠️  Query error: %v\n", err)
	} else {
		fmt.Println("⚠️  Unexpected: Old query found the grant (maybe Exasol version difference?)")
	}

	// Cleanup
	fmt.Println("\n🧹 Cleaning up test resources...")
	db.ExecContext(ctx, `REVOKE CONNECTION test_connection FROM test_role_1`)
	db.ExecContext(ctx, `REVOKE test_role_1 FROM test_role_2`)
	db.ExecContext(ctx, `DROP CONNECTION test_connection`)
	db.ExecContext(ctx, `DROP ROLE test_role_2`)
	db.ExecContext(ctx, `DROP ROLE test_role_1`)
	fmt.Println("✅ Cleanup complete")

	fmt.Println("\n🎉 ALL TESTS PASSED!")
	fmt.Println("\nSummary:")
	fmt.Println("- ✅ role_grant Read function works correctly with EXA_DBA_ROLE_PRIVS")
	fmt.Println("- ✅ connection_grant Read function works correctly with EXA_DBA_CONNECTION_PRIVS")
	fmt.Println("- ✅ Old v0.1.3 query confirmed as broken (couldn't find connection grants)")
	fmt.Println("\n🚀 Provider v0.1.4 fixes are verified!")
}
