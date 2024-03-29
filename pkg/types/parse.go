package types

import (
	"fmt"
	"io"

	"os"

	"github.com/ghodss/yaml"
)

func ParseFile(bumpFile string) (map[string]*Package, error) {
	if bumpFile == "" {
		return nil, fmt.Errorf("no filename specified")
	}
	var pkgVersions map[string]*Package
	var packageList PackageList
	file, err := os.Open(bumpFile)
	if err != nil {
		return nil, fmt.Errorf("failed reading file: %w", err)
	}
	defer file.Close()
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
