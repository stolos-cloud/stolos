package helpers

import (
	"encoding/json"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/siderolabs/talos/pkg/machinery/config/configpatcher"
)

// RawJSON converts a value to *json.RawMessage for use in JSON patches
func RawJSON(v any) *json.RawMessage {
	b, _ := json.Marshal(v)
	rm := json.RawMessage(b)
	return &rm
}

// RemoveDiskSelector creates a JSON patch that removes the diskSelector field.
// This is needed because base configs have hardware-specific busPath that won't work on new nodes.
// Returns the patched config, or the original if removal fails (field might not exist).
func RemoveDiskSelector(configBytes []byte) []byte {
	removePatch := jsonpatch.Patch{
		jsonpatch.Operation{
			"op":   RawJSON("remove"),
			"path": RawJSON("/machine/install/diskSelector"),
		},
	}

	patched, err := configpatcher.JSON6902(configBytes, removePatch)
	if err != nil {
		// If removal fails, return original config
		return configBytes
	}

	return patched
}
