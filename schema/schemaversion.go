package schema

import (
	"fmt"
	"regexp"

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

// GetSemverSchemaVersion returns a SchemaVersion from the provided string,
// trimming the non-semver prefix according to schema versioning in the
// cnabio/cnab-spec repo
func GetSemverSchemaVersion(schemaVersion string) (SchemaVersion, error) {
	r := regexp.MustCompile("^cnab-[a-z]+-(.*)")
	match := r.FindStringSubmatch(schemaVersion)
	if len(match) < 2 {
		return "", fmt.Errorf("no semver submatch for schemaVersion %q using regex %q", schemaVersion, r)
	}

	return SchemaVersion(match[1]), nil
}
