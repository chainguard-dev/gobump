package update

import (
	"fmt"
	"os"
	"os/exec"
	"path"

	"golang.org/x/mod/modfile"
	"golang.org/x/mod/semver"

	"github.com/chainguard-dev/gobump/pkg/types"
)

func DoUpdate(pkgVersions []*types.Package, replaces []string, modroot string) (*modfile.File, error) {
	modpath := path.Join(modroot, "go.mod")
	modFileContent, err := os.ReadFile(modpath)
	if err != nil {
		return nil, fmt.Errorf("error reading go.mod: %w", err)
	}

	modFile, err := modfile.Parse("go.mod", modFileContent, nil)
	if err != nil {
		return nil, fmt.Errorf("error parsing go.mod: %w", err)
	}

	// Do replaces in the beginning
	for _, replace := range replaces {
		cmd := exec.Command("go", "mod", "edit", "-replace", replace)
		cmd.Dir = modroot
		if err := cmd.Run(); err != nil {
			return nil, fmt.Errorf("error running go mod edit -replace %s: %w", replace, err)
		}
	}

	for _, pkg := range pkgVersions {
		currentVersion := getVersion(modFile, pkg.Name)
		if currentVersion == "" {
			return nil, fmt.Errorf("package %s not found in go.mod", pkg.Name)
		}
		// Sometimes we request to pin to a specific commit.
		// In that case, skip the compare check.
		if semver.IsValid(pkg.Version) {
			if semver.Compare(currentVersion, pkg.Version) > 0 {
				return nil, fmt.Errorf("package %s is already at version %s", pkg.Name, pkg.Version)
			}
		} else {
			fmt.Printf("Requesting pin to %s\n. This is not a valid SemVer, so skipping version check.", pkg.Version)
		}

		if err := updatePackage(modFile, pkg.Name, pkg.Version, modroot); err != nil {
			return nil, fmt.Errorf("error updating package: %w", err)
		}
	}

	// Read the entire go.mod one more time into memory and check that all the version constraints are met.
	newFileContent, err := os.ReadFile(modpath)
	if err != nil {
		return nil, fmt.Errorf("error reading go.mod: %w", err)
	}
	newModFile, err := modfile.Parse("go.mod", newFileContent, nil)
	if err != nil {
		return nil, fmt.Errorf("error parsing go.mod: %w", err)
	}
	for _, pkg := range pkgVersions {
		verStr := getVersion(newModFile, pkg.Name)
		if semver.Compare(verStr, pkg.Version) < 0 {
			return nil, fmt.Errorf("package %s is less than the desired version %s", pkg.Name, pkg.Version)
		}
	}

	return newModFile, nil
}

func updatePackage(modFile *modfile.File, name, version, modroot string) error {
	// Check if the package is replaced first
	for _, replace := range modFile.Replace {
		if replace.Old.Path == name {
			cmd := exec.Command("go", "mod", "edit", "-replace", fmt.Sprintf("%s=%s@%s", replace.Old.Path, name, version)) //nolint:gosec
			cmd.Dir = modroot
			return cmd.Run()
		}
	}

	// No replace, just update!
	cmd := exec.Command("go", "get", fmt.Sprintf("%s@%s", name, version)) //nolint:gosec
	cmd.Dir = modroot
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func getVersion(modFile *modfile.File, packageName string) string {
	// Handle package update, including 'replace' clause

	// Replace checks have to come first!
	for _, replace := range modFile.Replace {
		if replace.Old.Path == packageName {
			return replace.New.Version
		}
	}

	for _, req := range modFile.Require {
		if req.Mod.Path == packageName {
			return req.Mod.Version
		}
	}

	return ""
}
