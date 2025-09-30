package resources

import (
	"regexp"
	"strings"
)

// isValidIdentifier validates Exasol identifiers.
// Exasol identifiers must start with a letter and contain only letters, digits, and underscores.
// They are stored in uppercase in the database.
func isValidIdentifier(name string) bool {
	if name == "" {
		return false
	}
	// Exasol identifier pattern: must start with A-Z, followed by A-Z, 0-9, or _
	// We check the uppercase version since Exasol stores identifiers in uppercase
	upName := strings.ToUpper(name)
	matched, _ := regexp.MatchString(`^[A-Z][A-Z0-9_]*$`, upName)
	return matched
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
// In SQL, single quotes are escaped by doubling them: ' becomes ''
func escapeStringLiteral(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

// escapeIdentifierLiteral escapes double quotes in identifier literals for SQL.
// In SQL, double quotes within quoted identifiers are escaped by doubling them: " becomes ""
func escapeIdentifierLiteral(s string) string {
	return strings.ReplaceAll(s, `"`, `""`)
}