// Package run provides utilities for running go commands.
package run

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	versionutil "k8s.io/apimachinery/pkg/util/version"
)

// GoModTidy runs go mod tidy with the specified go version and compatibility settings.
func GoModTidy(modroot, goVersion, compat string) (string, error) {
	if goVersion == "" {
		cmd := exec.Command("go", "env", "GOVERSION")
		cmd.Stderr = os.Stderr
		out, err := cmd.Output()
		if err != nil {
			return "", fmt.Errorf("%v: %w", cmd, err)
		}
		goVersion = strings.TrimPrefix(strings.TrimSpace(string(out)), "go")

		v := versionutil.MustParseGeneric(goVersion)
		goVersion = fmt.Sprintf("%d.%d", v.Major(), v.Minor())
	}

	log.Printf("Running go mod tidy with go version '%s' ...\n", goVersion)
	args := []string{"mod", "tidy", "-go", goVersion}
	if compat != "" {
		log.Printf("Running go mod tidy with compat '%s' ...\n", compat)
		args = append(args, "-compat", compat)
	}

	cmd := exec.Command("go", args...) //nolint:gosec
	cmd.Dir = modroot
	if bytes, err := cmd.CombinedOutput(); err != nil {
		return strings.TrimSpace(string(bytes)), err
	}
	return "", nil
}

func findWorkspaceFile(dir string) (root string) {
	dir = filepath.Clean(dir)
	// Look for enclosing go.mod.
	for {
		f := filepath.Join(dir, "go.work")
		if fi, err := os.Stat(f); err == nil && !fi.IsDir() {
			return f
		}
		d := filepath.Dir(dir)
		if d == dir {
			break
		}
		dir = d
	}
	return ""
}

func findGoWork(modroot string) string {
	switch gowork := os.Getenv("GOWORK"); gowork {
	case "off":
		return ""
	case "", "auto":
		return findWorkspaceFile(modroot)
	default:
		return gowork
	}
}

// GoVendor runs go mod vendor or go work vendor depending on workspace configuration.
func GoVendor(dir string, forceWork bool) (string, error) {
	if forceWork || findGoWork(dir) != "" {
		log.Print("Running go work vendor...")
		cmd := exec.Command("go", "work", "vendor")
		if bytes, err := cmd.CombinedOutput(); err != nil {
			return strings.TrimSpace(string(bytes)), err
		}
	} else {
		log.Print("Running go mod vendor...")
		cmd := exec.Command("go", "mod", "vendor")
		if bytes, err := cmd.CombinedOutput(); err != nil {
			return strings.TrimSpace(string(bytes)), err
		}
	}

	return "", nil
}

// GoGetModule runs go get for a specific module and version.
func GoGetModule(name, version, modroot string) (string, error) {
	cmd := exec.Command("go", "get", fmt.Sprintf("%s@%s", name, version)) //nolint:gosec
	cmd.Dir = modroot
	if bytes, err := cmd.CombinedOutput(); err != nil {
		return strings.TrimSpace(string(bytes)), err
	}
	return "", nil
}

// GoModEditReplaceModule edits go.mod to replace one module with another.
func GoModEditReplaceModule(nameOld, nameNew, version, modroot string) (string, error) {
	cmd := exec.Command("go", "mod", "edit", "-dropreplace", nameOld) //nolint:gosec
	cmd.Dir = modroot
	if bytes, err := cmd.CombinedOutput(); err != nil {
		return strings.TrimSpace(string(bytes)), fmt.Errorf("error running go command to drop replace modules: %w", err)
	}

	cmd = exec.Command("go", "mod", "edit", "-replace", fmt.Sprintf("%s=%s@%s", nameOld, nameNew, version)) //nolint:gosec
	cmd.Dir = modroot
	if bytes, err := cmd.CombinedOutput(); err != nil {
		return strings.TrimSpace(string(bytes)), fmt.Errorf("error running go command to replace modules: %w", err)
	}
	return "", nil
}

// GoModEditDropRequireModule drops a require directive from go.mod.
func GoModEditDropRequireModule(name, modroot string) (string, error) {
	cmd := exec.Command("go", "mod", "edit", "-droprequire", name) //nolint:gosec
	cmd.Dir = modroot
	if bytes, err := cmd.CombinedOutput(); err != nil {
		return strings.TrimSpace(string(bytes)), err
	}

	return "", nil
}

// GoModEditRequireModule adds or updates a require directive in go.mod.
func GoModEditRequireModule(name, version, modroot string) (string, error) {
	if bytes, err := GoModEditDropRequireModule(name, modroot); err != nil {
		return strings.TrimSpace(string(bytes)), err
	}

	cmd := exec.Command("go", "mod", "edit", "-require", fmt.Sprintf("%s@%s", name, version)) //nolint:gosec
	cmd.Dir = modroot
	if bytes, err := cmd.CombinedOutput(); err != nil {
		return strings.TrimSpace(string(bytes)), err
	}
	return "", nil
}
