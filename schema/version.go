package schema

import (
	"fmt"
	"regexp"

	"github.com/Masterminds/semver"
)

// Version represents the schema version of an object
type Version string

// Validate the provided schema version is present and adheres
// to semantic versioning
func (v Version) Validate() error {
	version := string(v)

	_, err := semver.NewVersion(version)
	if err != nil {
		return fmt.Errorf("invalid schema version %q: %v", version, err)
	}
	return nil
}

// GetSemver returns a proper semver Version from the provided string,
// trimming the non-semver prefix according to schema versioning protocol in the
// cnabio/cnab-spec repo
func GetSemver(schemaVersion string) (Version, error) {
	r := regexp.MustCompile("^cnab-[a-z]+-(.*)")
	match := r.FindStringSubmatch(schemaVersion)
	if len(match) < 2 {
		return "", fmt.Errorf("no semver submatch for schemaVersion %q using regex %q", schemaVersion, r)
	}

	version := Version(match[1])
	err := version.Validate()
	if err != nil {
		return "", err
	}

	return version, nil
}
