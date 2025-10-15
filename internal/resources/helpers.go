package resources

import (
	"fmt"
	"strings"
)

func qualify(obj string) string {
	// Allow user to pass SCHEMA.TABLE or just SCHEMA.
	// We quote identifiers but keep dots as separators.
	// Also validate each part to prevent SQL injection.
	parts := strings.Split(obj, ".")
	for i, p := range parts {
		// Remove existing quotes if any
		cleaned := strings.Trim(p, `"`)

		// Validate the identifier
		if !isValidIdentifier(cleaned) {
			// If validation fails, still escape it to prevent SQL injection
			// but don't panic - let the database return an error
			cleaned = escapeIdentifierLiteral(cleaned)
		}

		parts[i] = fmt.Sprintf(`"%s"`, cleaned)
	}
	return strings.Join(parts, ".")
}
