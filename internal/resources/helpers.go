package resources

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

func exec(ctx context.Context, db *sql.DB, sqlText string, args ...any) error {
	_, err := db.ExecContext(ctx, sqlText, args...)
	return err
}

func qualify(obj string) string {
	// Allow user to pass SCHEMA.TABLE or just SCHEMA.
	// We quote identifiers but keep dots as separators.
	parts := strings.Split(obj, ".")
	for i, p := range parts {
		parts[i] = fmt.Sprintf(`"%s"`, strings.Trim(p, `"`))
	}
	return strings.Join(parts, ".")
}
