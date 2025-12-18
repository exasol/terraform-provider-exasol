package resources

import (
	"regexp"
	"strings"
)

// isValidIdentifier validates Exasol identifiers.
// When using quoted identifiers (double quotes), Exasol allows any characters.
// The only restriction is that the identifier must not be empty.
// Double quotes within the identifier must be escaped by doubling them,
// which is handled by escapeIdentifierLiteral().
func isValidIdentifier(name string) bool {
	return name != ""
}

// sanitizeLogSQL redacts sensitive information (passwords) from SQL statements before logging.
// This prevents passwords from appearing in logs.
func sanitizeLogSQL(sql string) string {
	// Redact passwords in CREATE/ALTER USER statements
	// Pattern: IDENTIFIED BY "password" or IDENTIFIED BY 'password'
	re := regexp.MustCompile(`(?i)(IDENTIFIED\s+BY\s+)["']([^"']+)["']`)
	sanitized := re.ReplaceAllString(sql, `${1}"***REDACTED***"`)
	return sanitized
}

// escapeStringLiteral escapes single quotes in string literals for SQL.
// In SQL, single quotes are escaped by doubling them: ' becomes ”
func escapeStringLiteral(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

// escapeIdentifierLiteral escapes double quotes in identifier literals for SQL.
// In SQL, double quotes within quoted identifiers are escaped by doubling them: " becomes ""
func escapeIdentifierLiteral(s string) string {
	return strings.ReplaceAll(s, `"`, `""`)
}
