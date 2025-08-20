// Package types defines types and parsing functions for gobump.
package types

import (
	"fmt"
	"io"
	"path/filepath"

	"os"

	"github.com/ghodss/yaml"
)

// ParseFile parses a YAML file containing package update specifications.
func ParseFile(bumpFile string) (map[string]*Package, error) {
	if bumpFile == "" {
		return nil, fmt.Errorf("no filename specified")
	}
	bumpFile = filepath.Clean(bumpFile)
	var pkgVersions map[string]*Package
	var packageList PackageList
	file, err := os.Open(bumpFile) //nolint:gosec
	if err != nil {
		return nil, fmt.Errorf("failed reading file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			// Log error if needed, but we're already in defer
			_ = err
		}
	}()
	bytes, _ := io.ReadAll(file)
	if err := yaml.Unmarshal(bytes, &packageList); err != nil {
		return nil, fmt.Errorf("unmarshaling file: %w", err)
	}
	for i, p := range packageList.Packages {
		if p.Name == "" {
			return nil, fmt.Errorf("invalid package spec at [%d], missing name", i)
		}
		if p.Version == "" {
			return nil, fmt.Errorf("invalid package spec at [%d], missing version", i)
		}
		if pkgVersions == nil {
			pkgVersions = make(map[string]*Package, 1)
		}
		pkgVersions[p.Name] = &packageList.Packages[i]
		pkgVersions[p.Name].Index = i
	}
	return pkgVersions, nil
}
