package resources

import (
	"fmt"
	"strings"

	_ "github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs"
)

func qualify(obj string) string {
	// Allow user to pass SCHEMA.TABLE or just SCHEMA.
	// We quote identifiers but keep dots as separators.
	parts := strings.Split(obj, ".")
	for i, p := range parts {
		parts[i] = fmt.Sprintf(`"%s"`, strings.Trim(p, `"`))
	}
	return strings.Join(parts, ".")
}
