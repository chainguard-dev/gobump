package update

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"sort"
	"strings"

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
	if _, ok := pkgVersions[modFile.Module.Mod.Path]; ok {
		return fmt.Errorf("bumping the main module is not allowed '%s'", modFile.Module.Mod.Path)
	}
	type pkgVersion struct {
		ReqVersion, AvailableVersion string
	}
	errorPkgVer := make(map[string]pkgVersion)

	// Detect if the list of packages contain any replace statement for the package, if so we might drop that replace with a new one.
	for _, replace := range modFile.Replace {
		if replace != nil {
			if _, ok := pkgVersions[replace.New.Path]; ok {
				// pkg is already been replaced
				pkgVersions[replace.New.Path].Replace = true
				// This happens when we found a replace in the go mod for a dependency that we defined in deps.
				// We need to drop that replace, so we need to set the name of the old path to use the existing one in the go.mod.
				if pkgVersions[replace.New.Path].OldName == "" {
					pkgVersions[replace.New.Path].OldName = replace.Old.Path
				}
				if semver.IsValid(pkgVersions[replace.New.Path].Version) {
					if semver.Compare(replace.New.Version, pkgVersions[replace.New.Path].Version) > 0 {
						errorPkgVer[replace.New.Path] = pkgVersion{
							ReqVersion:       pkgVersions[replace.New.Path].Version,
							AvailableVersion: replace.New.Version,
						}
						continue
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
						// Already present, check if the version is smaller or not
						if existingPkg, exists := errorPkgVer[require.Mod.Path]; exists {
							if semver.Compare(require.Mod.Version, existingPkg.AvailableVersion) > 0 {
								errorPkgVer[require.Mod.Path] = pkgVersion{
									ReqVersion:       pkgVersions[require.Mod.Path].Version, // Requested version stays the same
									AvailableVersion: require.Mod.Version,                   // Update to higher available version
								}
							}
						} else {
							// First time, add it to the map
							errorPkgVer[require.Mod.Path] = pkgVersion{
								ReqVersion:       pkgVersions[require.Mod.Path].Version,
								AvailableVersion: require.Mod.Version,
							}
						}
						continue
					}
				} else {
					fmt.Printf("Requesting pin to %s.\n This is not a valid SemVer, so skipping version check.\n", pkgVersions[require.Mod.Path].Version)
				}
			}
		}
	}

	if len(errorPkgVer) > 0 {
		var errorMsg strings.Builder
		errorMsg.WriteString("The following errors were found::\n")
		for pkg, ver := range errorPkgVer {
			errorMsg.WriteString(fmt.Sprintf("  - package %s: requested version '%s', is already at version '%s'\n", pkg, ver.ReqVersion, ver.AvailableVersion))
		}
		return fmt.Errorf("%s", errorMsg.String())
	}

	return nil
}

func DoUpdate(pkgVersions map[string]*types.Package, cfg *types.Config) (*modfile.File, error) {
	var err error
	goVersion := cfg.GoVersion
	if goVersion == "" {
		if goVersion, err = getGoVersionFromEnvironment(); err != nil {
			return nil, fmt.Errorf("failed to get the Go version from the local system: %v", err)
		}
	}

	// Run go mod tidy before
	if cfg.Tidy && !cfg.TidySkipInitial {
		output, err := run.GoModTidy(cfg.Modroot, goVersion, cfg.TidyCompat)
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

	depsBumpOrdered := orderPkgVersionsMap(pkgVersions)

	// Replace the packages first.
	for _, k := range depsBumpOrdered {
		pkg := pkgVersions[k]
		if pkg.Replace {
			log.Printf("Update package: %s\n", k)
			log.Println("Running go mod edit replace ...")
			if output, err := run.GoModEditReplaceModule(pkg.OldName, pkg.Name, pkg.Version, cfg.Modroot); err != nil {
				return nil, fmt.Errorf("failed to run 'go mod edit -replace': %v for package %s/%s@%s with output: %v", err, pkg.OldName, pkg.Name, pkg.Version, output)
			}
		}
	}
	// Bump the require or new get packages in the specified order.
	for _, k := range depsBumpOrdered {
		pkg := pkgVersions[k]
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
		output, err := run.GoModTidy(cfg.Modroot, goVersion, cfg.TidyCompat)
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
		if verStr != "" && semver.Compare(verStr, pkg.Version) < 0 {
			return nil, fmt.Errorf("package %s with %s is less than the desired version %s", pkg.Name, verStr, pkg.Version)
		}
		if verStr == "" {
			return nil, fmt.Errorf("package %s was not found on the go.mod file. Please remove the package or add it to the list of 'replaces'", pkg.Name)
		}
	}

	if cfg.ShowDiff {
		if diff := cmp.Diff(string(content), string(newContent)); diff != "" {
			fmt.Println(diff)
		}
	}

	if _, err := os.Stat(path.Join(cfg.Modroot, "vendor")); err == nil {
		output, err := run.GoVendor(cfg.Modroot)
		if err != nil {
			return nil, fmt.Errorf("failed to run 'go vendor': %v with output: %v", err, output)
		}
	}

	return newModFile, nil
}

func orderPkgVersionsMap(pkgVersions map[string]*types.Package) []string {
	depsBumpOrdered := make([]string, 0, len(pkgVersions))
	for repo := range pkgVersions {
		depsBumpOrdered = append(depsBumpOrdered, repo)
	}
	sort.SliceStable(depsBumpOrdered, func(i, j int) bool {
		return pkgVersions[depsBumpOrdered[i]].Index < pkgVersions[depsBumpOrdered[j]].Index
	})
	return depsBumpOrdered
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

// getGoVersionFromEnvironment returns the Go version from the local environment.
func getGoVersionFromEnvironment() (string, error) {
	cmd := exec.Command("go", "version")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("failed to execute 'go version': %v", err)
	}
	versionOutput := out.String()
	return parseGoVersionString(versionOutput)
}

// parseGoVersionString parses the output of `go version` command and extracts the Go version.
func parseGoVersionString(versionOutput string) (string, error) {
	parts := strings.Fields(versionOutput)
	if len(parts) < 3 || !strings.HasPrefix(parts[2], "go") {
		return "", fmt.Errorf("unexpected format of 'go version' output")
	}

	// Remove the "go" prefix from the version
	goVersion := strings.TrimPrefix(parts[2], "go")
	log.Println("Local Go version:", goVersion)
	return goVersion, nil
}
