package update

import (
	"fmt"
	"log"
	"os"
	"path"

	"github.com/chainguard-dev/gobump/pkg/run"
	"github.com/chainguard-dev/gobump/pkg/types"
	"github.com/google/go-cmp/cmp"
	"golang.org/x/mod/modfile"
	"golang.org/x/mod/semver"
)

func ParseGoModfile(path string) (*modfile.File, []byte, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, content, err
	}
	mod, err := modfile.Parse("go.mod", content, nil)
	if err != nil {
		return nil, content, err
	}

	return mod, content, nil
}

func checkPackageValues(pkgVersions map[string]*types.Package, modFile *modfile.File) error {
	// Detect if the list of packages contain any replace statement for the package, if so we might drop that replace with a new one.
	for _, replace := range modFile.Replace {
		if replace != nil {
			if _, ok := pkgVersions[replace.New.Path]; ok {
				// pkg is already been replaced
				pkgVersions[replace.New.Path].Replace = true
				if semver.IsValid(pkgVersions[replace.New.Path].Version) {
					if semver.Compare(replace.New.Version, pkgVersions[replace.New.Path].Version) > 0 {
						return fmt.Errorf("package %s with version '%s' is already at version %s", replace.New.Path, replace.New.Version, pkgVersions[replace.New.Path].Version)
					}
				} else {
					fmt.Printf("Requesting pin to %s.\n This is not a valid SemVer, so skipping version check.\n", pkgVersions[replace.New.Path].Version)
				}
			}
		}
	}
	// Detect if the list of packages contain any require statement for the package, if so we might drop that require with a new one.
	for _, require := range modFile.Require {
		if require != nil {
			if _, ok := pkgVersions[require.Mod.Path]; ok {
				// pkg is already been required
				pkgVersions[require.Mod.Path].Require = true
				// Sometimes we request to pin to a specific commit.
				// In that case, skip the compare check.
				if semver.IsValid(pkgVersions[require.Mod.Path].Version) {
					if semver.Compare(require.Mod.Version, pkgVersions[require.Mod.Path].Version) > 0 {
						return fmt.Errorf("package %s with version '%s' is already at version %s", require.Mod.Path, require.Mod.Version, pkgVersions[require.Mod.Path].Version)
					}
				} else {
					fmt.Printf("Requesting pin to %s.\n This is not a valid SemVer, so skipping version check.\n", pkgVersions[require.Mod.Path].Version)
				}
			}
		}
	}

	return nil
}

func DoUpdate(pkgVersions map[string]*types.Package, cfg *types.Config) (*modfile.File, error) {
	// Run go mod tidy before
	if cfg.Tidy {
		output, err := run.GoModTidy(cfg.Modroot, cfg.GoVersion)
		if err != nil {
			return nil, fmt.Errorf("failed to run 'go mod tidy': %v with output: %v", err, output)
		}
	}

	// Read the entire go.mod one more time into memory and check that all the version constraints are met.
	modpath := path.Join(cfg.Modroot, "go.mod")
	modFile, content, err := ParseGoModfile(modpath)
	if err != nil {
		return nil, fmt.Errorf("unable to parse the go mod file with error: %v", err)
	}

	// Detect require/replace modules and validate the version values
	err = checkPackageValues(pkgVersions, modFile)
	if err != nil {
		return nil, err
	}

	// Replace the packages first.
	for k, pkg := range pkgVersions {
		if pkg.Replace {
			log.Printf("Update package: %s\n", k)
			log.Println("Running go mod edit replace ...")
			if output, err := run.GoModEditReplaceModule(pkg.OldName, pkg.Name, pkg.Version, cfg.Modroot); err != nil {
				return nil, fmt.Errorf("failed to run 'go mod edit -replace': %v with output: %v", err, output)
			}
		}
	}
	// Bump the require or new get packages.
	for k, pkg := range pkgVersions {
		// Skip the replace that have been updated above
		if !pkg.Replace {
			log.Printf("Update package: %s\n", k)
			if pkg.Require {
				log.Println("Running go mod edit -droprequire ...")
				if output, err := run.GoModEditDropRequireModule(pkg.Name, cfg.Modroot); err != nil {
					return nil, fmt.Errorf("failed to run 'go mod edit -droprequire': %v with output: %v", err, output)
				}
			}
			log.Println("Running go get ...")
			if output, err := run.GoGetModule(pkg.Name, pkg.Version, cfg.Modroot); err != nil {
				return nil, fmt.Errorf("failed to run 'go get': %v with output: %v", err, output)
			}
		}
	}

	// Run go mod tidy
	if cfg.Tidy {
		output, err := run.GoModTidy(cfg.Modroot, cfg.GoVersion)
		if err != nil {
			return nil, fmt.Errorf("failed to run 'go mod tidy': %v with output: %v", err, output)
		}
	}

	// Read the entire go.mod one more time into memory and check that all the version constraints are met.
	newModFile, newContent, err := ParseGoModfile(modpath)
	if err != nil {
		return nil, fmt.Errorf("unable to parse the go mod file with error: %v", err)
	}
	for _, pkg := range pkgVersions {
		verStr := getVersion(newModFile, pkg.Name)
		if semver.Compare(verStr, pkg.Version) < 0 {
			return nil, fmt.Errorf("package %s with %s is less than the desired version %s", pkg.Name, verStr, pkg.Version)
		}
	}

	if cfg.ShowDiff {
		if diff := cmp.Diff(string(content), string(newContent)); diff != "" {
			fmt.Println(diff)
		}
	}

	return newModFile, nil
}

func getVersion(modFile *modfile.File, packageName string) string {
	// Handle package update, including 'replace' clause

	// Replace checks have to come first!
	for _, replace := range modFile.Replace {
		// Check if there is a new
		if replace.New.Path == packageName {
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
