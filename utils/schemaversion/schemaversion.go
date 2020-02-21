package schemaversion

import (
	"fmt"

	"github.com/Masterminds/semver"
)

// SchemaVersion represents the schema version of an object
type SchemaVersion string

// Validate the provided schema version is present and adheres
// to semantic versioning
func (v SchemaVersion) Validate() error {
	version := string(v)

	_, err := semver.NewVersion(version)
	if err != nil {
		return fmt.Errorf("invalid schema version %q: %v", version, err)
	}
	return nil
}
