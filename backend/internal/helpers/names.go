package helpers

import (
	"regexp"
	"strings"
)

// SanitizeResourceName converts a name to a valid resource name
// Requirements: lowercase, letters/numbers/hyphens only, start with letter, no trailing hyphen
func SanitizeResourceName(name string) string {
	// Convert to lowercase
	name = strings.ToLower(name)

	// Replace invalid characters with hyphens
	reg := regexp.MustCompile(`[^a-z0-9-]+`)
	name = reg.ReplaceAllString(name, "-")

	// Remove leading non-letter characters
	name = regexp.MustCompile(`^[^a-z]+`).ReplaceAllString(name, "")

	// Remove trailing hyphens
	name = strings.TrimRight(name, "-")

	// If empty after sanitization, use default
	if name == "" {
		name = "stolos"
	}

	return name
}
