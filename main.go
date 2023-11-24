package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/mod/modfile"
	"golang.org/x/mod/semver"
)

var packagesFlag = flag.String("packages", "", "A comma-separated list of packages to update")
var modrootFlag = flag.String("modroot", "", "path to the go.mod root")

func main() {
	flag.Parse()

	if *packagesFlag == "" {
		fmt.Println("Usage: gobump -packages=<package@version>,...")
		os.Exit(1)
	}
	packages := strings.Split(*packagesFlag, ",")
	pkgVersions := []pkgVersion{}
	for _, pkg := range packages {
		parts := strings.Split(pkg, "@")
		if len(parts) != 2 {
			fmt.Println("Usage: gobump -packages=<package@version>,...")
			os.Exit(1)
		}
		pkgVersions = append(pkgVersions, pkgVersion{
			Name:    parts[0],
			Version: parts[1],
		})
	}

	if _, err := doUpdate(pkgVersions, *modrootFlag); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func doUpdate(pkgVersions []pkgVersion, modroot string) (*modfile.File, error) {
	modpath := path.Join(modroot, "go.mod")
	modFileContent, err := os.ReadFile(modpath)
	if err != nil {
		return nil, errors.Wrap(err, "error reading go.mod")
	}

	modFile, err := modfile.Parse("go.mod", modFileContent, nil)
	if err != nil {
		return nil, errors.Wrap(err, "error parsing go.mod")
	}

	// Step 3: Check and update packages
	for _, pkg := range pkgVersions {
		currentVersion := getVersion(modFile, pkg.Name)
		if currentVersion == "" {
			return nil, errors.Errorf("Package %s not found in go.mod\n", pkg.Name)
		}
		if semver.Compare(currentVersion, pkg.Version) < 0 {
			if err := updatePackage(pkg.Name, pkg.Version, modroot); err != nil {
				fmt.Printf("Error updating package: %s\n", err)
			}
		} else {
			fmt.Printf("Package %s is already at version %s\n", pkg.Name, pkg.Version)
		}
	}

	newFileContent, err := os.ReadFile(modpath)
	if err != nil {
		return nil, err
	}
	return modfile.Parse("go.mod", newFileContent, nil)
}

func updatePackage(name, version, modroot string) error {
	cmd := exec.Command("go", "get", fmt.Sprintf("%s@%s", name, version))
	cmd.Dir = modroot
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func getVersion(modFile *modfile.File, packageName string) string {
	// Handle package update, including 'replace' clause
	for _, req := range modFile.Require {
		if req.Mod.Path == packageName {
			return req.Mod.Version
		}
	}

	for _, replace := range modFile.Replace {
		if replace.Old.Path == packageName {
			return replace.New.Version
		}
	}
	return ""
}

type pkgVersion struct {
	Name    string
	Version string
}
